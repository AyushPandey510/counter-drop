package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"counter-drop/api/internal/config"
	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/httpapi"
	"counter-drop/api/internal/realtime"
	"counter-drop/api/internal/storage"
	"counter-drop/api/internal/store"
	"counter-drop/api/internal/tasks"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	if err := run(cfg, logger); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.DatabaseURL == "" {
		return errors.New("CD_DATABASE_URL is required (start Postgres with: docker compose -f deploy/docker-compose.yml up -d postgres)")
	}
	policy := domain.Policy{UndoWindow: cfg.UndoWindow, AbandonAfter: cfg.AbandonAfter}
	st, err := store.New(ctx, cfg.DatabaseURL, policy)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.ApplyMigrations(ctx, cfg.MigrationsDir); err != nil {
		return err
	}
	if cfg.DemoSeed {
		if err := st.SeedDemo(ctx); err != nil {
			return err
		}
		logger.Info("demo shop ready", "slug", "demo-print", "staff", "Owner (PIN 1234), Kavita (PIN 1111)")
	}

	objects, local, err := buildStorage(ctx, cfg, logger)
	if err != nil {
		return err
	}

	hub := realtime.NewHub()
	srv := &httpapi.Server{Store: st, Objects: objects, Local: local, Hub: hub, Logger: logger, Cfg: cfg}

	deleter := &tasks.Deleter{Store: st, Objects: objects, Hub: hub, Logger: logger, Interval: cfg.DeletionInterval}
	go deleter.Run(ctx)

	httpServer := &http.Server{Addr: cfg.Addr, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdown)
	}()
	logger.Info("counter drop api listening", "addr", cfg.Addr, "env", cfg.Env)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func buildStorage(ctx context.Context, cfg config.Config, logger *slog.Logger) (storage.ObjectStore, *storage.LocalStore, error) {
	dur := storage.Durations{PutTTL: cfg.Storage.PutTTL, GetTTL: cfg.Storage.GetTTL}
	if cfg.Storage.S3Enabled() {
		s3, err := storage.NewS3Store(ctx, storage.S3Config{
			Endpoint: cfg.Storage.Endpoint, Region: cfg.Storage.Region, Bucket: cfg.Storage.Bucket,
			AccessKey: cfg.Storage.AccessKey, SecretKey: cfg.Storage.SecretKey, Durations: dur,
		})
		if err != nil {
			return nil, nil, err
		}
		logger.Info("storage: S3-compatible bucket", "bucket", cfg.Storage.Bucket)
		return s3, nil, nil
	}
	key := cfg.Storage.SigningKey
	if key == "" {
		if cfg.Env == "prod" {
			return nil, nil, errors.New("CD_STORAGE_SIGNING_KEY is required for local storage in prod")
		}
		key = store.NewSecret()
		logger.Warn("CD_STORAGE_SIGNING_KEY not set; using a random key (upload links reset on restart)")
	}
	local, err := storage.NewLocalStore(cfg.Storage.Dir, cfg.PublicAPIURL, []byte(key), dur)
	if err != nil {
		return nil, nil, err
	}
	logger.Info("storage: local disk", "dir", cfg.Storage.Dir, "public_api_url", cfg.PublicAPIURL)
	return local, local, nil
}
