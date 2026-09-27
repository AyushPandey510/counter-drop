package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKey       string
	SecretKey       string
	PublicBaseURL   string
	PresignDuration time.Duration
}

func (c Config) Enabled() bool {
	return c.Bucket != "" && c.AccessKey != "" && c.SecretKey != ""
}

type Presigner struct {
	bucket   string
	presign  *s3.PresignClient
	duration time.Duration
}

func NewPresigner(ctx context.Context, cfg Config) (*Presigner, error) {
	if !cfg.Enabled() {
		return nil, fmt.Errorf("storage config is incomplete")
	}
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	if cfg.PresignDuration == 0 {
		cfg.PresignDuration = 15 * time.Minute
	}

	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	})

	return &Presigner{
		bucket:   cfg.Bucket,
		presign:  s3.NewPresignClient(client),
		duration: cfg.PresignDuration,
	}, nil
}

func (p *Presigner) PresignUpload(ctx context.Context, objectKey string, contentType string) (string, error) {
	request, err := p.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}, func(options *s3.PresignOptions) {
		options.Expires = p.duration
	})
	if err != nil {
		return "", fmt.Errorf("presign put object: %w", err)
	}
	return request.URL, nil
}
