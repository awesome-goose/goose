package sql

import (
	"fmt"
	"gorm.io/gorm"
	"time"

	"github.com/awesome-goose/goose/errors"
)

type Runner struct {
	db *Db
}

func (r *Runner) CreateTableSchema() error {
	return r.db.AutoMigrate(&MigrationRecord{})
}

// Named lets a migration or seeder choose the name it is recorded under.
// Without it the name is fmt.Sprintf("%T", m) ("*migrations.CreateUsers"),
// which carries the package name but not its path, so two libraries with a
// `migrations` package and the same type name collide and the second never
// runs. Changing a recorded name makes it run again, so existing ones keep
// their %T name; give new ones (and new versions of reference data) a
// qualified name such as "wallet/seed-currencies@2".
type Named interface {
	MigrationName() string
}

// Repeatable marks a seeder (or migration) that runs on every boot instead
// of once per database: idempotent reference data that must reach databases
// seeded before it changed. It is still run in a transaction, and recorded
// the first time.
type Repeatable interface {
	Repeat() bool
}

// RecordName is the name m is recorded under.
func RecordName(m Runnable) string {
	if n, ok := m.(Named); ok && n.MigrationName() != "" {
		return n.MigrationName()
	}
	return fmt.Sprintf("%T", m)
}

func (r *Runner) Run(m Runnable) error {
	name := RecordName(m)
	repeat := false
	if rp, ok := m.(Repeatable); ok {
		repeat = rp.Repeat()
	}

	// Check if this migration/seeder has already been executed
	var count int64
	if err := r.db.Model(&MigrationRecord{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return errors.ErrFailedToCheckMigrationStatus.WithError(err)
	}
	if count > 0 && !repeat {
		// Already executed, skip
		return nil
	}

	query := &Query{db: r.db}
	return query.Tx(func(query *Query) error {
		err := m.Run(&Query{db: &Db{DB: query.db.DB, log: r.db.log}})
		if err != nil {
			return err
		}

		var recordType MigrationRecordType
		switch m.(type) {
		case Seeder:
			recordType = MigrationRecordTypeSeeder
		case Migration:
			recordType = MigrationRecordTypeMigration
		default:
			return errors.ErrUnknownMigrationType
		}

		if count > 0 {
			return nil // a Repeatable that already has its record
		}
		return query.db.Create(&MigrationRecord{
			Name:       name,
			Type:       recordType,
			ExecutedAt: time.Now(),
		}).Error
	})
}

// Migrate runs migrations against db, skipping those already recorded. It is
// the same bookkeeping the module uses at boot, so a host that needs its
// tables before the module has booted can call it first and the module's own
// pass will find every record in place.
func Migrate(db *gorm.DB, migrations []Migration) error {
	r := &Runner{db: &Db{DB: db}}
	if err := r.CreateTableSchema(); err != nil {
		return err
	}
	for _, m := range migrations {
		if err := r.Run(m); err != nil {
			return err
		}
	}
	return nil
}
