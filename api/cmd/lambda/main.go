// Command lambda is the AWS Lambda binary (ADR-001). One binary, several handlers, chosen by CD_RUNTIME:
//
//	lambda-api      the HTTP API behind API Gateway (HTTP API, payload 2.0) and CloudFront
//	lambda-sweeper  the deletion worker, run every minute by EventBridge Scheduler (reserved concurrency 1)
//	lambda-ws       API Gateway WebSocket routes: $connect (ticket check), $disconnect, $default (heartbeat)
//	lambda-push     DynamoDB stream of the main table → live-update messages to open connections
//
// Build for Lambda (arm64, provided.al2023): GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap ./cmd/lambda
package main

import (
	"context"
	"log/slog"
	"os"

	"counter-drop/api/internal/config"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx := context.Background()
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		fatal(logger, "aws config", err)
	}
	switch rt := os.Getenv("CD_RUNTIME"); rt {
	case "lambda-api":
		startAPI(ctx, cfg, logger)
	case "lambda-sweeper":
		startSweeper(ctx, cfg, logger)
	case "lambda-ws":
		startWS(cfg, awsCfg, logger)
	case "lambda-push":
		startPush(cfg, awsCfg, logger)
	default:
		logger.Error("unknown CD_RUNTIME (lambda-api, lambda-sweeper, lambda-ws, lambda-push)", "runtime", rt)
		os.Exit(1)
	}
}

// fatal stops a cold start with a clear log line; Lambda reports the init failure.
func fatal(logger *slog.Logger, msg string, err error) {
	logger.Error(msg, "error", err)
	os.Exit(1)
}
