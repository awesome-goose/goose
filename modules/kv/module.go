package kv

import (
	"github.com/awesome-goose/goose/modules/kv/migrations"
	"github.com/awesome-goose/goose/modules/sql"
	"github.com/awesome-goose/goose/types"
)

// kvModule implements the types.Module interface
type kvModule struct {
	config *Config
	isRoot bool
}

// NewModule creates a new KV module
func NewModule(config *Config, isRoot bool) *kvModule {
	if config == nil {
		config = &Config{}
	}
	return &kvModule{config: config, isRoot: isRoot}
}

func (m *kvModule) Imports() []types.Module {
	return []types.Module{
		sql.Child(&sql.Config{
			Migrations: migrations.Migrations(),
		}),
	}
}

func (m *kvModule) Exports() []any {
	return []any{
		&KV{},
	}
}

func (m *kvModule) Declarations() []any {
	return []any{
		&KV{},
	}
}

func (m *kvModule) Boot(k types.Kernel) error {
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

		if m.config.CleanupInterval > 0 {
			go startCleanup(db, m.config.CleanupInterval)
		}
	}

	return nil
}
