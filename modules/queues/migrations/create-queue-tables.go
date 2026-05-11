package migrations

import (
	"github.com/awesome-goose/goose/modules/sql"
)

// CreateQueueTables creates the QueueQueues, QueueJobs, and QueueLogs tables
type CreateQueueTables struct {
	sql.BaseMigration
}

// Run executes the migration to create all queue tables
func (m *CreateQueueTables) Run(q *sql.Query) error {
	// Create QueueQueues table
	_, err := q.Exec(`
		CREATE TABLE IF NOT EXISTS "QueueQueues" (
			id UUID PRIMARY KEY NOT NULL,
			created_at TIMESTAMP(6),
			updated_at TIMESTAMP(6),
			deleted_at TIMESTAMP(6),
			name VARCHAR(255) NOT NULL,
			priority INTEGER DEFAULT 0,
			config JSONB,
			status VARCHAR(255) DEFAULT 'active'
		);

		CREATE INDEX IF NOT EXISTS idx_queue_queues_created_at ON "QueueQueues"(created_at);
		CREATE INDEX IF NOT EXISTS idx_queue_queues_updated_at ON "QueueQueues"(updated_at);
		CREATE INDEX IF NOT EXISTS idx_queue_queues_deleted_at ON "QueueQueues"(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_queue_queues_status ON "QueueQueues"(status);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_queue_name ON "QueueQueues"(name);
	`)
	if err != nil {
		return err
	}

	// Create QueueJobs table
	_, err = q.Exec(`
		CREATE TABLE IF NOT EXISTS "QueueJobs" (
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

		CREATE INDEX IF NOT EXISTS idx_queue_jobs_created_at ON "QueueJobs"(created_at);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_updated_at ON "QueueJobs"(updated_at);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_deleted_at ON "QueueJobs"(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_queue_id ON "QueueJobs"(queue_id);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_name ON "QueueJobs"(name);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_priority ON "QueueJobs"(priority);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_start_at ON "QueueJobs"(start_at);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_expire_at ON "QueueJobs"(expire_at);
		CREATE INDEX IF NOT EXISTS idx_queue_jobs_status ON "QueueJobs"(status);
	`)
	if err != nil {
		return err
	}

	// Create QueueLogs table
	_, err = q.Exec(`
		CREATE TABLE IF NOT EXISTS "QueueLogs" (
			id UUID PRIMARY KEY NOT NULL,
			created_at TIMESTAMP(6),
			updated_at TIMESTAMP(6),
			deleted_at TIMESTAMP(6),
			queue_id UUID NOT NULL,
			job_id UUID NOT NULL,
			output JSONB,
			status VARCHAR(255)
		);

		CREATE INDEX IF NOT EXISTS idx_queue_logs_created_at ON "QueueLogs"(created_at);
		CREATE INDEX IF NOT EXISTS idx_queue_logs_updated_at ON "QueueLogs"(updated_at);
		CREATE INDEX IF NOT EXISTS idx_queue_logs_deleted_at ON "QueueLogs"(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_queue_logs_job_id ON "QueueLogs"(job_id);
		CREATE INDEX IF NOT EXISTS idx_queue_logs_status ON "QueueLogs"(status);
	`)

	return err
}
