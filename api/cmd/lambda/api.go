package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"counter-drop/api/internal/app"
	"counter-drop/api/internal/backend"
	"counter-drop/api/internal/config"
	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/httpapi"
	"counter-drop/api/internal/lambdahttp"
	"counter-drop/api/internal/ratelimit"
	"counter-drop/api/internal/realtime"

	"github.com/aws/aws-lambda-go/lambda"
)

var errNoBucket = errors.New("CD_STORAGE_BUCKET is required (Lambda has no lasting local disk)")

// checkAPIConfig refuses settings that can't work on Lambda.
func checkAPIConfig(cfg config.Config) error {
	switch {
	case !strings.HasPrefix(cfg.Realtime.WSURL, "wss://"):
		return errors.New("CD_REALTIME_WS_URL must be the wss:// address of the WebSocket API (Lambda can't hold Server-Sent Events open)")
	case len(cfg.Realtime.Key) < 32:
		return errors.New("CD_REALTIME_KEY must be at least 32 characters (the same value as the WebSocket Lambda)")
	case !cfg.Storage.S3Enabled():
		return errNoBucket
	case cfg.Dynamo.Endpoint != "":
		return errors.New("CD_DYNAMODB_ENDPOINT must be empty on AWS")
	}
	return nil
}

// buildAPI wires the same server as cmd/api, with the Lambda differences: WebSocket instead of SSE,
// shared (DynamoDB) limits for sign-in and new jobs, and no background goroutines.
func buildAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) (*httpapi.Server, error) {
	policy := domain.Policy{UndoWindow: cfg.UndoWindow, AbandonAfter: cfg.AbandonAfter}
	st, err := backend.Open(ctx, cfg, policy)
	if err != nil {
		return nil, err
	}
	if cfg.DemoSeed {
		if err := st.SeedDemo(ctx); err != nil {
			return nil, err
		}
	}
	objects, local, err := app.BuildStorage(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}
	srv := &httpapi.Server{Store: st, Objects: objects, Local: local, Hub: realtime.NewHub(), Logger: logger, Cfg: cfg, NoSSE: true}
	if shared, ok := st.(ratelimit.Shared); ok {
		srv.SharedLimits = shared
	}
	return srv, nil
}

func startAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) {
	if err := checkAPIConfig(cfg); err != nil {
		fatal(logger, "config", err)
	}
	srv, err := buildAPI(ctx, cfg, logger)
	if err != nil {
		fatal(logger, "startup", err)
	}
	lambda.Start(lambdahttp.Handler(srv.Handler()))
}
