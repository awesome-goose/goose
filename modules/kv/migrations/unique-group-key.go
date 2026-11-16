package migrations

import (
	"github.com/awesome-goose/goose/modules/sql"
)

// UniqueGroupKey makes ("group", "key") unique, so a key has one row that
// IncrBy, SetNX, GetSet and Set lock (U-G11). Earlier versions could create
// two rows for one key when two first writes raced; of those, the active row
// updated last is kept and the others are removed.
type UniqueGroupKey struct {
	sql.BaseMigration
}

func (m *UniqueGroupKey) MigrationName() string { return "kv/unique-group-key" }

func (m *UniqueGroupKey) Run(q *sql.Query) error {
	_, err := q.Exec(`
		DELETE FROM "KVStore" WHERE id IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (
					PARTITION BY "group", "key"
					ORDER BY CASE WHEN status = 'active' THEN 0 ELSE 1 END, updated_at DESC, id DESC
				) AS rn FROM "KVStore"
			) ranked WHERE rn > 1
		);

		DROP INDEX IF EXISTS idx_kv_group_key;
		CREATE UNIQUE INDEX IF NOT EXISTS idx_kv_group_key_unique ON "KVStore"("group", "key");
	`)
	return err
}
