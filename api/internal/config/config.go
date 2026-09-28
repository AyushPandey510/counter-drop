package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env           string // dev | staging | prod
	Addr          string
	PublicAPIURL  string // origin customers' phones use to reach the API (for local storage URLs)
	DatabaseURL   string
	LogLevel      slog.Level
	WebOrigins    []string // allowed CORS origins
	WebDir        string   // optional: serve the built PWA from this folder
	MigrationsDir string

	Storage StorageConfig

	UndoWindow       time.Duration
	AbandonAfter     time.Duration
	DeletionInterval time.Duration
	MaxFileBytes     int64
	MaxJobBytes      int64
	MaxFiles         int
	SessionTTL       time.Duration
	DemoSeed         bool
}

type StorageConfig struct {
	// S3 / R2
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	// Local disk (dev / single-box pilot)
	Dir        string
	SigningKey string

	PutTTL time.Duration
	GetTTL time.Duration
}

func (c StorageConfig) S3Enabled() bool {
	return c.Bucket != "" && c.AccessKey != "" && c.SecretKey != ""
}

func Load() Config {
	env := str("CD_ENV", "dev")
	return Config{
		Env:           env,
		Addr:          str("CD_API_ADDR", ":8080"),
		PublicAPIURL:  str("CD_PUBLIC_API_URL", "http://localhost:8080"),
		DatabaseURL:   str("CD_DATABASE_URL", ""),
		LogLevel:      parseLogLevel(str("CD_LOG_LEVEL", "info")),
		WebOrigins:    list("CD_WEB_ORIGINS", "http://localhost:5173"),
		WebDir:        str("CD_WEB_DIR", ""),
		MigrationsDir: str("CD_MIGRATIONS_DIR", "migrations"),
		Storage: StorageConfig{
			Endpoint:   str("CD_STORAGE_ENDPOINT", ""),
			Region:     str("CD_STORAGE_REGION", "auto"),
			Bucket:     str("CD_STORAGE_BUCKET", ""),
			AccessKey:  str("CD_STORAGE_ACCESS_KEY", ""),
			SecretKey:  str("CD_STORAGE_SECRET_KEY", ""),
			Dir:        str("CD_STORAGE_DIR", ".data/files"),
			SigningKey: str("CD_STORAGE_SIGNING_KEY", ""),
			PutTTL:     dur("CD_STORAGE_PUT_TTL", 15*time.Minute),
			GetTTL:     dur("CD_STORAGE_GET_TTL", 5*time.Minute),
		},
		UndoWindow:       dur("CD_UNDO_WINDOW", 10*time.Minute),
		AbandonAfter:     dur("CD_ABANDON_AFTER", 60*time.Minute),
		DeletionInterval: dur("CD_DELETION_INTERVAL", 30*time.Second),
		MaxFileBytes:     int64(num("CD_MAX_FILE_MB", 25)) << 20,
		MaxJobBytes:      int64(num("CD_MAX_JOB_MB", 50)) << 20,
		MaxFiles:         num("CD_MAX_FILES", 20),
		SessionTTL:       dur("CD_SESSION_TTL", 12*time.Hour),
		DemoSeed:         boolean("CD_DEMO_SEED", env == "dev"),
	}
}

func str(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func list(key, fallback string) []string {
	var out []string
	for _, p := range strings.Split(str(key, fallback), ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func dur(key string, fallback time.Duration) time.Duration {
	if d, err := time.ParseDuration(str(key, "")); err == nil {
		return d
	}
	return fallback
}

func num(key string, fallback int) int {
	if n, err := strconv.Atoi(str(key, "")); err == nil {
		return n
	}
	return fallback
}

func boolean(key string, fallback bool) bool {
	if b, err := strconv.ParseBool(str(key, "")); err == nil {
		return b
	}
	return fallback
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(value) {
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
