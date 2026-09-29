package router

import (
	"sync"

	"github.com/awesome-goose/goose/types"
)

// Group constructs a single Route at `prefix` whose Children are the supplied
// routes. Use it to compose nested routes inline without spinning up an
// extra Module:
//
//	router.ForRoutes(
//	    router.Group("apps/identity",
//	        router.Get("/", []any{AppController{}, "Health"}),
//	        router.Get("/now", []any{AppController{}, "Now"}),
//	    ),
//	)
func Group(prefix string, routes ...types.Route) types.Route {
	return types.Route{Path: prefix, Children: routes}
}

// Mount wraps a Module so that every route registered by it or anything in
// its Imports() subtree is nested under `prefix`. This is the Goose
// equivalent of NestJS's RouterModule.register([{ path: prefix, module }]).
//
// middlewares, when given, run for every route in the mounted subtree — the
// one place a caller can attach something like authentication to an entire
// app (or an entire tree of apps) without editing every individual route
// declaration inside it. They are applied to the innermost path segment of
// `prefix` (the one immediately above the mounted routes themselves, e.g.
// "widgets" in "apps/widgets"), which is always unique to this Mount call:
// AppendRoutes (core/kernel.go) merges two route nodes that share a Method
// and Path by combining their Children only, not their Middlewares, so two
// sibling Mount calls sharing an outer segment (both under "apps", say)
// would silently lose whichever one's Middlewares didn't win that merge if
// they were attached there instead. Nothing about a plain Mount call with no
// middlewares changes.
//
// Behavior:
//   - If the target is a *staticRouter or *dynamicRouters (the modules
//     returned by ForRoutes / ForRouters), Mount returns a clone with the
//     prefix (and middlewares) applied at Boot.
//   - Otherwise Mount returns a wrapping Module that recursively visits the
//     target's Imports(), substituting any nested router modules with their
//     prefixed clones. DI declarations from the inner module are forwarded
//     unchanged so controllers, services, and entities still resolve
//     normally.
//
// Each call returns a fresh wrapping Module; importing the same target with
// two different prefixes registers two independent prefixed copies.
func Mount(prefix string, mod types.Module, middlewares ...types.Middleware) types.Module {
	switch m := mod.(type) {
	case *staticRouter:
		return &staticRouter{routes: m.routes, prefix: prefix, middlewares: appendMiddlewares(m.middlewares, middlewares)}
	case *dynamicRouters:
		return &dynamicRouters{routers: m.routers, prefix: prefix, middlewares: appendMiddlewares(m.middlewares, middlewares)}
	default:
		return &mountedModule{prefix: prefix, inner: mod, middlewares: middlewares}
	}
}

// appendMiddlewares combines a router's own already-set middlewares (from an
// earlier Mount call wrapping the same target) with newly-supplied ones,
// without mutating either input slice.
func appendMiddlewares(existing, add []types.Middleware) []types.Middleware {
	if len(existing) == 0 {
		return add
	}
	if len(add) == 0 {
		return existing
	}
	out := make([]types.Middleware, 0, len(existing)+len(add))
	out = append(out, existing...)
	out = append(out, add...)
	return out
}

// mountedModule wraps an arbitrary Module so its route-registering children
// are prefixed. It forwards DI declarations from the inner module and
// recurses into the inner's Imports(), wrapping each sub-module so the
// prefix (and any middlewares) propagate through the entire subtree.
type mountedModule struct {
	prefix      string
	inner       types.Module
	middlewares []types.Middleware

	once    sync.Once
	imports []types.Module
}

func (m *mountedModule) Imports() []types.Module {
	m.once.Do(func() {
		subs := m.inner.Imports()
		m.imports = make([]types.Module, 0, len(subs))
		for _, sub := range subs {
			m.imports = append(m.imports, Mount(m.prefix, sub, m.middlewares...))
		}
	})
	return m.imports
}

// Exports / Declarations forward through to the inner module unchanged so
// the host kernel still sees the same DI surface as if the inner module had
// been imported directly.
func (m *mountedModule) Exports() []any      { return m.inner.Exports() }
func (m *mountedModule) Declarations() []any { return m.inner.Declarations() }

// Boot forwards to the inner module's Boot when it implements Bootable. The
// inner's own Boot would normally register routes flat at the kernel; since
// we replaced every route-registering child via Mount, the inner is left
// only with non-route side effects, which we want to run as-is.
func (m *mountedModule) Boot(k types.Kernel) error {
	if b, ok := m.inner.(types.Bootable); ok {
		return b.Boot(k)
	}
	return nil
}

// Shutdown mirrors Boot — forwarded when the inner module declares it.
func (m *mountedModule) Shutdown(k types.Kernel) error {
	if s, ok := m.inner.(types.Shutdownable); ok {
		return s.Shutdown(k)
	}
	return nil
}
