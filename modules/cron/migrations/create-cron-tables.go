package migrations

import (
	"github.com/awesome-goose/goose/modules/sql"
)

// CreateCronTables creates the CronJobs and CronLogs tables
type CreateCronTables struct {
	sql.BaseMigration
}

// Run executes the migration to create all cron tables
func (m *CreateCronTables) Run(q *sql.Query) error {
	// Create CronJobs table
	_, err := q.Exec(`
		CREATE TABLE IF NOT EXISTS "CronJobs" (
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

		CREATE INDEX IF NOT EXISTS idx_cron_jobs_created_at ON "CronJobs"(created_at);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_updated_at ON "CronJobs"(updated_at);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_deleted_at ON "CronJobs"(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_status ON "CronJobs"(status);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_group ON "CronJobs"("group");
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_name ON "CronJobs"(name);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_priority ON "CronJobs"(priority);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_start_at ON "CronJobs"(start_at);
		CREATE INDEX IF NOT EXISTS idx_cron_jobs_expire_at ON "CronJobs"(expire_at);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_cron_job_group_name ON "CronJobs"("group", name);
	`)
	if err != nil {
		return err
	}

	// Create CronLogs table
	_, err = q.Exec(`
		CREATE TABLE IF NOT EXISTS "CronLogs" (
			id UUID PRIMARY KEY NOT NULL,
			created_at TIMESTAMP(6),
			updated_at TIMESTAMP(6),
			deleted_at TIMESTAMP(6),
			job_id UUID NOT NULL,
			output JSONB,
			status VARCHAR(255)
		);

		CREATE INDEX IF NOT EXISTS idx_cron_logs_created_at ON "CronLogs"(created_at);
		CREATE INDEX IF NOT EXISTS idx_cron_logs_updated_at ON "CronLogs"(updated_at);
		CREATE INDEX IF NOT EXISTS idx_cron_logs_deleted_at ON "CronLogs"(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_cron_logs_job_id ON "CronLogs"(job_id);
		CREATE INDEX IF NOT EXISTS idx_cron_logs_status ON "CronLogs"(status);
	`)

	return err
}
