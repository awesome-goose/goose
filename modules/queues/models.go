package queues

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// QueueQueue represents a queue in the database
type QueueQueue struct {
	Id        string          `gorm:"primaryKey;column:id;type:uuid;not null"`
	CreatedAt *time.Time      `gorm:"index;column:created_at;type:timestamp(6)"`
	UpdatedAt *time.Time      `gorm:"index;column:updated_at;type:timestamp(6)"`
	DeletedAt *time.Time      `gorm:"index;column:deleted_at;type:timestamp(6)"`
	Name      string          `gorm:"column:name;type:varchar(255);not null;uniqueIndex:idx_queue_name"`
	Priority  int             `gorm:"column:priority;type:integer;default:0"`
	Config    json.RawMessage `gorm:"column:config;type:jsonb"`
	Status    string          `gorm:"index;column:status;type:varchar(255);default:'active'"`
}

// TableName returns the table name for QueueQueue
func (QueueQueue) TableName() string {
	return "QueueQueues"
}

// BeforeCreate hook to set defaults
func (e *QueueQueue) BeforeCreate(tx *gorm.DB) error {
	if e.Id == "" {
		e.Id = uuid.New().String()
	}
	now := time.Now().UTC()
	if e.CreatedAt == nil {
		e.CreatedAt = &now
	}
	e.UpdatedAt = &now
	if e.Status == "" {
		e.Status = QueueStatusActive
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (e *QueueQueue) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now().UTC()
	e.UpdatedAt = &now
	return nil
}

// QueueJob represents a job in the database
type QueueJob struct {
	Id         string          `gorm:"primaryKey;column:id;type:uuid;not null"`
	CreatedAt  *time.Time      `gorm:"index;column:created_at;type:timestamp(6)"`
	UpdatedAt  *time.Time      `gorm:"index;column:updated_at;type:timestamp(6)"`
	DeletedAt  *time.Time      `gorm:"index;column:deleted_at;type:timestamp(6)"`
	QueueId    string          `gorm:"index;column:queue_id;type:uuid;not null"`
	Name       string          `gorm:"index;column:name;type:varchar(255);not null"`
	Priority   int             `gorm:"index;column:priority;type:integer;default:0"`
	Data       json.RawMessage `gorm:"column:data;type:jsonb"`
	Config     json.RawMessage `gorm:"column:config;type:jsonb"`
	RetryLimit int             `gorm:"column:retry_limit;type:integer;default:0"`
	RetryDelay int             `gorm:"column:retry_delay;type:integer;default:0"`
	RetryCount int             `gorm:"column:retry_count;type:integer;default:0"`
	StartAt    *time.Time      `gorm:"index;column:start_at;type:timestamp(6)"`
	StartedAt  *time.Time      `gorm:"column:started_at;type:timestamp(6)"`
	ExpireAt   *time.Time      `gorm:"index;column:expire_at;type:timestamp(6)"`
	ExpiredAt  *time.Time      `gorm:"column:expired_at;type:timestamp(6)"`
	Status     string          `gorm:"index;column:status;type:varchar(255);default:'new'"`

	// Relations
	Queue *QueueQueue `gorm:"foreignKey:QueueId;references:Id"`
}

// TableName returns the table name for QueueJob
func (QueueJob) TableName() string {
	return "QueueJobs"
}

// BeforeCreate hook to set defaults
func (e *QueueJob) BeforeCreate(tx *gorm.DB) error {
	if e.Id == "" {
		e.Id = uuid.New().String()
	}
	now := time.Now().UTC()
	if e.CreatedAt == nil {
		e.CreatedAt = &now
	}
	e.UpdatedAt = &now
	if e.Status == "" {
		e.Status = JobStatusNew
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (e *QueueJob) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now().UTC()
	e.UpdatedAt = &now
	return nil
}

// QueueLog represents a job execution log in the database
type QueueLog struct {
	Id        string          `gorm:"primaryKey;column:id;type:uuid;not null"`
	CreatedAt *time.Time      `gorm:"index;column:created_at;type:timestamp(6)"`
	UpdatedAt *time.Time      `gorm:"index;column:updated_at;type:timestamp(6)"`
	DeletedAt *time.Time      `gorm:"index;column:deleted_at;type:timestamp(6)"`
	QueueId   string          `gorm:"column:queue_id;type:uuid;not null"`
	JobId     string          `gorm:"index;column:job_id;type:uuid;not null"`
	Output    json.RawMessage `gorm:"column:output;type:jsonb"`
	Status    string          `gorm:"index;column:status;type:varchar(255)"`

	// Relations
	Queue *QueueQueue `gorm:"foreignKey:QueueId;references:Id"`
	Job   *QueueJob   `gorm:"foreignKey:JobId;references:Id"`
}

// TableName returns the table name for QueueLog
func (QueueLog) TableName() string {
	return "QueueLogs"
}

// BeforeCreate hook to set defaults
func (e *QueueLog) BeforeCreate(tx *gorm.DB) error {
	if e.Id == "" {
		e.Id = uuid.New().String()
	}
	now := time.Now().UTC()
	if e.CreatedAt == nil {
		e.CreatedAt = &now
	}
	e.UpdatedAt = &now
	return nil
}

// BeforeUpdate hook to update timestamp
func (e *QueueLog) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now().UTC()
	e.UpdatedAt = &now
	return nil
}
