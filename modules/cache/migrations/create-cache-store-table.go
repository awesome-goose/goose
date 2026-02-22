package migrations

import (
	"github.com/awesome-goose/goose/modules/sql"
)

// CreateCacheStoreTable creates the cache_store table for caching
type CreateCacheStoreTable struct {
	sql.BaseMigration
}

// Run executes the migration to create the cache_store table
func (m *CreateCacheStoreTable) Run(q *sql.Query) error {
	_, err := q.Exec(`
		CREATE TABLE IF NOT EXISTS cache_store (
			id UUID PRIMARY KEY NOT NULL,
			created_at TIMESTAMP(6) NOT NULL,
			updated_at TIMESTAMP(6) NOT NULL,
			deleted_at TIMESTAMP(6),
			"group" VARCHAR(255) NOT NULL,
			"key" VARCHAR(255) NOT NULL,
			value JSONB,
			expired_at TIMESTAMP(6),
			status VARCHAR(255) NOT NULL DEFAULT 'active'
		);
		
		CREATE INDEX IF NOT EXISTS idx_cache_store_created_at ON cache_store(created_at);
		CREATE INDEX IF NOT EXISTS idx_cache_store_updated_at ON cache_store(updated_at);
		CREATE INDEX IF NOT EXISTS idx_cache_store_deleted_at ON cache_store(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_cache_store_expired_at ON cache_store(expired_at);
		CREATE INDEX IF NOT EXISTS idx_cache_store_status ON cache_store(status);
		CREATE INDEX IF NOT EXISTS idx_cache_group_key ON cache_store("group", "key");
	`)
	return err
}
