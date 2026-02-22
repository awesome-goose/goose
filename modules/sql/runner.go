package sql

import (
	"fmt"
	"time"
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
		return fmt.Errorf("failed to check migration status: %w", err)
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
			return fmt.Errorf("unknown migration type")
		}

		return query.db.Create(&MigrationRecord{
			Name:       name,
			Type:       recordType,
			ExecutedAt: time.Now(),
		}).Error
	})
}
