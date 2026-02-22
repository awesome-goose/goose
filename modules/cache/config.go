package cache

import "time"

// Config holds configuration for the Cache module
type Config struct {
	// Group is the namespace for cache entries (similar to Redis DB number)
	Group string
	// DefaultTTL is the default time-to-live for cache entries (default: 5 minutes)
	DefaultTTL time.Duration
	// CleanupInterval is how often expired entries are cleaned up (0 means no cleanup)
	CleanupInterval time.Duration
}

// Option is a functional option for configuring the Cache module
type Option func(*Config)

// WithGroup sets the group/namespace for cache entries
func WithGroup(group string) Option {
	return func(c *Config) {
		c.Group = group
	}
}

// WithDefaultTTL sets the default TTL for cache entries
func WithDefaultTTL(ttl time.Duration) Option {
	return func(c *Config) {
		c.DefaultTTL = ttl
	}
}

// WithCleanupInterval sets how often expired entries are cleaned up
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.CleanupInterval = interval
	}
}
