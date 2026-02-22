package migrations

import (
	"github.com/awesome-goose/goose/modules/sql"
)

// CreateCronTables creates the cron_jobs and cron_logs tables
type CreateCronTables struct {
	sql.BaseMigration
}

// Run executes the migration to create all cron tables
func (m *CreateCronTables) Run(q *sql.Query) error {
	// Create cron_jobs table
	_, err := q.Exec(`
		CREATE TABLE IF NOT EXISTS cron_jobs (
			id UUID PRIMARY KEY NOT NULL,
			created_at TIMESTAMP(6),
			updated_at TIMESTAMP(6),
			deleted_at TIMESTAMP(6),
			"group" VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			pattern VARCHAR(255) NOT NULL,
			priority INTEGER DEFAULT 0,
			config JSONB,
			retry_limit INTEGER DEFAULT 0,
			retry_delay INTEGER DEFAULT 0,
			start_at TIMESTAMP(6),
			started_at TIMESTAMP(6),
			expire_at TIMESTAMP(6),
			expired_at TIMESTAMP(6),
			status VARCHAR(255) DEFAULT 'pending'
		);
		
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_created_at ON cron_jobs(created_at);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_updated_at ON cron_jobs(updated_at);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_deleted_at ON cron_jobs(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_status ON cron_jobs(status);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_group ON cron_jobs("group");
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_name ON cron_jobs(name);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_priority ON cron_jobs(priority);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_start_at ON cron_jobs(start_at);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_expire_at ON cron_jobs(expire_at);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_cron_job_group_name ON cron_jobs("group", name);
	`)
	if err != nil {
		return err
	}

	// Create cron_logs table
	_, err = q.Exec(`
		CREATE TABLE IF NOT EXISTS cron_logs (
			id UUID PRIMARY KEY NOT NULL,
			created_at TIMESTAMP(6),
			updated_at TIMESTAMP(6),
			deleted_at TIMESTAMP(6),
			job_id UUID NOT NULL,
			output JSONB,
			status VARCHAR(255)
		);
		
		CREATE INDEX IF NOT EXISTS idx_cron_logs_created_at ON cron_logs(created_at);
		CREATE INDEX IF NOT EXISTS idx_cron_logs_updated_at ON cron_logs(updated_at);
		CREATE INDEX IF NOT EXISTS idx_cron_logs_deleted_at ON cron_logs(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_cron_logs_job_id ON cron_logs(job_id);
		CREATE INDEX IF NOT EXISTS idx_cron_logs_status ON cron_logs(status);
	`)

	return err
}
