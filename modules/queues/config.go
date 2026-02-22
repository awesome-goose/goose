package queues

import "time"

// Config holds configuration for the Queue module
type Config struct {
	// Queue is the default queue name for jobs
	Queue string
	// DefaultRetryLimit is the default number of retries for failed jobs
	DefaultRetryLimit int
	// DefaultRetryDelay is the default delay between retries (in milliseconds)
	DefaultRetryDelay int
	// PollInterval is how often workers poll for new jobs
	PollInterval time.Duration
	// CleanupInterval is how often expired/completed jobs are cleaned up (0 means no cleanup)
	CleanupInterval time.Duration
	// RetentionPeriod is how long to keep completed/failed jobs (default: 7 days)
	RetentionPeriod time.Duration
	// StaleJobTimeout is how long a job can be in_progress before being recovered (default: 5 minutes)
	StaleJobTimeout time.Duration
	// EnableStaleJobRecovery enables automatic recovery of stale jobs (default: true)
	EnableStaleJobRecovery bool
}

// JobConfig holds configuration for individual jobs
type JobConfig struct {
	Priority   int        `json:"priority,omitempty"`
	RetryLimit int        `json:"retryLimit,omitempty"`
	RetryDelay int        `json:"retryDelay,omitempty"` // milliseconds
	StartAt    *time.Time `json:"startAt,omitempty"`
	ExpireAt   *time.Time `json:"expireAt,omitempty"`
	Singleton  bool       `json:"singleton,omitempty"`
	Frequency  string     `json:"frequency,omitempty"` // cron expression
	TimeoutMs  int        `json:"timeoutMs,omitempty"` // job execution timeout
}

// Option is a functional option for configuring the Queue module
type Option func(*Config)

// WithQueue sets the default queue name
func WithQueue(queue string) Option {
	return func(c *Config) {
		c.Queue = queue
	}
}

// WithDefaultRetryLimit sets the default retry limit for jobs
func WithDefaultRetryLimit(limit int) Option {
	return func(c *Config) {
		c.DefaultRetryLimit = limit
	}
}

// WithDefaultRetryDelay sets the default retry delay for jobs
func WithDefaultRetryDelay(delay int) Option {
	return func(c *Config) {
		c.DefaultRetryDelay = delay
	}
}

// WithPollInterval sets how often workers poll for new jobs
func WithPollInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.PollInterval = interval
	}
}

// WithCleanupInterval sets how often expired jobs are cleaned up
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.CleanupInterval = interval
	}
}

// WithRetentionPeriod sets how long to keep completed/failed jobs
func WithRetentionPeriod(period time.Duration) Option {
	return func(c *Config) {
		c.RetentionPeriod = period
	}
}

// WithStaleJobTimeout sets how long a job can be in_progress before recovery
func WithStaleJobTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.StaleJobTimeout = timeout
	}
}

// WithStaleJobRecovery enables or disables stale job recovery
func WithStaleJobRecovery(enabled bool) Option {
	return func(c *Config) {
		c.EnableStaleJobRecovery = enabled
	}
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Queue:                  DefaultQueue,
		DefaultRetryLimit:      DefaultRetryLimit,
		DefaultRetryDelay:      DefaultRetryDelay,
		PollInterval:           time.Duration(DefaultPollInterval) * time.Millisecond,
		CleanupInterval:        time.Hour,
		RetentionPeriod:        7 * 24 * time.Hour,
		StaleJobTimeout:        time.Duration(DefaultStaleJobTimeout) * time.Second,
		EnableStaleJobRecovery: true,
	}
}
