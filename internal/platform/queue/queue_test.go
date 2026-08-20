package queue

import (
	"context"
	"testing"
	"time"

	"echobackend/config"
)

func TestQueue_EmptyConfig(t *testing.T) {
	svc := NewService(config.QueueConfig{})
	if svc.IsConfigured() {
		t.Fatal("expected queue to not be configured with empty RedisURL")
	}

	// Calling Handle, Start, Close, EnqueueJSON should not panic
	svc.Handle("test:task", func(ctx context.Context, payload []byte) error {
		return nil
	})
	svc.Start()

	err := svc.EnqueueJSON("test:task", map[string]string{"foo": "bar"}, TaskOptions{})
	if err == nil {
		t.Fatal("expected error enqueuing on unconfigured queue")
	}

	if err := svc.Close(); err != nil {
		t.Fatalf("unexpected error closing unconfigured queue: %v", err)
	}
}

func TestQueue_NilService(t *testing.T) {
	var svc *Service
	if svc.IsConfigured() {
		t.Fatal("nil service should not be configured")
	}

	svc.Handle("test:task", nil)
	svc.Start()

	if err := svc.Close(); err != nil {
		t.Fatalf("unexpected error closing nil service: %v", err)
	}

	if err := svc.EnqueueJSON("test:task", nil, TaskOptions{}); err == nil {
		t.Fatal("expected error on nil service EnqueueJSON")
	}
}

func TestQueue_InvalidRedisURL(t *testing.T) {
	svc := NewService(config.QueueConfig{
		RedisURL:       "invalid-url://localhost",
		ConnectTimeout: 100 * time.Millisecond,
		DefaultQueue:   "default",
	})
	if svc.IsConfigured() {
		t.Fatal("expected queue to fail-open with invalid redis URL")
	}
}
