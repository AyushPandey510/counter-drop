package config

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr        string
	DatabaseURL string
	LogLevel    slog.Level
	Storage     StorageConfig
}

type StorageConfig struct {
	Endpoint      string
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	PublicBaseURL string
}

func Load() Config {
	return Config{
		Addr:        env("CD_API_ADDR", ":8080"),
		DatabaseURL: env("CD_DATABASE_URL", ""),
		LogLevel:    parseLogLevel(env("CD_LOG_LEVEL", "info")),
		Storage: StorageConfig{
			Endpoint:      env("CD_STORAGE_ENDPOINT", ""),
			Region:        env("CD_STORAGE_REGION", "auto"),
			Bucket:        env("CD_STORAGE_BUCKET", ""),
			AccessKey:     env("CD_STORAGE_ACCESS_KEY", ""),
			SecretKey:     env("CD_STORAGE_SECRET_KEY", ""),
			PublicBaseURL: env("CD_PUBLIC_BASE_URL", ""),
		},
	}
}

func (c StorageConfig) Enabled() bool {
	return c.Bucket != "" && c.AccessKey != "" && c.SecretKey != ""
}

func (c StorageConfig) PresignDuration() time.Duration {
	return 15 * time.Minute
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
