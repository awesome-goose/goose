package types

type WithMiddleware interface {
	Middleware() []Middleware
}

type WithResource[T any] interface {
	Get(ctx *Context) T
	List(ctx *Context) []T
	Create(ctx *Context) T
	Update(ctx *Context) T
	Delete(ctx *Context) T
}
