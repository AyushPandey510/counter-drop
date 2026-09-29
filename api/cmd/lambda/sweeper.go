package main

import (
	"context"
	"log/slog"

	"counter-drop/api/internal/app"
	"counter-drop/api/internal/backend"
	"counter-drop/api/internal/config"
	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/tasks"

	"github.com/aws/aws-lambda-go/lambda"
)

// buildSweeper is the deletion worker for one scheduled run: abandon drafts, close uncollected jobs,
// delete due files (Delete + Head check). Live updates for these changes come from the DynamoDB
// stream, so it publishes nothing itself.
func buildSweeper(ctx context.Context, cfg config.Config, logger *slog.Logger) (*tasks.Deleter, error) {
	policy := domain.Policy{UndoWindow: cfg.UndoWindow, AbandonAfter: cfg.AbandonAfter}
	st, err := backend.Open(ctx, cfg, policy)
	if err != nil {
		return nil, err
	}
	objects, _, err := app.BuildStorage(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}
	return &tasks.Deleter{Store: st, Objects: objects, Logger: logger}, nil
}

func startSweeper(ctx context.Context, cfg config.Config, logger *slog.Logger) {
	if !cfg.Storage.S3Enabled() {
		fatal(logger, "config", errNoBucket)
	}
	d, err := buildSweeper(ctx, cfg, logger)
	if err != nil {
		fatal(logger, "startup", err)
	}
	// A failed run is retried by the scheduler; every step is safe to repeat.
	lambda.Start(func(ctx context.Context) error { return d.RunOnce(ctx) })
}
