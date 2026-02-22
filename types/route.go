package types

type Route struct {
	Method      Method
	Path        string
	Handler     any
	Middlewares Middlewares
	Children    Routes
}

func (r *Route) HasChildren() bool {
	return len(r.Children) > 0
}

func (r *Route) IsMethod(method string) bool {
	return r.Method.Is(method)
}

func (r *Route) IsPath(path string) bool {
	return r.Path == path
}

func (r *Route) IsEmpty() bool {
	return r.Path == "" && r.Method == ""
}

func (r *Route) Equals(other *Route) bool {
	return r.Method == other.Method && r.Path == other.Path
}

func (r *Route) Clone() Route {
	clone := *r
	clone.Middlewares = make(Middlewares, len(r.Middlewares))
	copy(clone.Middlewares, r.Middlewares)
	clone.Children = make([]Route, len(r.Children))
	copy(clone.Children, r.Children)
	return clone
}

func (r *Route) AddChildren(children ...Route) *Route {
	r.Children = append(r.Children, children...)
	return r
}

func (r *Route) AddMiddlewares(middlewares ...Middleware) *Route {
	r.Middlewares = append(r.Middlewares, middlewares...)
	return r
}

func (r *Route) SetHandler(handler func(Context) any) *Route {
	r.Handler = handler
	return r
}

func (r *Route) SetPath(path string) *Route {
	r.Path = path
	return r
}

func (r *Route) SetMethod(method Method) *Route {
	r.Method = method
	return r
}

type Routes = []Route
