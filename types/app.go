package types

type App interface {
	// Run starts the app with the provided request handler
	Run(fn func(c Context) error) error
	// Shutdown gracefully stops the app
	Shutdown() error
}
