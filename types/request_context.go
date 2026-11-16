package types

import "context"

// ContextCarrier is the optional interface of a Context or Request that has a
// per-request context.Context: one that is cancelled when the client goes away
// (and, on the api and web platforms, when the server shuts down), and that
// carries the request's deadline. The api and web requests implement it; so
// does testing.MockContext. It is optional so that a Context or Request
// written before it existed still satisfies its interface.
type ContextCarrier interface {
	Context() context.Context
}

// RequestContext returns the request's context.Context: pass it to anything
// that should stop when the request does (a database query, an outgoing HTTP
// call, a queue push). It asks the Context first, then its Request, and falls
// back to context.Background() on a platform with no per-request context (the
// cli), so it never returns nil.
//
// A handler does not need to call it: a DTO field of type context.Context is
// bound to the same value.
func RequestContext(c Context) context.Context {
	if c == nil {
		return context.Background()
	}
	if cc, ok := c.(ContextCarrier); ok {
		if ctx := cc.Context(); ctx != nil {
			return ctx
		}
	}
	if rq := c.Request(); rq != nil {
		if cc, ok := rq.(ContextCarrier); ok {
			if ctx := cc.Context(); ctx != nil {
				return ctx
			}
		}
	}
	return context.Background()
}
