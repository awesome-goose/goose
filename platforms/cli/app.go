package cli

import (
	"github.com/awesome-goose/goose/types"
)

type App struct {
	config *Config
}

func NewApp(config *Config) *App {
	return &App{config: config}
}

func (a *App) Run(fn func(c types.Context) error) error {
	context := NewContext()
	return fn(context)
}

func (a *App) Shutdown() error {
	// CLI apps don't have a long-running server to shutdown
	return nil
}
