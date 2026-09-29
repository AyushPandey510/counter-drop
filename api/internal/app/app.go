// Package app holds wiring shared by the long-running server (cmd/api) and the Lambda binary (cmd/lambda).
package app

import (
	"context"
	"errors"
	"log/slog"

	"counter-drop/api/internal/config"
	"counter-drop/api/internal/storage"
	"counter-drop/api/internal/store"
)

// BuildStorage returns the file store: S3/R2 when CD_STORAGE_BUCKET is set, otherwise local disk
// (the second value is non-nil only for local disk, whose signed links the API serves itself).
func BuildStorage(ctx context.Context, cfg config.Config, logger *slog.Logger) (storage.ObjectStore, *storage.LocalStore, error) {
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
