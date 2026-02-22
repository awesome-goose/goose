package types

type Platform interface {
	// Type returns the platform type (api, web, cli)
	Type() PlatformType
	// Name returns the platform instance name
	Name() string
	// Boot initializes and returns the app
	Boot(container Container) (App, error)
}
