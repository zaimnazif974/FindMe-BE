package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv            string
	AppPort           string
	DatabaseURL       string
	RedisAddr         string
	RedisPassword     string
	JWTSecret         string
	JWTExpiresIn      time.Duration
	SpacesRegion      string
	SpacesEndpoint    string
	SpacesAccessKey   string
	SpacesSecretKey   string
	SpacesBucket      string
	PresignedURLTTL   time.Duration
	CORSAllowedOrigin []string
}

func Load() (Config, error) {
	jwtTTL, err := time.ParseDuration(env("JWT_EXPIRES_IN", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JWT_EXPIRES_IN: %w", err)
	}
	minutes, err := strconv.Atoi(env("SPACES_PRESIGNED_URL_EXPIRES_MINUTES", "15"))
	if err != nil || minutes < 1 {
		return Config{}, fmt.Errorf("SPACES_PRESIGNED_URL_EXPIRES_MINUTES must be a positive integer")
	}
	spacesRegion := env("SPACES_REGION", "sgp1")
	cfg := Config{
		AppEnv:            env("APP_ENV", "development"),
		AppPort:           env("APP_PORT", "8080"),
		DatabaseURL:       env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/findme?sslmode=disable"),
		RedisAddr:         env("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		JWTExpiresIn:      jwtTTL,
		SpacesRegion:      spacesRegion,
		SpacesEndpoint:    env("SPACES_ENDPOINT", "https://"+spacesRegion+".digitaloceanspaces.com"),
		SpacesAccessKey:   os.Getenv("SPACES_ACCESS_KEY_ID"),
		SpacesSecretKey:   os.Getenv("SPACES_SECRET_ACCESS_KEY"),
		SpacesBucket:      os.Getenv("SPACES_BUCKET"),
		PresignedURLTTL:   time.Duration(minutes) * time.Minute,
		CORSAllowedOrigin: splitCSV(env("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
	}
	if len(cfg.JWTSecret) < 16 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
