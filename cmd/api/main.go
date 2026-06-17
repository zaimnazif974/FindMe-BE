package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"findme/backend/internal/config"
	"findme/backend/internal/database"
	"findme/backend/internal/handlers"
	"findme/backend/internal/middlewares"
	redisclient "findme/backend/internal/redis"
	"findme/backend/internal/repositories"
	"findme/backend/internal/services"
	"findme/backend/internal/storage"
	"findme/backend/internal/utils"
	ws "findme/backend/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	if err := database.Migrate(ctx, db, "migrations"); err != nil {
		log.Fatal(err)
	}

	redis, err := redisclient.Connect(ctx, cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = redis.Close() }()

	s3Client, err := storage.NewS3Client(ctx, cfg.AWSRegion)
	if err != nil {
		log.Fatal(err)
	}
	storageService := storage.NewService(s3Client, cfg.AWSS3Bucket, cfg.PresignedURLTTL)

	userRepo := repositories.NewUserRepository(db)
	userService := services.NewUserService(userRepo)
	authService := services.NewAuthService(userRepo, userService, cfg.JWTSecret, cfg.JWTExpiresIn)
	authHandler := handlers.NewAuthHandler(authService)

	groupRepo := repositories.NewGroupRepository(db)
	groupService := services.NewGroupService(groupRepo)
	groupHandler := handlers.NewGroupHandler(groupService)

	locationRepo := repositories.NewLocationRepository(db)
	locationService := services.NewLocationService(locationRepo, groupRepo, storageService)
	locationHandler := handlers.NewLocationHandler(locationService)

	hub := ws.NewHub()
	go hub.Run()
	wsHandler := ws.NewHandler(hub, groupRepo, cfg.JWTSecret, cfg.CORSAllowedOrigin)

	liveRepo := repositories.NewLiveLocationRepository(db)
	liveService := services.NewLiveLocationService(liveRepo, groupRepo, redis, hub)
	liveHandler := handlers.NewLiveLocationHandler(liveService)
	go liveService.RunExpirationWorker(ctx)

	memoryRepo := repositories.NewMemoryRepository(db)
	memoryService := services.NewMemoryService(memoryRepo, groupRepo, storageService)
	memoryHandler := handlers.NewMemoryHandler(memoryService)

	router := gin.New()
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Log only the path so a WebSocket JWT query parameter is never emitted.
		return param.TimeStamp.Format(time.RFC3339) + " " + param.Method + " " +
			param.Path + " " + param.StatusCodeColor() + http.StatusText(param.StatusCode) +
			param.ResetColor() + " " + param.Latency.String() + "\n"
	}), gin.Recovery(), middlewares.CORS(cfg.CORSAllowedOrigin))
	router.GET("/health", func(c *gin.Context) {
		utils.OK(c, http.StatusOK, gin.H{"status": "healthy"})
	})
	router.GET("/ws/groups/:groupId", wsHandler.Connect)

	api := router.Group("/api/v1")
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middlewares.Auth(cfg.JWTSecret))
	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/auth/me", authHandler.Me)
	protected.PATCH("/auth/me", authHandler.UpdateMe)

	protected.POST("/groups", groupHandler.Create)
	protected.GET("/groups", groupHandler.List)
	protected.POST("/groups/join", groupHandler.Join)
	protected.GET("/groups/:groupId", groupHandler.Get)
	protected.PATCH("/groups/:groupId", groupHandler.Update)
	protected.DELETE("/groups/:groupId", groupHandler.Delete)
	protected.POST("/groups/:groupId/leave", groupHandler.Leave)
	protected.GET("/groups/:groupId/members", groupHandler.Members)
	protected.DELETE("/groups/:groupId/members/:userId", groupHandler.RemoveMember)
	protected.POST("/groups/:groupId/invite-code/regenerate", groupHandler.RegenerateInviteCode)

	protected.POST("/locations/share", locationHandler.Share)
	protected.GET("/groups/:groupId/locations", locationHandler.List)
	protected.GET("/groups/:groupId/locations/latest", locationHandler.Latest)
	protected.POST("/locations/:locationShareId/photos", locationHandler.AddPhotos)

	live := protected.Group("/groups/:groupId/live-location")
	live.POST("/start", liveHandler.Start)
	live.POST("/update", middlewares.LiveLocationRateLimit(redis, 30, time.Minute), liveHandler.Update)
	live.POST("/stop", liveHandler.Stop)
	live.GET("/active", liveHandler.Active)

	protected.POST("/groups/:groupId/memory-points", memoryHandler.Create)
	protected.GET("/groups/:groupId/memory-points", memoryHandler.List)
	protected.GET("/memory-points/:memoryPointId", memoryHandler.Get)
	protected.PATCH("/memory-points/:memoryPointId", memoryHandler.Update)
	protected.DELETE("/memory-points/:memoryPointId", memoryHandler.Delete)
	protected.POST("/memory-points/:memoryPointId/ratings", memoryHandler.Rate)
	protected.POST("/memory-points/:memoryPointId/comments", memoryHandler.AddComment)
	protected.GET("/memory-points/:memoryPointId/comments", memoryHandler.Comments)
	protected.POST("/memory-points/:memoryPointId/photos", memoryHandler.AddPhotos)

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		log.Printf("FindMe API listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
	os.Exit(0)
}
