package cli

import (
	"os"

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
	if err := fn(context); err != nil {
		return err
	}

	// A nonzero Output.Code() (e.g. output.ConsoleError(...).WithExitCode(1))
	// signals command failure; translate it into a real process exit code so
	// `$?` reflects it. Code 0 (the default for Console/ConsoleSuccess/etc.)
	// falls through normally so callers can still run their own deferred
	// cleanup after goose.Start(...) returns.
	if code := context.response.Code(); code != 0 {
		os.Exit(code)
	}

	return nil
}

func (a *App) Shutdown() error {
	// CLI apps don't have a long-running server to shutdown
	return nil
}
