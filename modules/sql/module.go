package sql

import (
	"fmt"
	"time"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/types"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type sqlModule struct {
	config *Config
	db     *Db

	isRoot bool
}

func NewModule(config *Config, isRoot bool) *sqlModule {
	return &sqlModule{config: config, isRoot: isRoot}
}

func (m *sqlModule) Imports() []types.Module {
	return []types.Module{}
}

func (m *sqlModule) Exports() []any {
	return []any{
		Query{},
	}
}

func (m *sqlModule) Declarations() []any {
	return []any{
		Query{},
	}
}

// Configure registers infrastructure dependencies before declarations are created.
// This ensures *Db is available for injection into Entity structs.
func (m *sqlModule) Configure(container types.Container) error {
	if m.isRoot {
		container.Register(
			func() *Config {
				return m.config
			},
			"",
			true,
		)

		var log types.Log
		err := container.Resolve(&log, "")
		if err != nil {
			return errors.ErrFailedToResolveLog.WithError(err)
		}

		db := m.initialize(log)
		m.db = db

		container.Register(func() *Db {
			return db
		}, "", true)
	}
	return nil
}

func (m *sqlModule) Boot(k types.Kernel) error {
	var db *Db
	container := k.Container()

	// For root module, use the already-initialized db
	// For child modules, resolve from container
	if m.isRoot {
		db = m.db
	} else {
		if err := container.Resolve(&db, ""); err != nil {
			return errors.ErrFailedToResolveDatabase.WithError(err)
		}
	}

	if db == nil {
		panic(errors.ErrNoDatabaseConfigured.Error())
	}

	runner := &Runner{db}
	runner.CreateTableSchema()

	for _, migration := range m.config.Migrations {
		err := runner.Run(migration)
		if err != nil {
			panic(errors.ErrFailedToRunMigration.WithError(err).Error())
		}
	}

	for _, seeder := range m.config.Seeders {
		err := runner.Run(seeder)
		if err != nil {
			panic(errors.ErrFailedToRunSeeder.WithError(err).Error())
		}
	}

	return nil
}

func (m *sqlModule) initialize(log types.Log) *Db {
	dialect := m.config.Dialect
	if dialect == "" {
		dialect = "postgres"
	}

	var dialector gorm.Dialector
	var tablePrefix string

	switch dialect {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			m.config.User,
			m.config.Pass,
			m.config.Host,
			m.config.Port,
			m.config.Name,
		)
		dialector = mysql.Open(dsn)
		tablePrefix = ""
	case "sqlite":
		dialector = sqlite.Open(m.config.Name)
		tablePrefix = ""
	case "postgres":
		fallthrough
	default:
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s search_path=%s TimeZone=%s",
			m.config.Host,
			m.config.User,
			m.config.Pass,
			m.config.Name,
			m.config.Port,
			m.config.SSLMode,
			m.config.Schema,
			m.config.TimeZone,
		)
		dialector = postgres.Open(dsn)
		tablePrefix = m.config.Schema + "."
	}

	var logMod gormLogger.LogLevel = gormLogger.Warn
	if m.config.Log {
		logMod = gormLogger.Info
	}

	gormConfig := &gorm.Config{
		Logger: gormLogger.Default.LogMode(logMod),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   tablePrefix,
			SingularTable: false,
		},
		SkipDefaultTransaction: true,
	}

	if m.config.TimeZone != "" {
		gormConfig.NowFunc = func() time.Time {
			loc, err := time.LoadLocation(m.config.TimeZone)
			if err != nil {
				return time.Now()
			}
			return time.Now().In(loc)
		}
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		panic(err)
	}

	sqlDb, err := db.DB()
	if err != nil {
		panic(err)
	}

	err = sqlDb.Ping()
	if err != nil {
		panic(err)
	}

	return &Db{DB: db, log: log}
}
