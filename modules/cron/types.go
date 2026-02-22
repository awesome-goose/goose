package cron

import (
	"context"
	"errors"
	"time"
)

// Common errors
var (
	ErrJobNotFound    = errors.New("cron job not found")
	ErrJobExpired     = errors.New("cron job expired")
	ErrJobLocked      = errors.New("cron job locked by another worker")
	ErrShuttingDown   = errors.New("cron service is shutting down")
	ErrJobTimeout     = errors.New("cron job execution timed out")
	ErrRetryExhausted = errors.New("retry limit exceeded")
	ErrInvalidHandler = errors.New("invalid cron handler")
	ErrInvalidPattern = errors.New("invalid cron pattern")
)

// Job status constants
const (
	JobStatusPending    = "pending"
	JobStatusInProgress = "in_progress"
	JobStatusSuccess    = "success"
	JobStatusFailed     = "failed"
	JobStatusExpired    = "expired"
)

// Log status constants
const (
	LogStatusSuccess = "success"
	LogStatusFailed  = "failed"
)

// Default values
const (
	DefaultPriority          = 0
	DefaultRetryLimit        = 3
	DefaultRetryDelay        = 10000 // milliseconds
	DefaultMaxRetryDelay     = 15000 // max retry delay allowed
	DefaultTickInterval      = 60    // seconds (check every minute)
	DefaultJobTimeout        = 60000 // milliseconds (60 seconds)
	DefaultStaleJobTimeout   = 300   // seconds (5 minutes)
	DefaultToleranceMs       = 60000 // 1 minute tolerance for pattern matching
	DefaultBackoffMultiplier = 2     // exponential backoff multiplier
	DefaultMaxBackoffDelay   = 60000 // max backoff delay in milliseconds
)

// CronHandlerFn is a function that processes a cron job
// It receives a context and the job, returns a result or error
type CronHandlerFn func(ctx context.Context, job *CronJob) (any, error)

// CronHandlerFnSimple is a simplified handler without context (for convenience)
type CronHandlerFnSimple func(job *CronJob) (any, error)

// CronHandler defines a handler for processing cron jobs
type CronHandler struct {
	// Group is the group/category of the cron job
	Group string
	// Name is the unique name of the cron job within the group
	Name string
	// Pattern is the cron expression (e.g., "0 * * * *" for every hour)
	Pattern string
	// Handler is the function that processes the job (with context)
	Handler CronHandlerFn
	// Config contains optional job configuration
	Config *CronConfig
	// TimeoutMs is the maximum time a job can run in milliseconds (default: 60000)
	TimeoutMs int
	// UseExponentialBackoff enables exponential backoff for retries (default: false)
	UseExponentialBackoff bool
}

// CronConfig holds optional configuration for a cron job
type CronConfig struct {
	Priority   int        `json:"priority,omitempty"`
	RetryLimit int        `json:"retryLimit,omitempty"`
	RetryDelay int        `json:"retryDelay,omitempty"` // milliseconds (must be <= 15000)
	StartAt    *time.Time `json:"startAt,omitempty"`
	ExpireAt   *time.Time `json:"expireAt,omitempty"`
}

// WithPriority sets the priority for the cron handler
func (h *CronHandler) WithPriority(priority int) *CronHandler {
	if h.Config == nil {
		h.Config = &CronConfig{}
	}
	h.Config.Priority = priority
	return h
}

// WithRetryLimit sets the retry limit for the cron handler
func (h *CronHandler) WithRetryLimit(limit int) *CronHandler {
	if h.Config == nil {
		h.Config = &CronConfig{}
	}
	h.Config.RetryLimit = limit
	return h
}

// WithRetryDelay sets the retry delay in milliseconds (max 15000)
func (h *CronHandler) WithRetryDelay(delayMs int) *CronHandler {
	if h.Config == nil {
		h.Config = &CronConfig{}
	}
	if delayMs > DefaultMaxRetryDelay {
		delayMs = DefaultMaxRetryDelay
	}
	h.Config.RetryDelay = delayMs
	return h
}

// WithStartAt sets when the cron job should start being active
func (h *CronHandler) WithStartAt(t time.Time) *CronHandler {
	if h.Config == nil {
		h.Config = &CronConfig{}
	}
	h.Config.StartAt = &t
	return h
}

// WithExpireAt sets when the cron job should expire
func (h *CronHandler) WithExpireAt(t time.Time) *CronHandler {
	if h.Config == nil {
		h.Config = &CronConfig{}
	}
	h.Config.ExpireAt = &t
	return h
}

// WithTimeout sets the job execution timeout in milliseconds
func (h *CronHandler) WithTimeout(ms int) *CronHandler {
	h.TimeoutMs = ms
	return h
}

// WithExponentialBackoff enables exponential backoff for retries
func (h *CronHandler) WithExponentialBackoff() *CronHandler {
	h.UseExponentialBackoff = true
	return h
}

// NewHandler creates a new cron handler with defaults
func NewHandler(group, name, pattern string, fn CronHandlerFn) *CronHandler {
	return &CronHandler{
		Group:                 group,
		Name:                  name,
		Pattern:               pattern,
		Handler:               fn,
		TimeoutMs:             DefaultJobTimeout,
		UseExponentialBackoff: false,
	}
}

// NewSimpleHandler creates a handler from a simple function (without context)
func NewSimpleHandler(group, name, pattern string, fn CronHandlerFnSimple) *CronHandler {
	return NewHandler(group, name, pattern, func(ctx context.Context, j *CronJob) (any, error) {
		return fn(j)
	})
}

// TypedHandler is a helper to create type-safe cron handlers
// T is the type of the config data payload
type TypedHandler[T any] struct {
	Group   string
	Name    string
	Pattern string
	Handler func(ctx context.Context, config T, job *CronJob) (any, error)
}

// ToCronHandler converts a TypedHandler to a CronHandler
func (th *TypedHandler[T]) ToCronHandler() *CronHandler {
	return NewHandler(th.Group, th.Name, th.Pattern, func(ctx context.Context, job *CronJob) (any, error) {
		config, err := GetJobConfig[T](job)
		if err != nil {
			return nil, err
		}
		return th.Handler(ctx, config, job)
	})
}

// NewTypedHandler creates a type-safe handler
func NewTypedHandler[T any](group, name, pattern string, fn func(ctx context.Context, config T, job *CronJob) (any, error)) *TypedHandler[T] {
	return &TypedHandler[T]{
		Group:   group,
		Name:    name,
		Pattern: pattern,
		Handler: fn,
	}
}

// CronResult represents the result of a cron job execution
type CronResult struct {
	Success bool
	Result  any
	Error   error
	Attempt int
	Elapsed time.Duration
}
