package types

type Middleware interface {
	Handle(ctx Context) error
}

type Middlewares []Middleware
