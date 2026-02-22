package cron

import "time"

// Config holds configuration for the Cron module
type Config struct {
	// TickInterval is how often the cron runner checks for jobs to run (default: 60s)
	TickInterval time.Duration
	// Timezone is the timezone for cron pattern evaluation (default: UTC)
	Timezone string
	// DefaultRetryLimit is the default number of retries for failed jobs
	DefaultRetryLimit int
	// DefaultRetryDelay is the default delay between retries (in milliseconds)
	DefaultRetryDelay int
	// StaleJobTimeout is how long a job can be in_progress before being recovered (default: 5 minutes)
	StaleJobTimeout time.Duration
	// EnableStaleJobRecovery enables automatic recovery of stale jobs (default: true)
	EnableStaleJobRecovery bool
	// CleanupInterval is how often old logs are cleaned up (0 means no cleanup)
	CleanupInterval time.Duration
	// LogRetentionPeriod is how long to keep execution logs (default: 7 days)
	LogRetentionPeriod time.Duration
}

// Option is a functional option for configuring the Cron module
type Option func(*Config)

// WithTickInterval sets how often the cron runner checks for jobs
func WithTickInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.TickInterval = interval
	}
}

// WithTimezone sets the timezone for cron pattern evaluation
func WithTimezone(tz string) Option {
	return func(c *Config) {
		c.Timezone = tz
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

// WithStaleJobTimeout sets how long a job can be in_progress before recovery
func WithStaleJobTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.StaleJobTimeout = timeout
	}
}

// WithStaleJobRecovery enables/disables automatic stale job recovery
func WithStaleJobRecovery(enabled bool) Option {
	return func(c *Config) {
		c.EnableStaleJobRecovery = enabled
	}
}

// WithCleanupInterval sets how often old logs are cleaned up
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.CleanupInterval = interval
	}
}

// WithLogRetentionPeriod sets how long to keep execution logs
func WithLogRetentionPeriod(period time.Duration) Option {
	return func(c *Config) {
		c.LogRetentionPeriod = period
	}
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		TickInterval:           time.Duration(DefaultTickInterval) * time.Second,
		Timezone:               "Etc/UTC",
		DefaultRetryLimit:      DefaultRetryLimit,
		DefaultRetryDelay:      DefaultRetryDelay,
		StaleJobTimeout:        time.Duration(DefaultStaleJobTimeout) * time.Second,
		EnableStaleJobRecovery: true,
		CleanupInterval:        time.Hour,
		LogRetentionPeriod:     7 * 24 * time.Hour,
	}
}

// Apply applies the given options to the config
func (c *Config) Apply(opts ...Option) *Config {
	for _, opt := range opts {
		opt(c)
	}
	return c
}
