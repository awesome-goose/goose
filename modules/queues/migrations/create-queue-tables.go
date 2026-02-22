package migrations

import (
	"github.com/awesome-goose/goose/modules/sql"
)

// CreateQueueTables creates the queue_queues, queue_jobs, and queue_logs tables
type CreateQueueTables struct {
	sql.BaseMigration
}

// Run executes the migration to create all queue tables
func (m *CreateQueueTables) Run(q *sql.Query) error {
	// Create queue_queues table
	_, err := q.Exec(`
		CREATE TABLE IF NOT EXISTS queue_queues (
			id UUID PRIMARY KEY NOT NULL,
			created_at TIMESTAMP(6),
			updated_at TIMESTAMP(6),
			deleted_at TIMESTAMP(6),
			name VARCHAR(255) NOT NULL,
			priority INTEGER DEFAULT 0,
			config JSONB,
			status VARCHAR(255) DEFAULT 'active'
		);
		
		CREATE INDEX IF NOT EXISTS idx_queue_queues_created_at ON queue_queues(created_at);
		CREATE INDEX IF NOT EXISTS idx_queue_queues_updated_at ON queue_queues(updated_at);
		CREATE INDEX IF NOT EXISTS idx_queue_queues_deleted_at ON queue_queues(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_queue_queues_status ON queue_queues(status);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_queue_name ON queue_queues(name);
	`)
	if err != nil {
		return err
	}

	// Create queue_jobs table
	_, err = q.Exec(`
		CREATE TABLE IF NOT EXISTS queue_jobs (
			id UUID PRIMARY KEY NOT NULL,
			created_at TIMESTAMP(6),
			updated_at TIMESTAMP(6),
			deleted_at TIMESTAMP(6),
			queue_id UUID NOT NULL,
			name VARCHAR(255) NOT NULL,
			priority INTEGER DEFAULT 0,
			data JSONB,
			config JSONB,
			retry_limit INTEGER DEFAULT 0,
			retry_delay INTEGER DEFAULT 0,
			retry_count INTEGER DEFAULT 0,
			start_at TIMESTAMP(6),
			started_at TIMESTAMP(6),
			expire_at TIMESTAMP(6),
			expired_at TIMESTAMP(6),
			status VARCHAR(255) DEFAULT 'new'
		);
		
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_created_at ON queue_jobs(created_at);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_updated_at ON queue_jobs(updated_at);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_deleted_at ON queue_jobs(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_queue_id ON queue_jobs(queue_id);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_name ON queue_jobs(name);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_priority ON queue_jobs(priority);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_start_at ON queue_jobs(start_at);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_expire_at ON queue_jobs(expire_at);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_status ON queue_jobs(status);
	`)
	if err != nil {
		return err
	}

	// Create queue_logs table
	_, err = q.Exec(`
		CREATE TABLE IF NOT EXISTS queue_logs (
			id UUID PRIMARY KEY NOT NULL,
			created_at TIMESTAMP(6),
			updated_at TIMESTAMP(6),
			deleted_at TIMESTAMP(6),
			queue_id UUID NOT NULL,
			job_id UUID NOT NULL,
			output JSONB,
			status VARCHAR(255)
		);
		
		CREATE INDEX IF NOT EXISTS idx_queue_logs_created_at ON queue_logs(created_at);
		CREATE INDEX IF NOT EXISTS idx_queue_logs_updated_at ON queue_logs(updated_at);
		CREATE INDEX IF NOT EXISTS idx_queue_logs_deleted_at ON queue_logs(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_queue_logs_job_id ON queue_logs(job_id);
		CREATE INDEX IF NOT EXISTS idx_queue_logs_status ON queue_logs(status);
	`)

	return err
}
