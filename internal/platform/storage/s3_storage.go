package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"echobackend/config"
	"echobackend/pkg/applog"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var log = applog.Component("storage")

type S3Storage struct {
	client *s3.Client
	bucket string
}

const (
	s3SaveTimeout   = 30 * time.Second
	s3GetTimeout    = 30 * time.Second
	s3DeleteTimeout = 10 * time.Second
)

func NewS3Storage(cfg *config.Config) *S3Storage {
	if cfg == nil || cfg.S3.Endpoint == "" || cfg.S3.Bucket == "" {
		log.Warn("S3 configuration missing, storage disabled")
		return nil
	}

	endpoint := cfg.S3.Endpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		scheme := "https"
		if !cfg.S3.UseSSL {
			scheme = "http"
		}
		endpoint = fmt.Sprintf("%s://%s", scheme, endpoint)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.S3.AccessKey,
			cfg.S3.SecretKey,
			"",
		)),
		awsconfig.WithRegion("us-east-1"),
	)
	if err != nil {
		log.Error("failed to load AWS/S3 config", "error", err)
		return nil
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &S3Storage{
		client: client,
		bucket: cfg.S3.Bucket,
	}
}

func (s *S3Storage) Save(ctx context.Context, path string, file io.Reader, contentType string) error {
	if s == nil || s.client == nil {
		return errors.New("storage is not configured")
	}
	if file == nil {
		return errors.New("file cannot be nil")
	}

	ctx, cancel := context.WithTimeout(ctx, s3SaveTimeout)
	defer cancel()

	if rc, ok := file.(io.ReadCloser); ok {
		defer func() { _ = rc.Close() }()
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   file,
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	_, err := s.client.PutObject(ctx, input)
	return err
}

func (s *S3Storage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("storage is not configured")
	}

	ctx, cancel := context.WithTimeout(ctx, s3GetTimeout)
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		cancel()
		return nil, err
	}

	return &readCloserWithCancel{out.Body, cancel}, nil
}

func (s *S3Storage) Delete(ctx context.Context, path string) error {
	if s == nil || s.client == nil {
		return errors.New("storage is not configured")
	}

	ctx, cancel := context.WithTimeout(ctx, s3DeleteTimeout)
	defer cancel()

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	return err
}

type readCloserWithCancel struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *readCloserWithCancel) Close() error {
	err := r.ReadCloser.Close()
	r.cancel()
	return err
}
