package migrations

import (
	"github.com/awesome-goose/goose/modules/sql"
)

// CreateKVStoreTable creates the KVStore table for key-value storage
type CreateKVStoreTable struct {
	sql.BaseMigration
}

// Run executes the migration to create the KVStore table
func (m *CreateKVStoreTable) Run(q *sql.Query) error {
	_, err := q.Exec(`
		CREATE TABLE IF NOT EXISTS "KVStore" (
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

		CREATE INDEX IF NOT EXISTS idx_kv_store_created_at ON "KVStore"(created_at);
		CREATE INDEX IF NOT EXISTS idx_kv_store_updated_at ON "KVStore"(updated_at);
		CREATE INDEX IF NOT EXISTS idx_kv_store_deleted_at ON "KVStore"(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_kv_store_expired_at ON "KVStore"(expired_at);
		CREATE INDEX IF NOT EXISTS idx_kv_store_status ON "KVStore"(status);
		CREATE INDEX IF NOT EXISTS idx_kv_group_key ON "KVStore"("group", "key");
	`)
	return err
}
