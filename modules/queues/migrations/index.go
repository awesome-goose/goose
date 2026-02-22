package migrations

import "github.com/awesome-goose/goose/modules/sql"

// Migrations returns all Queues migrations in order
func Migrations() []sql.Migration {
	return []sql.Migration{
		&CreateQueueTables{},
	}
}
