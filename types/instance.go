package types

// PlatformType identifies the type of platform
type PlatformType string

const (
	PlatformTypeAPI PlatformType = "api"
	PlatformTypeWeb PlatformType = "web"
	PlatformTypeCLI PlatformType = "cli"
)

// Instance represents a single app instance with its platform, module, and initializers.
// Multiple instances can be run together in a multi-platform application.
type Instance struct {
	Name         string
	Type         PlatformType
	Platform     Platform
	Module       Module
	Initializers []func(container Container) error
}

// InstanceOption is a functional option for configuring an Instance
type InstanceOption func(*Instance)

// WithName sets the instance name
func WithName(name string) InstanceOption {
	return func(i *Instance) {
		i.Name = name
	}
}

// RunnableApp extends App with lifecycle management
type RunnableApp interface {
	App
	// Shutdown gracefully shuts down the app
	Shutdown() error
}
