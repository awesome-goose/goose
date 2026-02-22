package queues

import (
	"time"

	"github.com/awesome-goose/goose/modules/queues/migrations"
	"github.com/awesome-goose/goose/modules/sql"
	"github.com/awesome-goose/goose/types"
)

// queuesModule implements the types.Module interface
type queuesModule struct {
	config   *Config
	handlers []*JobHandler
	isRoot   bool
}

// NewModule creates a new Queues module
func NewModule(config *Config, handlers []*JobHandler, isRoot bool) *queuesModule {
	if config == nil {
		config = &Config{}
	}
	return &queuesModule{config: config, handlers: handlers, isRoot: isRoot}
}

func (m *queuesModule) Imports() []types.Module {
	return []types.Module{
		sql.Child(&sql.Config{
			Migrations: migrations.Migrations(),
		}),
	}
}

func (m *queuesModule) Exports() []any {
	return []any{
		&Queue{},
	}
}

func (m *queuesModule) Declarations() []any {
	return []any{
		&Queue{},
	}
}

func (m *queuesModule) Boot(k types.Kernel) error {
	container := k.Container()

	if m.isRoot {
		container.Register(
			func() *Config {
				return m.config
			},
			"",
			true,
		)

		var db *sql.Db
		err := container.Resolve(&db, "")
		if err != nil {
			return err
		}

		// Start cleanup goroutine if configured
		if m.config.CleanupInterval > 0 {
			retentionPeriod := m.config.RetentionPeriod
			if retentionPeriod == 0 {
				retentionPeriod = 7 * 24 * time.Hour
			}
			go startCleanup(db, m.config.CleanupInterval, retentionPeriod)
		}
	}

	// Start handlers if any are registered
	if len(m.handlers) > 0 {
		var queue *Queue
		err := container.Resolve(&queue, "")
		if err != nil {
			return err
		}

		// Initialize and start processing for each handler
		queue.Initialize()
		for _, handler := range m.handlers {
			queue.Process(handler)
		}
	}

	return nil
}
