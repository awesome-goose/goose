package queues

import (
	"context"
	"errors"
	"time"
)

// Common errors
var (
	ErrQueueNotFound  = errors.New("queue not found")
	ErrJobNotFound    = errors.New("job not found")
	ErrJobExpired     = errors.New("job expired")
	ErrJobLocked      = errors.New("job locked by another worker")
	ErrShuttingDown   = errors.New("queue service is shutting down")
	ErrJobTimeout     = errors.New("job execution timed out")
	ErrRetryExhausted = errors.New("retry limit exceeded")
	ErrInvalidHandler = errors.New("invalid job handler")
)

// Queue status constants
const (
	QueueStatusActive   = "active"
	QueueStatusPaused   = "paused"
	QueueStatusInactive = "inactive"
)

// Job status constants
const (
	JobStatusNew        = "new"
	JobStatusInProgress = "in_progress"
	JobStatusSuccess    = "success"
	JobStatusFailed     = "failed"
	JobStatusRetrying   = "retrying"
	JobStatusExpired    = "expired"
	JobStatusTimeout    = "timeout"
)

// DefaultQueue is used when no queue is specified
const DefaultQueue = "default"

// Default values
const (
	DefaultPriority          = 0
	DefaultRetryLimit        = 3
	DefaultRetryDelay        = 1000 // milliseconds
	DefaultMinWorkers        = 1
	DefaultMaxWorkers        = 4
	DefaultPollInterval      = 3000 // milliseconds
	DefaultIdleThreshold     = 3
	DefaultJobTimeout        = 30000 // milliseconds (30 seconds)
	DefaultStaleJobTimeout   = 300   // seconds (5 minutes) - jobs stuck in_progress longer than this are recovered
	DefaultBackoffMultiplier = 2     // exponential backoff multiplier
	DefaultMaxBackoffDelay   = 60000 // max backoff delay in milliseconds (1 minute)
)

// JobHandlerFn is a function that processes a job
// It receives a context and the job, returns a result or error
// The context can be used for cancellation and timeout
type JobHandlerFn func(ctx context.Context, job *QueueJob) (any, error)

// JobHandlerFnSimple is a simplified handler without context (for convenience)
type JobHandlerFnSimple func(job *QueueJob) (any, error)

// JobHandler defines a handler for processing jobs from a specific queue
type JobHandler struct {
	// Queue is the name of the queue to process jobs from
	Queue string
	// Job is the name/type of the job to process
	Job string
	// Handler is the function that processes the job (with context)
	Handler JobHandlerFn
	// MinWorkers is the minimum number of workers (default: 1)
	MinWorkers int
	// MaxWorkers is the maximum number of workers (default: 4)
	MaxWorkers int
	// PollIntervalMs is how often to poll for jobs in milliseconds (default: 3000)
	PollIntervalMs int
	// IdleThreshold is how many idle polls before a worker self-terminates (default: 3)
	IdleThreshold int
	// TimeoutMs is the maximum time a job can run in milliseconds (default: 30000)
	TimeoutMs int
	// UseExponentialBackoff enables exponential backoff for retries (default: false)
	UseExponentialBackoff bool
}

// WithMinWorkers sets the minimum number of workers for a handler
func (h *JobHandler) WithMinWorkers(min int) *JobHandler {
	h.MinWorkers = min
	return h
}

// WithMaxWorkers sets the maximum number of workers for a handler
func (h *JobHandler) WithMaxWorkers(max int) *JobHandler {
	h.MaxWorkers = max
	return h
}

// WithPollInterval sets the poll interval in milliseconds
func (h *JobHandler) WithPollInterval(ms int) *JobHandler {
	h.PollIntervalMs = ms
	return h
}

// WithIdleThreshold sets the idle threshold before worker self-terminates
func (h *JobHandler) WithIdleThreshold(threshold int) *JobHandler {
	h.IdleThreshold = threshold
	return h
}

// WithTimeout sets the job execution timeout in milliseconds
func (h *JobHandler) WithTimeout(ms int) *JobHandler {
	h.TimeoutMs = ms
	return h
}

// WithExponentialBackoff enables exponential backoff for retries
func (h *JobHandler) WithExponentialBackoff() *JobHandler {
	h.UseExponentialBackoff = true
	return h
}

// NewHandler creates a new job handler with defaults (with context)
func NewHandler(queue, job string, fn JobHandlerFn) *JobHandler {
	return &JobHandler{
		Queue:                 queue,
		Job:                   job,
		Handler:               fn,
		MinWorkers:            DefaultMinWorkers,
		MaxWorkers:            DefaultMaxWorkers,
		PollIntervalMs:        DefaultPollInterval,
		IdleThreshold:         DefaultIdleThreshold,
		TimeoutMs:             DefaultJobTimeout,
		UseExponentialBackoff: false,
	}
}

// NewSimpleHandler creates a handler from a simple function (without context)
// The context is ignored in the wrapper
func NewSimpleHandler(queue, job string, fn JobHandlerFnSimple) *JobHandler {
	return NewHandler(queue, job, func(ctx context.Context, j *QueueJob) (any, error) {
		return fn(j)
	})
}

// TypedHandler is a helper to create type-safe job handlers
// T is the type of the job data payload
type TypedHandler[T any] struct {
	Queue   string
	Job     string
	Handler func(ctx context.Context, data T, job *QueueJob) (any, error)
}

// ToJobHandler converts a TypedHandler to a JobHandler
func (th *TypedHandler[T]) ToJobHandler() *JobHandler {
	return NewHandler(th.Queue, th.Job, func(ctx context.Context, job *QueueJob) (any, error) {
		data, err := GetJobData[T](job)
		if err != nil {
			return nil, err
		}
		return th.Handler(ctx, data, job)
	})
}

// NewTypedHandler creates a type-safe handler
func NewTypedHandler[T any](queue, job string, fn func(ctx context.Context, data T, job *QueueJob) (any, error)) *TypedHandler[T] {
	return &TypedHandler[T]{
		Queue:   queue,
		Job:     job,
		Handler: fn,
	}
}

// JobResult represents the result of a job execution
type JobResult struct {
	Success bool
	Result  any
	Error   error
	Attempt int
	Elapsed time.Duration
}
