package types

type Kernel interface {
	Start(platform Platform, module Module, initializers []func(container Container) error) (stop func() error, err error)

	Routes() []Route
	AppendRoutes(routes ...Route) ([]Route, error)
	Container() Container
	Registry() Registry
}
