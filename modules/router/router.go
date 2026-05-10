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
func wrapWithPrefix(prefix string, routes types.Routes) types.Routes {
	trimmed := strings.Trim(prefix, "/")
	if trimmed == "" {
		return routes
	}
	segments := strings.Split(trimmed, "/")
	current := routes
	for i := len(segments) - 1; i >= 0; i-- {
		current = types.Routes{{Path: segments[i], Children: current}}
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
	routes types.Routes
	prefix string // optional — applied during Boot when non-empty
}

// Routes returns the routes this module owns. When a prefix has been applied
// (via Mount), the routes are wrapped in a chain of nested parent routes —
// one Route per "/"-separated segment of the prefix — so the kernel's
// segment-by-segment matcher can descend into them.
func (s *staticRouter) Routes() (types.Routes, error) {
	return wrapWithPrefix(s.prefix, s.routes), nil
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
