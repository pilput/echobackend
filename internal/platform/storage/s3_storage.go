package storage

import (
	"context"
	"errors"
	"io"
	"time"

	"echobackend/config"
	"echobackend/pkg/applog"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var log = applog.Component("storage")

type S3Storage struct {
	client *minio.Client
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

	minioClient, err := minio.New(cfg.S3.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3.AccessKey, cfg.S3.SecretKey, ""),
		Secure: cfg.S3.UseSSL,
	})
	if err != nil {
		log.Error("failed to create MinIO/S3 client", "error", err)
		return nil
	}

	return &S3Storage{
		client: minioClient,
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

	_, err := s.client.PutObject(ctx, s.bucket, path, file, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *S3Storage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("storage is not configured")
	}

	ctx, cancel := context.WithTimeout(ctx, s3GetTimeout)
	object, err := s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		cancel()
		return nil, err
	}

	// Return a wrapper that will cancel the context when closed
	return &readCloserWithCancel{object, cancel}, nil
}

func (s *S3Storage) Delete(ctx context.Context, path string) error {
	if s == nil || s.client == nil {
		return errors.New("storage is not configured")
	}

	ctx, cancel := context.WithTimeout(ctx, s3DeleteTimeout)
	defer cancel()

	return s.client.RemoveObject(ctx, s.bucket, path, minio.RemoveObjectOptions{})
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
