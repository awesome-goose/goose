package tests

import (
	"testing"

	"github.com/awesome-goose/goose/modules/sql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type countingMigration struct {
	sql.BaseMigration
	runs *int
}

func (m *countingMigration) Run(q *sql.Query) error { *m.runs++; return nil }

func TestSQLMigrateRunsOnceAndRecords(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	ms := []sql.Migration{&countingMigration{runs: &n}}
	for i := 0; i < 3; i++ {
		if err := sql.Migrate(db, ms); err != nil {
			t.Fatal(err)
		}
	}
	if n != 1 {
		t.Fatalf("a recorded migration ran %d times", n)
	}
}
