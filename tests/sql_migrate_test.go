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

type namedMigration struct {
	sql.BaseMigration
	name string
	runs *int
}

func (m *namedMigration) Run(q *sql.Query) error { *m.runs++; return nil }
func (m *namedMigration) MigrationName() string  { return m.name }

type repeatMigration struct {
	sql.BaseMigration
	runs *int
}

func (m *repeatMigration) Run(q *sql.Query) error { *m.runs++; return nil }
func (m *repeatMigration) Repeat() bool           { return true }

// Two migrations of the same Go type (two libraries' *migrations.X) collide
// under %T; a MigrationName keeps them apart. A Repeatable runs every time.
func TestSQLMigrateNamedAndRepeatable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	a, b, r := 0, 0, 0
	ms := []sql.Migration{&namedMigration{name: "liba/create", runs: &a}, &namedMigration{name: "libb/create", runs: &b}, &repeatMigration{runs: &r}}
	for i := 0; i < 3; i++ {
		if err := sql.Migrate(db, ms); err != nil {
			t.Fatal(err)
		}
	}
	if a != 1 || b != 1 || r != 3 {
		t.Fatalf("runs a=%d b=%d repeat=%d, want 1 1 3", a, b, r)
	}
	var n int64
	db.Model(&sql.MigrationRecord{}).Where("name = ?", sql.RecordName(&repeatMigration{})).Count(&n)
	if n != 1 {
		t.Fatalf("the repeatable was recorded %d times", n)
	}
}
