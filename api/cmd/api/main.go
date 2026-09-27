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
	"counter-drop/api/internal/httpapi"
	"counter-drop/api/internal/storage"
	"counter-drop/api/internal/store"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	ctx := context.Background()
	apiStore, cleanup := buildStore(ctx, cfg, logger)
	defer cleanup()

	uploadPresigner := buildUploadPresigner(ctx, cfg, logger)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.NewRouter(httpapi.RouterConfig{Store: apiStore, Uploads: uploadPresigner, Logger: logger}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("counter drop api listening", "addr", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-shutdownSignal.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("api server shutdown failed", "error", err)
		os.Exit(1)
	}
}

func buildStore(ctx context.Context, cfg config.Config, logger *slog.Logger) (httpapi.Store, func()) {
	if cfg.DatabaseURL == "" {
		memStore := store.NewMemoryStore()
		memStore.SeedDemoShop()
		logger.Warn("CD_DATABASE_URL is empty; using in-memory store")
		return memStore, func() {}
	}

	pgStore, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("postgres connection failed", "error", err)
		os.Exit(1)
	}
	if err := pgStore.ApplyMigrations(ctx, "migrations"); err != nil {
		logger.Error("postgres migrations failed", "error", err)
		os.Exit(1)
	}
	if err := pgStore.SeedDemoShop(ctx); err != nil {
		logger.Error("postgres demo shop seed failed", "error", err)
		os.Exit(1)
	}

	logger.Info("using postgres store")
	return pgStore, pgStore.Close
}

func buildUploadPresigner(ctx context.Context, cfg config.Config, logger *slog.Logger) httpapi.UploadPresigner {
	if !cfg.Storage.Enabled() {
		logger.Warn("storage config is incomplete; create-job responses will not include upload URLs")
		return nil
	}

	presigner, err := storage.NewPresigner(ctx, storage.Config{
		Endpoint:        cfg.Storage.Endpoint,
		Region:          cfg.Storage.Region,
		Bucket:          cfg.Storage.Bucket,
		AccessKey:       cfg.Storage.AccessKey,
		SecretKey:       cfg.Storage.SecretKey,
		PublicBaseURL:   cfg.Storage.PublicBaseURL,
		PresignDuration: cfg.Storage.PresignDuration(),
	})
	if err != nil {
		logger.Error("storage presigner setup failed", "error", err)
		os.Exit(1)
	}

	logger.Info("using storage presigner", "bucket", cfg.Storage.Bucket)
	return presigner
}
