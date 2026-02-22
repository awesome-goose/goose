package kv

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// KVStore represents a key-value store in the database
type KVStore struct {
	Id        string          `gorm:"primaryKey;column:id;type:uuid;not null"`
	CreatedAt *time.Time      `gorm:"index;column:created_at;type:timestamp(6);not null"`
	UpdatedAt *time.Time      `gorm:"index;column:updated_at;type:timestamp(6);not null"`
	DeletedAt *time.Time      `gorm:"index;column:deleted_at;type:timestamp(6)"`
	Group     string          `gorm:"column:group;type:varchar(255);not null;index:idx_kv_group_key"`
	Key       string          `gorm:"column:key;type:varchar(255);not null;index:idx_kv_group_key"`
	Value     json.RawMessage `gorm:"column:value;type:jsonb"`
	ExpiredAt *time.Time      `gorm:"index;column:expired_at;type:timestamp(6)"`
	Meta      json.RawMessage `gorm:"column:meta;type:jsonb"`
	Status    string          `gorm:"index;column:status;type:varchar(255);not null;default:'active'"`
}

// TableName returns the table name for KVStore
func (KVStore) TableName() string {
	return "kv_store"
}

// BeforeCreate hook to set defaults
func (e *KVStore) BeforeCreate(tx *gorm.DB) error {
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
func (e *KVStore) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now().UTC()
	e.UpdatedAt = &now
	return nil
}
