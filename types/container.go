package types

type Container interface {
	Register(resolver any, name string, singleton bool) error
	Resolve(abstraction any, name string) error
	Create(value any) (any, error)
	Close() error
}

type RegisterAware interface {
	OnRegister()
}

type ResolveAware interface {
	OnResolve()
}
type CloseAware interface {
	OnClose() error
}
