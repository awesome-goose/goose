package types

type Middleware interface {
	Handle(ctx Context) error
}

type Middlewares []Middleware

// AfterMiddleware is optionally implemented by a Middleware that also needs to
// see the handler's output: After runs once the handler has returned, innermost
// middleware first, and may return the same output, a replacement, or an error.
type AfterMiddleware interface {
	After(ctx Context, out Output) (Output, error)
}

// Respond is an error a Middleware returns to answer the request itself instead
// of failing it or continuing to the handler: the kernel writes Output and stops.
// Use it for a replayed idempotent response, a 429, or a redirect.
type Respond struct {
	Output Output
}

func (r *Respond) Error() string { return "middleware responded" }
