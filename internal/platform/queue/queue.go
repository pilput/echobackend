package queue

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"echobackend/config"
	"echobackend/pkg/applog"

	"github.com/hibiken/asynq"
)

var log = applog.Component("queue")

// SkipRetry marks a task error as permanent.
var SkipRetry = asynq.SkipRetry

// HandlerFunc handles a queued task payload.
type HandlerFunc func(ctx context.Context, payload []byte) error

// TaskOptions controls how a task is enqueued.
type TaskOptions struct {
	Queue    string
	Timeout  time.Duration
	MaxRetry int
}

// Service owns the shared Asynq client and worker server.
type Service struct {
	mu           sync.RWMutex
	client       *asynq.Client
	server       *asynq.Server
	mux          *asynq.ServeMux
	defaultQueue string
	maxRetry     int
	started      bool
}

// NewService creates a shared Asynq queue service. Empty Redis URL disables it.
func NewService(cfg config.QueueConfig) *Service {
	service := &Service{
		mux:          asynq.NewServeMux(),
		defaultQueue: cfg.DefaultQueue,
		maxRetry:     cfg.MaxRetry,
	}

	if cfg.RedisURL == "" {
		log.Warn("QUEUE_REDIS_URL/REDIS_URL is empty, background jobs disabled")
		return service
	}

	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		log.Warn("invalid Redis URL, background jobs disabled", "error", err)
		return service
	}

	connectTimeout := cfg.ConnectTimeout
	if connectTimeout <= 0 || connectTimeout > 2*time.Second {
		connectTimeout = 2 * time.Second
	}
	redisOpt = withRedisTimeouts(redisOpt, connectTimeout)

	client := asynq.NewClient(redisOpt)
	if err := client.Ping(); err != nil {
		log.Warn("failed to connect to Redis, background jobs disabled", "error", err)
		_ = client.Close()
		return service
	}

	service.client = client
	service.server = asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: cfg.Concurrency,
		Queues: map[string]int{
			cfg.DefaultQueue: 1,
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(_ context.Context, task *asynq.Task, err error) {
			log.Error("task failed", "error", err, "task_type", task.Type())
		}),
	})

	log.Info("Asynq enabled", "queue", cfg.DefaultQueue, "concurrency", cfg.Concurrency)
	return service
}

// IsConfigured reports whether the queue broker and worker are available.
func (s *Service) IsConfigured() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client != nil && s.server != nil
}

// Handle registers a task handler.
func (s *Service) Handle(taskType string, handler HandlerFunc) {
	if s == nil || s.mux == nil || taskType == "" || handler == nil {
		return
	}

	s.mux.HandleFunc(taskType, func(ctx context.Context, task *asynq.Task) error {
		return handler(ctx, task.Payload())
	})
}

// Start begins processing registered task handlers.
func (s *Service) Start() {
	if s == nil {
		return
	}

	s.mu.Lock()
	if s.client == nil || s.server == nil || s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	server := s.server
	mux := s.mux
	s.mu.Unlock()

	go func() {
		if err := server.Run(mux); err != nil {
			log.Error("Asynq server stopped", "error", err)
		}
	}()
}

// Close stops the worker and closes the client.
func (s *Service) Close() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	server := s.server
	client := s.client
	s.server = nil
	s.client = nil
	s.mu.Unlock()

	if server != nil {
		server.Shutdown()
	}

	if client != nil {
		return client.Close()
	}

	return nil
}

// EnqueueJSON marshals payload as JSON and enqueues it as a background task.
func (s *Service) EnqueueJSON(taskType string, payload any, opts TaskOptions) error {
	if !s.IsConfigured() {
		return errors.New("queue service not configured")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	queueName := opts.Queue
	if queueName == "" {
		queueName = s.defaultQueue
	}

	maxRetry := opts.MaxRetry
	if maxRetry == 0 {
		maxRetry = s.maxRetry
	}

	taskOptions := []asynq.Option{
		asynq.Queue(queueName),
		asynq.MaxRetry(maxRetry),
	}
	if opts.Timeout > 0 {
		taskOptions = append(taskOptions, asynq.Timeout(opts.Timeout))
	}

	task := asynq.NewTask(taskType, body)

	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return errors.New("queue service not configured")
	}

	_, err = client.Enqueue(task, taskOptions...)
	return err
}

func withRedisTimeouts(opt asynq.RedisConnOpt, timeout time.Duration) asynq.RedisConnOpt {
	if timeout <= 0 {
		return opt
	}

	switch v := opt.(type) {
	case asynq.RedisClientOpt:
		v.DialTimeout = timeout
		v.ReadTimeout = timeout
		v.WriteTimeout = timeout
		return v
	case asynq.RedisFailoverClientOpt:
		v.DialTimeout = timeout
		v.ReadTimeout = timeout
		v.WriteTimeout = timeout
		return v
	case asynq.RedisClusterClientOpt:
		v.DialTimeout = timeout
		v.ReadTimeout = timeout
		v.WriteTimeout = timeout
		return v
	default:
		return opt
	}
}
