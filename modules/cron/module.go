package cron

import (
	"context"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/modules/cron/migrations"
	"github.com/awesome-goose/goose/modules/sql"
	"github.com/awesome-goose/goose/types"
)

// cronModule implements the types.Module interface
type cronModule struct {
	config   *Config
	handlers []*CronHandler
	isRoot   bool
}

// NewModule creates a new Cron module
func NewModule(config *Config, handlers []*CronHandler, isRoot bool) *cronModule {
	if config == nil {
		config = DefaultConfig()
	}
	return &cronModule{config: config, handlers: handlers, isRoot: isRoot}
}

func (m *cronModule) Imports() []types.Module {
	return []types.Module{
		sql.Child(&sql.Config{
			Migrations: migrations.Migrations(),
		}),
	}
}

func (m *cronModule) Exports() []any {
	return []any{
		&Cron{},
	}
}

func (m *cronModule) Declarations() []any {
	return []any{
		&Cron{},
	}
}

func (m *cronModule) Boot(k types.Kernel) error {
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
		if m.config.CleanupInterval > 0 || m.config.EnableStaleJobRecovery {
			go startCleanup(db, m.config)
		}
	}

	// Start cron runner with registered handlers
	if len(m.handlers) > 0 {
		// *Cron is only made constructible via Declarations() (tracked by the
		// registry's own declarationIndex), not via container.Register, so it
		// must be looked up through the registry rather than container.Resolve.
		info, err := k.Registry().Get(&Cron{})
		if err != nil {
			return err
		}

		cronService, ok := info.Instance.(*Cron)
		if !ok {
			return errors.ErrDeclarationTypeMismatch.WithMeta("*cron.Cron")
		}

		// Start cron runner in background
		go func() {
			err := cronService.Start(context.Background(), m.handlers)
			if err != nil && err != context.Canceled {
				cronService.log.Error("Cron runner stopped with error: " + err.Error())
			}
		}()
	}

	return nil
}
