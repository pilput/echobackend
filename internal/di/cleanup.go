package di

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"
)

// CleanupManager manages cleanup of resources
type CleanupManager struct {
	cleaners []func() error
	mu       sync.RWMutex
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager() *CleanupManager {
	return &CleanupManager{
		cleaners: make([]func() error, 0),
	}
}

// Register adds a cleanup function to the cleanup manager
func (cm *CleanupManager) Register(cleanup func() error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.cleaners = append(cm.cleaners, cleanup)
}

// Cleanup executes all registered cleanup functions in reverse order
func (cm *CleanupManager) Cleanup(ctx context.Context) error {
	cm.mu.RLock()
	cleaners := make([]func() error, len(cm.cleaners))
	copy(cleaners, cm.cleaners)
	cm.mu.RUnlock()

	var errors []error

	// Cleanup in reverse order (LIFO)
	for _, cleaner := range slices.Backward(cleaners) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cleanup timeout: %w", ctx.Err())
		default:
			if err := cleaner(); err != nil {
				errors = append(errors, fmt.Errorf("cleanup error: %w", err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup completed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// CleanupWithTimeout performs cleanup with a timeout
func (cm *CleanupManager) CleanupWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return cm.Cleanup(ctx)
}
