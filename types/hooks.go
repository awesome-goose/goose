package types

// Configurable is called during module registration, before declarations are created.
// Use this to register infrastructure dependencies (e.g., database connections)
// that will be injected into declarations.
type Configurable interface {
	Configure(c Container) error
}

type Bootable interface {
	Boot(k Kernel) error
}

type Shutdownable interface {
	Shutdown(k Kernel) error
}
