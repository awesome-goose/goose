package cache

import (
	"github.com/awesome-goose/goose/modules/cache/migrations"
	"github.com/awesome-goose/goose/modules/sql"
	"github.com/awesome-goose/goose/types"
)

// cacheModule implements the types.Module interface
type cacheModule struct {
	config *Config
	isRoot bool
}

// NewModule creates a new Cache module
func NewModule(config *Config, isRoot bool) *cacheModule {
	if config == nil {
		config = &Config{}
	}
	return &cacheModule{config: config, isRoot: isRoot}
}

func (m *cacheModule) Imports() []types.Module {
	return []types.Module{
		sql.Child(&sql.Config{
			Migrations: migrations.Migrations(),
		}),
	}
}

func (m *cacheModule) Exports() []any {
	return []any{
		&Cache{},
	}
}

func (m *cacheModule) Declarations() []any {
	return []any{
		&Cache{},
	}
}

func (m *cacheModule) Boot(k types.Kernel) error {
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
