package queues

import (
	"time"

	"github.com/awesome-goose/goose/errors"
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
		// *Queue is only made constructible via Declarations() (tracked by
		// the registry's own declarationIndex), not via container.Register,
		// so it must be looked up through the registry rather than
		// container.Resolve. See BUGS.md #2 for the cron module's version of
		// this same mistake.
		info, err := k.Registry().Get(&Queue{})
		if err != nil {
			return err
		}

		queue, ok := info.Instance.(*Queue)
		if !ok {
			return errors.ErrDeclarationTypeMismatch.WithMeta("*queues.Queue")
		}

		// Initialize and start processing for each handler
		queue.Initialize()
		for _, handler := range m.handlers {
			queue.Process(handler)
		}
	}

	return nil
}
