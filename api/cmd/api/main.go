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

	"counter-drop/api/internal/app"
	"counter-drop/api/internal/backend"
	"counter-drop/api/internal/config"
	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/httpapi"
	"counter-drop/api/internal/realtime"
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

	if ws := cfg.Realtime.WSURL; ws != "" && ws != "local" && len(cfg.Realtime.Key) < 32 {
		return errors.New("CD_REALTIME_KEY (32+ characters, shared with the WebSocket Lambda) is required when CD_REALTIME_WS_URL is set")
	}
	policy := domain.Policy{UndoWindow: cfg.UndoWindow, AbandonAfter: cfg.AbandonAfter}
	st, err := backend.Open(ctx, cfg, policy)
	if err != nil {
		return err
	}
	defer st.Close()
	logger.Info("store ready", "table", cfg.Dynamo.Table, "local_endpoint", cfg.Dynamo.Endpoint)
	if cfg.DemoSeed {
		if err := st.SeedDemo(ctx); err != nil {
			return err
		}
		logger.Info("demo shop ready", "slug", "demo-print", "staff", "Owner (PIN 1234), Kavita (PIN 1111)")
	}

	objects, local, err := app.BuildStorage(ctx, cfg, logger)
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
