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

func (r *Runner) Run(m Runnable) error {
	name := fmt.Sprintf("%T", m)

	// Check if this migration/seeder has already been executed
	var count int64
	if err := r.db.Model(&MigrationRecord{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return errors.ErrFailedToCheckMigrationStatus.WithError(err)
	}
	if count > 0 {
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
