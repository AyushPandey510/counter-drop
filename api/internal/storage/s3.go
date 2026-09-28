package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	Durations Durations
}

func (c S3Config) Enabled() bool {
	return c.Bucket != ""
}

// S3Store talks to any S3-compatible service (Cloudflare R2 in production).
type S3Store struct {
	bucket  string
	client  *s3.Client
	presign *s3.PresignClient
	dur     Durations
}

func NewS3Store(ctx context.Context, cfg S3Config) (*S3Store, error) {
	if !cfg.Enabled() {
		return nil, fmt.Errorf("storage config is incomplete")
	}
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	if cfg.Durations.PutTTL == 0 {
		cfg.Durations.PutTTL = 15 * time.Minute
	}
	if cfg.Durations.GetTTL == 0 {
		cfg.Durations.GetTTL = 5 * time.Minute
	}
	loadOpts := []func(*config.LoadOptions) error{config.WithRegion(cfg.Region)}
	if cfg.AccessKey != "" || cfg.SecretKey != "" {
		if cfg.AccessKey == "" || cfg.SecretKey == "" {
			return nil, fmt.Errorf("storage access key and secret key must be set together")
		}
		loadOpts = append(loadOpts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	})
	return &S3Store{bucket: cfg.Bucket, client: client, presign: s3.NewPresignClient(client), dur: cfg.Durations}, nil
}

func (s *S3Store) PresignPut(ctx context.Context, key, contentType string, size int64) (string, map[string]string, error) {
	req, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}, func(o *s3.PresignOptions) { o.Expires = s.dur.PutTTL })
	if err != nil {
		return "", nil, fmt.Errorf("presign put: %w", err)
	}
	return req.URL, map[string]string{"Content-Type": contentType}, nil
}

func (s *S3Store) PresignGet(ctx context.Context, key, filename, contentType string, download bool) (string, error) {
	disp := "inline"
	if download {
		disp = "attachment"
	}
	req, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(s.bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(disp + `; filename="` + safeFilename(filename) + `"`),
		ResponseContentType:        aws.String(contentType),
		ResponseCacheControl:       aws.String("private, no-store"),
	}, func(o *s3.PresignOptions) { o.Expires = s.dur.GetTTL })
	if err != nil {
		return "", fmt.Errorf("presign get: %w", err)
	}
	return req.URL, nil
}

func (s *S3Store) Head(ctx context.Context, key string) (ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
			return ObjectInfo{}, ErrObjectNotFound
		}
		return ObjectInfo{}, err
	}
	info := ObjectInfo{}
	if out.ContentLength != nil {
		info.Size = *out.ContentLength
	}
	if out.ContentType != nil {
		info.ContentType = *out.ContentType
	}
	return info, nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey") {
			return nil
		}
	}
	return err
}

func safeFilename(name string) string {
	r := strings.NewReplacer(`"`, "", "\\", "", "\n", "", "\r", "")
	name = r.Replace(name)
	if name == "" {
		return "file"
	}
	return name
}
