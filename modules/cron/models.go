package cron

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CronJob represents a cron job in the database
type CronJob struct {
	Id         string          `gorm:"primaryKey;column:id;type:uuid;not null"`
	CreatedAt  *time.Time      `gorm:"index;column:created_at;type:timestamp(6)"`
	UpdatedAt  *time.Time      `gorm:"index;column:updated_at;type:timestamp(6)"`
	DeletedAt  *time.Time      `gorm:"index;column:deleted_at;type:timestamp(6)"`
	Group      string          `gorm:"column:group;type:varchar(255);not null;uniqueIndex:idx_cron_job_group_name,priority:1"`
	Name       string          `gorm:"column:name;type:varchar(255);not null;uniqueIndex:idx_cron_job_group_name,priority:2"`
	Pattern    string          `gorm:"column:pattern;type:varchar(255);not null"`
	Priority   int             `gorm:"index;column:priority;type:integer;default:0"`
	Config     json.RawMessage `gorm:"column:config;type:jsonb"`
	RetryLimit int             `gorm:"column:retry_limit;type:integer;default:0"`
	RetryDelay int             `gorm:"column:retry_delay;type:integer;default:0"`
	StartAt    *time.Time      `gorm:"index;column:start_at;type:timestamp(6)"`
	StartedAt  *time.Time      `gorm:"column:started_at;type:timestamp(6)"`
	ExpireAt   *time.Time      `gorm:"index;column:expire_at;type:timestamp(6)"`
	ExpiredAt  *time.Time      `gorm:"column:expired_at;type:timestamp(6)"`
	Status     string          `gorm:"index;column:status;type:varchar(255);default:'pending'"`
}

// TableName returns the table name for CronJob
func (CronJob) TableName() string {
	return "CronJobs"
}

// BeforeCreate hook to set defaults
func (e *CronJob) BeforeCreate(tx *gorm.DB) error {
	if e.Id == "" {
		e.Id = uuid.New().String()
	}
	now := time.Now().UTC()
	if e.CreatedAt == nil {
		e.CreatedAt = &now
	}
	e.UpdatedAt = &now
	if e.Status == "" {
		e.Status = JobStatusPending
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (e *CronJob) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now().UTC()
	e.UpdatedAt = &now
	return nil
}

// GetConfig unmarshals the job config into the given type
func (e *CronJob) GetConfig(dest any) error {
	if e.Config == nil {
		return nil
	}
	return json.Unmarshal(e.Config, dest)
}

// GetJobConfig is a helper to get typed config from a job
func GetJobConfig[T any](job *CronJob) (T, error) {
	var result T
	if job.Config == nil {
		return result, nil
	}
	err := json.Unmarshal(job.Config, &result)
	return result, err
}

// CronLog represents a cron job execution log in the database
type CronLog struct {
	Id        string          `gorm:"primaryKey;column:id;type:uuid;not null"`
	CreatedAt *time.Time      `gorm:"index;column:created_at;type:timestamp(6)"`
	UpdatedAt *time.Time      `gorm:"index;column:updated_at;type:timestamp(6)"`
	DeletedAt *time.Time      `gorm:"index;column:deleted_at;type:timestamp(6)"`
	JobId     string          `gorm:"index;column:job_id;type:uuid;not null"`
	Output    json.RawMessage `gorm:"column:output;type:jsonb"`
	Status    string          `gorm:"index;column:status;type:varchar(255)"`

	// Relations
	Job *CronJob `gorm:"foreignKey:JobId;references:Id"`
}

// TableName returns the table name for CronLog
func (CronLog) TableName() string {
	return "CronLogs"
}

// BeforeCreate hook to set defaults
func (e *CronLog) BeforeCreate(tx *gorm.DB) error {
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
func (e *CronLog) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now().UTC()
	e.UpdatedAt = &now
	return nil
}

// GetOutput unmarshals the log output into the given type
func (e *CronLog) GetOutput(dest any) error {
	if e.Output == nil {
		return nil
	}
	return json.Unmarshal(e.Output, dest)
}
