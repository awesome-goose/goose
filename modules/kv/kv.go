package kv

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/awesome-goose/goose/modules/sql"
	"github.com/awesome-goose/goose/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// KV is the key-value store service with Redis-compatible methods
type KV struct {
	db     *sql.Db   `inject:""`
	log    types.Log `inject:""`
	config *Config   `inject:""`
}

// group returns the configured group or default
func (kv *KV) group() string {
	if kv.config != nil && kv.config.Group != "" {
		return kv.config.Group
	}
	return DefaultGroup
}

// Get retrieves the value for a key. Returns ErrKeyNotFound if key doesn't exist.
// Redis compatible: GET key
func (kv *KV) Get(key string) (any, error) {
	group := kv.group()

	var store KVStore
	err := kv.db.DB.Where("\"group\" = ? AND \"key\" = ? AND status = ?", group, key, StatusActive).First(&store).Error
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

// Set stores a value for a key with optional TTL.
// Redis compatible: SET key value [EX seconds]
func (kv *KV) Set(key string, value any, ttl ...time.Duration) error {
	group := kv.group()

	valueBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var expiredAt *time.Time
	if len(ttl) > 0 && ttl[0] > 0 {
		t := time.Now().UTC().Add(ttl[0])
		expiredAt = &t
	} else if kv.config != nil && kv.config.DefaultTTL > 0 {
		t := time.Now().UTC().Add(kv.config.DefaultTTL)
		expiredAt = &t
	}

	now := time.Now().UTC()

	// Try to update existing store first
	result := kv.db.DB.Model(&KVStore{}).
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

	// If no rows affected, create new store
	if result.RowsAffected == 0 {
		store := &KVStore{
			Id:        uuid.New().String(),
			Group:     group,
			Key:       key,
			Value:     valueBytes,
			ExpiredAt: expiredAt,
			CreatedAt: &now,
			UpdatedAt: &now,
			Status:    StatusActive,
		}
		return kv.db.DB.Create(store).Error
	}

	return nil
}

// SetNX sets a key only if it does not already exist.
// Redis compatible: SETNX key value
// Returns true if key was set, false if key already exists.
func (kv *KV) SetNX(key string, value any, ttl ...time.Duration) (bool, error) {
	group := kv.group()

	valueBytes, err := json.Marshal(value)
	if err != nil {
		return false, err
	}

	var expiredAt *time.Time
	if len(ttl) > 0 && ttl[0] > 0 {
		t := time.Now().UTC().Add(ttl[0])
		expiredAt = &t
	} else if kv.config != nil && kv.config.DefaultTTL > 0 {
		t := time.Now().UTC().Add(kv.config.DefaultTTL)
		expiredAt = &t
	}

	// Start a transaction with row-level locking
	tx := kv.db.DB.Begin()
	if tx.Error != nil {
		return false, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if key exists with FOR UPDATE lock to prevent race conditions.
	// SQLite serializes write transactions and rejects the hint, so skip it there.
	var existing KVStore
	query := tx.Where("\"group\" = ? AND \"key\" = ? AND status = ?", group, key, StatusActive)
	if tx.Dialector.Name() != "sqlite" {
		query = query.Set("gorm:query_option", "FOR UPDATE")
	}
	err = query.First(&existing).Error

	if err == nil {
		// Key exists, check if expired
		if existing.ExpiredAt == nil || existing.ExpiredAt.After(time.Now().UTC()) {
			tx.Rollback()
			return false, nil // Key exists and is not expired
		}
		// Key is expired, we can overwrite it
		now := time.Now().UTC()
		err = tx.Model(&KVStore{}).
			Where("id = ?", existing.Id).
			Updates(map[string]any{
				"value":      valueBytes,
				"expired_at": expiredAt,
				"updated_at": &now,
				"status":     StatusActive,
			}).Error
		if err != nil {
			tx.Rollback()
			return false, err
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		// Key doesn't exist, create it
		now := time.Now().UTC()
		store := &KVStore{
			Id:        uuid.New().String(),
			Group:     group,
			Key:       key,
			Value:     valueBytes,
			ExpiredAt: expiredAt,
			CreatedAt: &now,
			UpdatedAt: &now,
			Status:    StatusActive,
		}
		if err := tx.Create(store).Error; err != nil {
			tx.Rollback()
			return false, err
		}
	} else {
		tx.Rollback()
		return false, err
	}

	if err := tx.Commit().Error; err != nil {
		return false, err
	}

	return true, nil
}

// GetSet atomically sets a key to a new value and returns the old value.
// Redis compatible: GETSET key value
func (kv *KV) GetSet(key string, value any, ttl ...time.Duration) (any, error) {
	group := kv.group()

	// Start a transaction
	tx := kv.db.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get the old value
	var oldStore KVStore
	err := tx.Where("\"group\" = ? AND \"key\" = ? AND status = ?", group, key, StatusActive).First(&oldStore).Error
	var oldValue any
	if err == nil {
		// Check if expired
		if oldStore.ExpiredAt == nil || oldStore.ExpiredAt.After(time.Now().UTC()) {
			if err := json.Unmarshal(oldStore.Value, &oldValue); err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return nil, err
	}

	// Set the new value
	valueBytes, err := json.Marshal(value)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	var expiredAt *time.Time
	if len(ttl) > 0 && ttl[0] > 0 {
		t := time.Now().UTC().Add(ttl[0])
		expiredAt = &t
	} else if kv.config != nil && kv.config.DefaultTTL > 0 {
		t := time.Now().UTC().Add(kv.config.DefaultTTL)
		expiredAt = &t
	}

	now := time.Now().UTC()

	if oldStore.Id != "" {
		// Update existing store
		err = tx.Model(&KVStore{}).
			Where("id = ?", oldStore.Id).
			Updates(map[string]any{
				"value":      valueBytes,
				"expired_at": expiredAt,
				"updated_at": &now,
				"status":     StatusActive,
				"deleted_at": nil,
			}).Error
	} else {
		// Create new store
		store := &KVStore{
			Id:        uuid.New().String(),
			Group:     group,
			Key:       key,
			Value:     valueBytes,
			ExpiredAt: expiredAt,
			CreatedAt: &now,
			UpdatedAt: &now,
			Status:    StatusActive,
		}
		err = tx.Create(store).Error
	}

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return oldValue, nil
}

// Del deletes one or more keys.
// Redis compatible: DEL key [key ...]
// Returns the number of keys that were deleted.
func (kv *KV) Del(keys ...string) (int64, error) {
	group := kv.group()

	if len(keys) == 0 {
		return 0, nil
	}

	now := time.Now().UTC()
	result := kv.db.DB.Model(&KVStore{}).
		Where("\"group\" = ? AND \"key\" IN ? AND status = ?", group, keys, StatusActive).
		Updates(map[string]any{
			"status":     StatusDeleted,
			"deleted_at": &now,
			"updated_at": &now,
		})

	return result.RowsAffected, result.Error
}

// TTL returns the time-to-live for a key in seconds.
// Redis compatible: TTL key
// Returns:
//   - -2 if the key does not exist
//   - -1 if the key exists but has no expiration
//   - The TTL in seconds otherwise
func (kv *KV) TTL(key string) (int64, error) {
	group := kv.group()

	var store KVStore
	err := kv.db.DB.Where("\"group\" = ? AND \"key\" = ? AND status = ?", group, key, StatusActive).First(&store).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return -2, nil // Key does not exist
		}
		return 0, err
	}

	if store.ExpiredAt == nil {
		return -1, nil // Key has no expiration
	}

	ttl := time.Until(*store.ExpiredAt)
	if ttl <= 0 {
		return -2, nil // Key is expired
	}

	return int64(ttl.Seconds()), nil
}

// Incr increments the integer value of a key by 1.
// Redis compatible: INCR key
// If the key does not exist, it is set to 0 before performing the operation.
// Returns the new value after the increment.
func (kv *KV) Incr(key string) (int64, error) {
	return kv.IncrBy(key, 1)
}

// IncrBy increments the integer value of a key by the given amount.
// Redis compatible: INCRBY key increment
// If the key does not exist, it is set to 0 before performing the operation.
// Returns the new value after the increment.
func (kv *KV) IncrBy(key string, increment int64) (int64, error) {
	group := kv.group()

	// Start a transaction
	tx := kv.db.DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get the current value
	var store KVStore
	err := tx.Where("\"group\" = ? AND \"key\" = ? AND status = ?", group, key, StatusActive).First(&store).Error

	var currentValue int64 = 0
	var storeExists bool

	if err == nil {
		storeExists = true
		// Check if expired
		if store.ExpiredAt != nil && store.ExpiredAt.Before(time.Now().UTC()) {
			currentValue = 0
		} else {
			// Try to parse as number
			var rawValue any
			if err := json.Unmarshal(store.Value, &rawValue); err != nil {
				tx.Rollback()
				return 0, errors.New("value is not an integer or out of range")
			}

			switch v := rawValue.(type) {
			case float64:
				currentValue = int64(v)
			case string:
				currentValue, err = strconv.ParseInt(v, 10, 64)
				if err != nil {
					tx.Rollback()
					return 0, errors.New("value is not an integer or out of range")
				}
			default:
				tx.Rollback()
				return 0, errors.New("value is not an integer or out of range")
			}
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return 0, err
	}

	// Increment the value
	newValue := currentValue + increment

	// Marshal the new value
	valueBytes, err := json.Marshal(newValue)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	now := time.Now().UTC()

	if storeExists {
		// Update existing store
		err = tx.Model(&KVStore{}).
			Where("id = ?", store.Id).
			Updates(map[string]any{
				"value":      valueBytes,
				"updated_at": &now,
				"status":     StatusActive,
			}).Error
	} else {
		// Create new store
		newStore := &KVStore{
			Id:        uuid.New().String(),
			Group:     group,
			Key:       key,
			Value:     valueBytes,
			CreatedAt: &now,
			UpdatedAt: &now,
			Status:    StatusActive,
		}
		err = tx.Create(newStore).Error
	}

	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return newValue, nil
}

// Expire sets a timeout on a key.
// Redis compatible: EXPIRE key seconds
// Returns true if the timeout was set, false if key does not exist.
func (kv *KV) Expire(key string, ttl time.Duration) (bool, error) {
	group := kv.group()

	expiredAt := time.Now().UTC().Add(ttl)
	now := time.Now().UTC()

	result := kv.db.DB.Model(&KVStore{}).
		Where("\"group\" = ? AND \"key\" = ? AND status = ?", group, key, StatusActive).
		Updates(map[string]any{
			"expired_at": &expiredAt,
			"updated_at": &now,
		})

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

// Persist removes the expiration from a key.
// Redis compatible: PERSIST key
// Returns true if the timeout was removed, false if key does not exist or has no timeout.
func (kv *KV) Persist(key string) (bool, error) {
	group := kv.group()

	now := time.Now().UTC()

	result := kv.db.DB.Model(&KVStore{}).
		Where("\"group\" = ? AND \"key\" = ? AND status = ? AND expired_at IS NOT NULL", group, key, StatusActive).
		Updates(map[string]any{
			"expired_at": nil,
			"updated_at": &now,
		})

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

// Exists checks if one or more keys exist.
// Redis compatible: EXISTS key [key ...]
// Returns the number of keys that exist.
func (kv *KV) Exists(keys ...string) (int64, error) {
	group := kv.group()

	if len(keys) == 0 {
		return 0, nil
	}

	var count int64
	err := kv.db.DB.Model(&KVStore{}).
		Where("\"group\" = ? AND \"key\" IN ? AND status = ? AND (expired_at IS NULL OR expired_at > ?)",
			group, keys, StatusActive, time.Now().UTC()).
		Count(&count).Error

	return count, err
}

// Keys returns all keys matching a pattern in the configured group.
// Note: This is a simplified version - pattern matching is basic.
// Redis compatible: KEYS pattern
func (kv *KV) Keys(pattern string) ([]string, error) {
	group := kv.group()

	var entries []KVStore
	query := kv.db.DB.Where("\"group\" = ? AND status = ? AND (expired_at IS NULL OR expired_at > ?)",
		group, StatusActive, time.Now().UTC())

	if pattern != "" && pattern != "*" {
		// Convert glob pattern to SQL LIKE pattern
		sqlPattern := globToSQL(pattern)
		query = query.Where("\"key\" LIKE ?", sqlPattern)
	}

	err := query.Select("key").Find(&entries).Error
	if err != nil {
		return nil, err
	}

	keys := make([]string, len(entries))
	for i, e := range entries {
		keys[i] = e.Key
	}

	return keys, nil
}
