package storage

import (
	"bytes"
	"context"
	"io"
	"testing"

	"echobackend/config"
)

func TestS3Storage_NilHandling(t *testing.T) {
	var s *S3Storage

	ctx := context.Background()
	if err := s.Save(ctx, "test.txt", bytes.NewReader([]byte("data")), "text/plain"); err == nil {
		t.Error("expected error when saving with nil S3Storage")
	}

	if _, err := s.Get(ctx, "test.txt"); err == nil {
		t.Error("expected error when getting with nil S3Storage")
	}

	if err := s.Delete(ctx, "test.txt"); err == nil {
		t.Error("expected error when deleting with nil S3Storage")
	}
}

func TestS3Storage_NilFileSave(t *testing.T) {
	s := &S3Storage{}
	ctx := context.Background()

	// Storage client is nil
	if err := s.Save(ctx, "test.txt", nil, "text/plain"); err == nil {
		t.Error("expected error when storage client is nil")
	}
}

func TestNewS3Storage_DisabledWhenMissingConfig(t *testing.T) {
	if s := NewS3Storage(nil); s != nil {
		t.Errorf("expected nil for nil config, got %v", s)
	}

	cfg := &config.Config{}
	if s := NewS3Storage(cfg); s != nil {
		t.Errorf("expected nil for empty S3 config, got %v", s)
	}
}

func TestNewS3Storage_Initialization(t *testing.T) {
	cfg := &config.Config{
		S3: config.S3Config{
			Endpoint:  "localhost:9000",
			AccessKey: "testkey",
			SecretKey: "testsecret",
			Bucket:    "testbucket",
			UseSSL:    false,
		},
	}

	s := NewS3Storage(cfg)
	if s == nil {
		t.Fatal("expected non-nil S3Storage")
	}
	if s.bucket != "testbucket" {
		t.Errorf("expected bucket 'testbucket', got %q", s.bucket)
	}
	if s.client == nil {
		t.Error("expected non-nil AWS S3 client")
	}

	// Test Save with nil file
	if err := s.Save(context.Background(), "path", nil, "text/plain"); err == nil {
		t.Error("expected error when file is nil")
	}
}

func TestReadCloserWithCancel(t *testing.T) {
	cancelled := false
	cancel := func() {
		cancelled = true
	}

	rc := &readCloserWithCancel{
		ReadCloser: io.NopCloser(bytes.NewReader([]byte("test"))),
		cancel:     cancel,
	}

	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, rc)
	_ = rc.Close()

	if !cancelled {
		t.Error("expected cancel function to be called on Close")
	}
}
