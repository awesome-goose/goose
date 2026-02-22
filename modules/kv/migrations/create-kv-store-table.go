package migrations

import (
	"github.com/awesome-goose/goose/modules/sql"
)

// CreateKVStoreTable creates the kv_store table for key-value storage
type CreateKVStoreTable struct {
	sql.BaseMigration
}

// Run executes the migration to create the kv_store table
func (m *CreateKVStoreTable) Run(q *sql.Query) error {
	_, err := q.Exec(`
		CREATE TABLE IF NOT EXISTS kv_store (
			id UUID PRIMARY KEY NOT NULL,
			created_at TIMESTAMP(6) NOT NULL,
			updated_at TIMESTAMP(6) NOT NULL,
			deleted_at TIMESTAMP(6),
			"group" VARCHAR(255) NOT NULL,
			"key" VARCHAR(255) NOT NULL,
			value JSONB,
			expired_at TIMESTAMP(6),
			meta JSONB,
			status VARCHAR(255) NOT NULL DEFAULT 'active'
		);
		
		CREATE INDEX IF NOT EXISTS idx_kv_store_created_at ON kv_store(created_at);
		CREATE INDEX IF NOT EXISTS idx_kv_store_updated_at ON kv_store(updated_at);
		CREATE INDEX IF NOT EXISTS idx_kv_store_deleted_at ON kv_store(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_kv_store_expired_at ON kv_store(expired_at);
		CREATE INDEX IF NOT EXISTS idx_kv_store_status ON kv_store(status);
		CREATE INDEX IF NOT EXISTS idx_kv_group_key ON kv_store("group", "key");
	`)
	return err
}
