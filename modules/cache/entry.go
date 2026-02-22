package cache

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CacheStore represents a cache entry in the database
type CacheStore struct {
	Id        string          `gorm:"primaryKey;column:id;type:uuid;not null"`
	CreatedAt *time.Time      `gorm:"index;column:created_at;type:timestamp(6);not null"`
	UpdatedAt *time.Time      `gorm:"index;column:updated_at;type:timestamp(6);not null"`
	DeletedAt *time.Time      `gorm:"index;column:deleted_at;type:timestamp(6)"`
	Group     string          `gorm:"column:group;type:varchar(255);not null;index:idx_cache_group_key"`
	Key       string          `gorm:"column:key;type:varchar(255);not null;index:idx_cache_group_key"`
	Value     json.RawMessage `gorm:"column:value;type:jsonb"`
	ExpiredAt *time.Time      `gorm:"index;column:expired_at;type:timestamp(6)"`
	Status    string          `gorm:"index;column:status;type:varchar(255);not null;default:'active'"`
}

// TableName returns the table name for CacheStore
func (CacheStore) TableName() string {
	return "cache_store"
}

// BeforeCreate hook to set defaults
func (e *CacheStore) BeforeCreate(tx *gorm.DB) error {
	if e.Id == "" {
		e.Id = uuid.New().String()
	}
	now := time.Now().UTC()
	if e.CreatedAt == nil {
		e.CreatedAt = &now
	}
	e.UpdatedAt = &now
	if e.Status == "" {
		e.Status = StatusActive
	}
	if e.Group == "" {
		e.Group = DefaultGroup
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (e *CacheStore) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now().UTC()
	e.UpdatedAt = &now
	return nil
}
