package migrations

import (
	"github.com/awesome-goose/goose/modules/sql"
)

// CreateCacheStoreTable creates the CacheStore table for caching
type CreateCacheStoreTable struct {
	sql.BaseMigration
}

// Run executes the migration to create the CacheStore table
func (m *CreateCacheStoreTable) Run(q *sql.Query) error {
	_, err := q.Exec(`
		CREATE TABLE IF NOT EXISTS "CacheStore" (
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

		CREATE INDEX IF NOT EXISTS idx_cache_store_created_at ON "CacheStore"(created_at);
		CREATE INDEX IF NOT EXISTS idx_cache_store_updated_at ON "CacheStore"(updated_at);
		CREATE INDEX IF NOT EXISTS idx_cache_store_deleted_at ON "CacheStore"(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_cache_store_expired_at ON "CacheStore"(expired_at);
		CREATE INDEX IF NOT EXISTS idx_cache_store_status ON "CacheStore"(status);
		CREATE INDEX IF NOT EXISTS idx_cache_group_key ON "CacheStore"("group", "key");
	`)
	return err
}
