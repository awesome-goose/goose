package kv

import "time"

// Config holds configuration for the KV module
type Config struct {
	// Group is the namespace for keys (similar to Redis DB number)
	Group string
	// DefaultTTL is the default time-to-live for keys (0 means no expiration)
	DefaultTTL time.Duration
	// CleanupInterval is how often expired keys are cleaned up (0 means no cleanup)
	CleanupInterval time.Duration
}

// Option is a functional option for configuring the KV module
type Option func(*Config)

// WithGroup sets the group/namespace for keys
func WithGroup(group string) Option {
	return func(c *Config) {
		c.Group = group
	}
}

// WithDefaultTTL sets the default TTL for keys
func WithDefaultTTL(ttl time.Duration) Option {
	return func(c *Config) {
		c.DefaultTTL = ttl
	}
}

// WithCleanupInterval sets how often expired keys are cleaned up
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.CleanupInterval = interval
	}
}
