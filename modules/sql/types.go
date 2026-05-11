package sql

import (
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Config struct {
	Dialect  string
	Host     string
	Port     int
	User     string
	Pass     string
	Name     string
	Sync     bool
	Log      bool
	SSLMode  string
	Schema   string
	TimeZone string

	Seeders    []Seeder
	Migrations []Migration
}

type Option func(*Config)

func WithDialect(dialect string) Option {
	return func(c *Config) {
		c.Dialect = dialect
	}
}

func WithHost(host string) Option {
	return func(c *Config) {
		c.Host = host
	}
}

func WithPort(port int) Option {
	return func(c *Config) {
		c.Port = port
	}
}

func WithUser(user string) Option {
	return func(c *Config) {
		c.User = user
	}
}

func WithPass(pass string) Option {
	return func(c *Config) {
		c.Pass = pass
	}
}

func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

func WithSync(sync bool) Option {
	return func(c *Config) {
		c.Sync = sync
	}
}

func WithLog(log bool) Option {
	return func(c *Config) {
		c.Log = log
	}
}

func WithSSLMode(sslmode string) Option {
	return func(c *Config) {
		c.SSLMode = sslmode
	}
}

func WithSchema(schema string) Option {
	return func(c *Config) {
		c.Schema = schema
	}
}

func WithTimeZone(timezone string) Option {
	return func(c *Config) {
		c.TimeZone = timezone
	}
}

func WithSeeders(seeders []Seeder) Option {
	return func(c *Config) {
		c.Seeders = seeders
	}
}

func WithMigrations(migrations []Migration) Option {
	return func(c *Config) {
		c.Migrations = migrations
	}
}

type Runnable interface {
	Run(*Query) error
}

type Seeder interface {
	Runnable
	isSeeder()
}

type Migration interface {
	Runnable
	isMigration()
}

// BaseSeeder can be embedded in seeders to satisfy the Seeder interface
type BaseSeeder struct{}

func (BaseSeeder) isSeeder() {}

// BaseMigration can be embedded in migrations to satisfy the Migration interface
type BaseMigration struct{}

func (BaseMigration) isMigration() {}

// MigrationRecordType represents the type of migration record
type MigrationRecordType string

const (
	MigrationRecordTypeMigration MigrationRecordType = "migration"
	MigrationRecordTypeSeeder    MigrationRecordType = "seeder"
)

// MigrationRecord represents a record in the migrations table
type MigrationRecord struct {
	ID         uint                `gorm:"primaryKey;autoIncrement"`
	Name       string              `gorm:"size:255;not null;uniqueIndex"`
	Type       MigrationRecordType `gorm:"size:50;not null"`
	ExecutedAt time.Time           `gorm:"autoCreateTime"`
}

// TableName returns the table name for MigrationRecord
func (MigrationRecord) TableName() string {
	return "Migrations"
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// toSnakeCase converts a string to snake_case
func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// toName converts a struct name to a table name (snake_case, pluralized)
func toName(name string) string {
	snake := toSnakeCase(name)

	// Simple pluralization rules
	if strings.HasSuffix(snake, "y") {
		return snake[:len(snake)-1] + "ies"
	}
	if strings.HasSuffix(snake, "s") || strings.HasSuffix(snake, "x") ||
		strings.HasSuffix(snake, "ch") || strings.HasSuffix(snake, "sh") {
		return snake + "es"
	}
	return snake + "s"
}

type UUIDAware struct {
	Id string `gorm:"primaryKey;column:id;type:string;size:36;not null" json:"id,omitempty" binding:"-"`
}

type TimeAware struct {
	CreatedAt *time.Time `gorm:"index;column:created_at;not null" json:"created_at,omitempty" binding:"-"`
	UpdatedAt *time.Time `gorm:"index;column:updated_at;not null" json:"updated_at,omitempty" binding:"-"`
}

type SoftDeleteAware struct {
	DeletedAt gorm.DeletedAt `gorm:"index;column:deleted_at" json:"deleted_at,omitempty" binding:"-"`
}

// Hook and morph lifecycle constants
const (
	BeforeCreate = "beforeCreate"
	AfterCreate  = "afterCreate"
	BeforeUpdate = "beforeUpdate"
	AfterUpdate  = "afterUpdate"
	BeforeDelete = "beforeDelete"
	AfterDelete  = "afterDelete"
	BeforeRead   = "beforeRead"
	AfterRead    = "afterRead"
)
