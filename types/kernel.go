package types

type Kernel interface {
	Start(instances ...*Instance) (stop func() error, err error)

	Routes() []Route
	AppendRoutes(routes ...Route) ([]Route, error)
	Container() Container
	Registry() Registry
}
