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
	AWSRegion         string
	AWSS3Bucket       string
	PresignedURLTTL   time.Duration
	CORSAllowedOrigin []string
}

func Load() (Config, error) {
	jwtTTL, err := time.ParseDuration(env("JWT_EXPIRES_IN", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JWT_EXPIRES_IN: %w", err)
	}
	minutes, err := strconv.Atoi(env("AWS_S3_PRESIGNED_URL_EXPIRES_MINUTES", "15"))
	if err != nil || minutes < 1 {
		return Config{}, fmt.Errorf("AWS_S3_PRESIGNED_URL_EXPIRES_MINUTES must be a positive integer")
	}
	cfg := Config{
		AppEnv:            env("APP_ENV", "development"),
		AppPort:           env("APP_PORT", "8080"),
		DatabaseURL:       env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/findme?sslmode=disable"),
		RedisAddr:         env("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		JWTExpiresIn:      jwtTTL,
		AWSRegion:         env("AWS_REGION", "ap-southeast-1"),
		AWSS3Bucket:       os.Getenv("AWS_S3_BUCKET"),
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
