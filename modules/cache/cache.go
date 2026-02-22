package cache

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/awesome-goose/goose/modules/sql"
	"github.com/awesome-goose/goose/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DefaultCacheTTL is the default TTL for cache entries (5 minutes)
const DefaultCacheTTL = 5 * time.Minute

// Cache is the caching service
type Cache struct {
	db     *sql.Db   `inject:""`
	log    types.Log `inject:""`
	config *Config   `inject:""`
}

// group returns the configured group or default
func (c *Cache) group() string {
	if c.config != nil && c.config.Group != "" {
		return c.config.Group
	}
	return DefaultGroup
}

// CacheFn is a function that generates a value to be cached
type CacheFn[T any] func() (T, error)

// Remember retrieves a value from cache or executes a function to generate and cache the value.
// This is the primary caching method - it handles the get-or-set pattern automatically.
//
// Parameters:
//   - key: The cache key
//   - fn: Function to execute if value is not in cache
//   - ttl: Optional time-to-live (default: 5 minutes)
//
// Returns the cached or newly generated value.
func Remember[T any](c *Cache, key string, fn CacheFn[T], ttl ...time.Duration) (T, error) {
	var zero T

	// Try to get from cache first
	existing, err := GetAs[T](c, key)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrKeyNotFound) && !errors.Is(err, ErrKeyExpired) {
		return zero, err
	}

	// Not in cache, execute the function
	value, err := fn()
	if err != nil {
		return zero, err
	}

	// Store in cache
	cacheTTL := DefaultCacheTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		cacheTTL = ttl[0]
	} else if c.config != nil && c.config.DefaultTTL > 0 {
		cacheTTL = c.config.DefaultTTL
	}

	if err := c.Set(key, value, cacheTTL); err != nil {
		// Log error but still return the value
		return value, nil
	}

	return value, nil
}

// GetAs retrieves a typed value from cache
func GetAs[T any](c *Cache, key string) (T, error) {
	var zero T
	group := c.group()

	var store CacheStore
	err := c.db.DB.Where("\"group\" = ? AND \"key\" = ? AND status = ?", group, key, StatusActive).First(&store).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return zero, ErrKeyNotFound
		}
		return zero, err
	}

	// Check if expired
	if store.ExpiredAt != nil && store.ExpiredAt.Before(time.Now().UTC()) {
		return zero, ErrKeyExpired
	}

	var value T
	if err := json.Unmarshal(store.Value, &value); err != nil {
		return zero, err
	}

	return value, nil
}

// Get retrieves a value from cache. Returns ErrKeyNotFound if key doesn't exist.
func (c *Cache) Get(key string) (any, error) {
	group := c.group()

	var store CacheStore
	err := c.db.DB.Where("\"group\" = ? AND \"key\" = ? AND status = ?", group, key, StatusActive).First(&store).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}

	// Check if expired
	if store.ExpiredAt != nil && store.ExpiredAt.Before(time.Now().UTC()) {
		return nil, ErrKeyExpired
	}

	var value any
	if err := json.Unmarshal(store.Value, &value); err != nil {
		return nil, err
	}

	return value, nil
}

// Set stores a value in cache with optional TTL.
func (c *Cache) Set(key string, value any, ttl ...time.Duration) error {
	group := c.group()

	valueBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var expiredAt *time.Time
	if len(ttl) > 0 && ttl[0] > 0 {
		t := time.Now().UTC().Add(ttl[0])
		expiredAt = &t
	} else if c.config != nil && c.config.DefaultTTL > 0 {
		t := time.Now().UTC().Add(c.config.DefaultTTL)
		expiredAt = &t
	} else {
		// Default TTL for cache
		t := time.Now().UTC().Add(DefaultCacheTTL)
		expiredAt = &t
	}

	now := time.Now().UTC()

	// Try to update existing entry first
	result := c.db.DB.Model(&CacheStore{}).
		Where("\"group\" = ? AND \"key\" = ?", group, key).
		Updates(map[string]any{
			"value":      valueBytes,
			"expired_at": expiredAt,
			"updated_at": &now,
			"status":     StatusActive,
			"deleted_at": nil,
		})

	if result.Error != nil {
		return result.Error
	}

	// If no rows affected, create new entry
	if result.RowsAffected == 0 {
		store := &CacheStore{
			Id:        uuid.New().String(),
			Group:     group,
			Key:       key,
			Value:     valueBytes,
			ExpiredAt: expiredAt,
			CreatedAt: &now,
			UpdatedAt: &now,
			Status:    StatusActive,
		}
		return c.db.DB.Create(store).Error
	}

	return nil
}

// Delete removes one or more keys from cache.
// Returns the number of keys that were deleted.
func (c *Cache) Delete(keys ...string) (int64, error) {
	group := c.group()

	if len(keys) == 0 {
		return 0, nil
	}

	now := time.Now().UTC()
	result := c.db.DB.Model(&CacheStore{}).
		Where("\"group\" = ? AND \"key\" IN ? AND status = ?", group, keys, StatusActive).
		Updates(map[string]any{
			"status":     StatusDeleted,
			"deleted_at": &now,
			"updated_at": &now,
		})

	return result.RowsAffected, result.Error
}

// Invalidate removes a key from cache (alias for Delete with single key)
func (c *Cache) Invalidate(key string) error {
	_, err := c.Delete(key)
	return err
}

// InvalidateAll removes all keys in the configured group
func (c *Cache) InvalidateAll() (int64, error) {
	group := c.group()

	now := time.Now().UTC()
	result := c.db.DB.Model(&CacheStore{}).
		Where("\"group\" = ? AND status = ?", group, StatusActive).
		Updates(map[string]any{
			"status":     StatusDeleted,
			"deleted_at": &now,
			"updated_at": &now,
		})

	return result.RowsAffected, result.Error
}

// Has checks if a key exists in cache and is not expired
func (c *Cache) Has(key string) (bool, error) {
	group := c.group()

	var count int64
	err := c.db.DB.Model(&CacheStore{}).
		Where("\"group\" = ? AND \"key\" = ? AND status = ? AND (expired_at IS NULL OR expired_at > ?)",
			group, key, StatusActive, time.Now().UTC()).
		Count(&count).Error

	return count > 0, err
}

// Flush removes all cache entries across all groups
func (c *Cache) Flush() (int64, error) {
	now := time.Now().UTC()
	result := c.db.DB.Model(&CacheStore{}).
		Where("status = ?", StatusActive).
		Updates(map[string]any{
			"status":     StatusDeleted,
			"deleted_at": &now,
			"updated_at": &now,
		})

	return result.RowsAffected, result.Error
}

// CleanupExpired removes expired cache entries from the database.
// This is useful for periodic maintenance to keep the cache table small.
// Returns the number of entries cleaned up.
func (c *Cache) CleanupExpired() (int64, error) {
	now := time.Now().UTC()
	result := c.db.DB.Model(&CacheStore{}).
		Where("status = ? AND expired_at IS NOT NULL AND expired_at <= ?", StatusActive, now).
		Updates(map[string]any{
			"status":     StatusDeleted,
			"deleted_at": &now,
			"updated_at": &now,
		})

	return result.RowsAffected, result.Error
}

// PermanentDelete physically removes deleted cache entries from the database.
// Use this for periodic cleanup to reclaim disk space.
// Returns the number of entries permanently deleted.
func (c *Cache) PermanentDelete() (int64, error) {
	result := c.db.DB.Unscoped().Where("status = ?", StatusDeleted).Delete(&CacheStore{})
	return result.RowsAffected, result.Error
}
