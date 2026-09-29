package router

import (
	"strings"

	"github.com/awesome-goose/goose/types"
)

// wrapWithPrefix nests routes under the segments of prefix, producing a chain
// of single-segment parent routes. Example: prefix "apps/identity" wrapping
// [GET /] yields [{Path: "apps", Children: [{Path: "identity", Children: [GET /]}]}].
//
// The router matcher walks one URL segment at a time, so prefixes containing
// "/" must be exploded into nested Children — a single Route at Path
// "apps/identity" would otherwise never match a URL like /apps/identity/...
// because no single URL segment equals "apps/identity".
//
// middlewares, if given, are attached to the innermost wrapping node — the
// one immediately above `routes` itself ("identity" in the example above),
// not the outermost one ("apps"). See Mount's doc comment for why: that
// segment is the one guaranteed not to be shared with a sibling Mount call,
// so it can't have its Middlewares silently dropped by AppendRoutes' merge.
func wrapWithPrefix(prefix string, routes types.Routes, middlewares []types.Middleware) types.Routes {
	trimmed := strings.Trim(prefix, "/")
	if trimmed == "" {
		if len(middlewares) == 0 {
			return routes
		}
		out := make(types.Routes, len(routes))
		for i, r := range routes {
			r.Middlewares = appendMiddlewares(middlewares, r.Middlewares)
			out[i] = r
		}
		return out
	}
	segments := strings.Split(trimmed, "/")
	current := routes
	for i := len(segments) - 1; i >= 0; i-- {
		node := types.Route{Path: segments[i], Children: current}
		if i == len(segments)-1 {
			node.Middlewares = middlewares
		}
		current = types.Routes{node}
	}
	return current
}

// staticRouter is the module returned by ForRoutes. It owns its own slice of
// routes and registers them with the kernel during Boot. Each ForRoutes call
// returns a fresh instance — there is no cross-call singleton — so importing
// the same module twice in different parents no longer leaks routes between
// them.
//
// staticRouter implements both types.Router (for compatibility with code that
// expects the legacy Router shape) and types.Module + Bootable so it can be
// included directly in another module's Imports().
type staticRouter struct {
	routes      types.Routes
	prefix      string             // optional — applied during Boot when non-empty
	middlewares []types.Middleware // optional — applied during Boot via Mount
}

// Routes returns the routes this module owns. When a prefix has been applied
// (via Mount), the routes are wrapped in a chain of nested parent routes —
// one Route per "/"-separated segment of the prefix — so the kernel's
// segment-by-segment matcher can descend into them, and any Mount-supplied
// middlewares are attached to that chain (see wrapWithPrefix).
func (s *staticRouter) Routes() (types.Routes, error) {
	return wrapWithPrefix(s.prefix, s.routes, s.middlewares), nil
}

// Imports returns no sub-modules: staticRouter is a leaf in the module tree.
func (s *staticRouter) Imports() []types.Module { return nil }

// Exports declares nothing for the DI container.
func (s *staticRouter) Exports() []any { return nil }

// Declarations declares nothing for the DI container.
func (s *staticRouter) Declarations() []any { return nil }

// Boot pushes the module's routes into the kernel. Mirrors what the old
// singleton routerModule did, but scoped to a single call's worth of routes.
func (s *staticRouter) Boot(k types.Kernel) error {
	routes, err := s.Routes()
	if err != nil {
		return err
	}
	_, err = k.AppendRoutes(routes...)
	return err
}
