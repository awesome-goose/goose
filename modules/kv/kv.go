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
	"gorm.io/gorm/clause"
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

// lockRow returns the one row for (group, key), locked until tx ends. A key
// that has no row yet gets a deleted placeholder first (ON CONFLICT DO NOTHING
// on the unique ("group", "key")), so there is always a row to lock: a second
// caller waits on the first one's insert or lock instead of reading a stale
// value or creating a second row (U-G11). SQLite serializes write transactions
// and rejects FOR UPDATE, so the hint is skipped there.
func lockRow(tx *gorm.DB, group, key string) (*KVStore, error) {
	now := time.Now().UTC()
	placeholder := &KVStore{
		Id:        uuid.New().String(),
		Group:     group,
		Key:       key,
		CreatedAt: &now,
		UpdatedAt: &now,
		DeletedAt: &now,
		Status:    StatusDeleted,
	}
	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "group"}, {Name: "key"}},
		DoNothing: true,
	}).Create(placeholder).Error
	if err != nil {
		return nil, err
	}

	var store KVStore
	query := tx.Where("\"group\" = ? AND \"key\" = ?", group, key)
	if tx.Dialector.Name() != "sqlite" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.First(&store).Error; err != nil {
		return nil, err
	}
	return &store, nil
}

// live reports whether a locked row holds a value: active and not expired.
func (e *KVStore) live() bool {
	return e.Status == StatusActive && (e.ExpiredAt == nil || e.ExpiredAt.After(time.Now().UTC()))
}

// expiry is the expiry a write with ttl gives a key: ttl, else the default TTL.
func (kv *KV) expiry(ttl []time.Duration) *time.Time {
	if len(ttl) > 0 && ttl[0] > 0 {
		t := time.Now().UTC().Add(ttl[0])
		return &t
	}
	if kv.config != nil && kv.config.DefaultTTL > 0 {
		t := time.Now().UTC().Add(kv.config.DefaultTTL)
		return &t
	}
	return nil
}

// write stores value in the locked row and makes it active.
func write(tx *gorm.DB, store *KVStore, value []byte, expiredAt *time.Time) error {
	now := time.Now().UTC()
	return tx.Model(&KVStore{}).
		Where("id = ?", store.Id).
		Updates(map[string]any{
			"value":      value,
			"expired_at": expiredAt,
			"updated_at": &now,
			"status":     StatusActive,
			"deleted_at": nil,
		}).Error
}

// Set stores a value for a key with optional TTL.
// Redis compatible: SET key value [EX seconds]
func (kv *KV) Set(key string, value any, ttl ...time.Duration) error {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	expiredAt := kv.expiry(ttl)
	return kv.db.DB.Transaction(func(tx *gorm.DB) error {
		store, err := lockRow(tx, kv.group(), key)
		if err != nil {
			return err
		}
		return write(tx, store, valueBytes, expiredAt)
	})
}

// SetNX sets a key only if it does not already exist.
// Redis compatible: SETNX key value
// Returns true if key was set, false if key already exists.
func (kv *KV) SetNX(key string, value any, ttl ...time.Duration) (bool, error) {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	expiredAt := kv.expiry(ttl)
	set := false
	err = kv.db.DB.Transaction(func(tx *gorm.DB) error {
		store, err := lockRow(tx, kv.group(), key)
		if err != nil {
			return err
		}
		if store.live() {
			return nil
		}
		set = true
		return write(tx, store, valueBytes, expiredAt)
	})
	if err != nil {
		return false, err
	}
	return set, nil
}

// GetSet atomically sets a key to a new value and returns the old value.
// Redis compatible: GETSET key value
func (kv *KV) GetSet(key string, value any, ttl ...time.Duration) (any, error) {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	expiredAt := kv.expiry(ttl)
	var oldValue any
	err = kv.db.DB.Transaction(func(tx *gorm.DB) error {
		store, err := lockRow(tx, kv.group(), key)
		if err != nil {
			return err
		}
		if store.live() {
			if err := json.Unmarshal(store.Value, &oldValue); err != nil {
				return err
			}
		}
		return write(tx, store, valueBytes, expiredAt)
	})
	if err != nil {
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

// errNotInteger is returned when IncrBy finds a value that is not an integer.
var errNotInteger = errors.New("value is not an integer or out of range")

// IncrBy increments the integer value of a key by the given amount.
// Redis compatible: INCRBY key increment
// If the key does not exist (or is deleted or expired), it is set to 0 before
// performing the operation and has no expiry; a live key keeps its expiry.
// The row is locked for the read and the write, so concurrent increments from
// any number of goroutines or processes are exact.
// Returns the new value after the increment.
func (kv *KV) IncrBy(key string, increment int64) (int64, error) {
	var newValue int64
	err := kv.db.DB.Transaction(func(tx *gorm.DB) error {
		store, err := lockRow(tx, kv.group(), key)
		if err != nil {
			return err
		}

		var current int64
		expiredAt := store.ExpiredAt
		if store.live() {
			var raw any
			if err := json.Unmarshal(store.Value, &raw); err != nil {
				return errNotInteger
			}
			switch v := raw.(type) {
			case float64:
				current = int64(v)
			case string:
				current, err = strconv.ParseInt(v, 10, 64)
				if err != nil {
					return errNotInteger
				}
			default:
				return errNotInteger
			}
		} else {
			expiredAt = nil
		}

		newValue = current + increment
		valueBytes, err := json.Marshal(newValue)
		if err != nil {
			return err
		}
		return write(tx, store, valueBytes, expiredAt)
	})
	if err != nil {
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
