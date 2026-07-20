package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppPort        string
	DatabaseURL    string
	JWTSecret      string
	JWTExpiry      time.Duration
	IdempotencyTTL time.Duration
	Env            string
}

func Load() *Config {
	return &Config{
		AppPort:        getEnv("APP_PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/taskdb?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", "change-me-in-production-please"),
		JWTExpiry:      getDurationEnv("JWT_EXPIRY_HOURS", 24) * time.Hour,
		IdempotencyTTL: getDurationEnv("IDEMPOTENCY_TTL_HOURS", 24) * time.Hour,
		Env:            getEnv("APP_ENV", "development"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDurationEnv(key string, fallbackHours int) time.Duration {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n)
		}
	}
	return time.Duration(fallbackHours)
}
