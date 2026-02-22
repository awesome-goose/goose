package migrations

import "github.com/awesome-goose/goose/modules/sql"

// Migrations returns all Cache migrations in order
func Migrations() []sql.Migration {
	return []sql.Migration{
		&CreateCacheStoreTable{},
	}
}
