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
// Behavior:
//   - If the target is a *staticRouter or *dynamicRouters (the modules
//     returned by ForRoutes / ForRouters), Mount returns a clone with the
//     prefix applied at Boot.
//   - Otherwise Mount returns a wrapping Module that recursively visits the
//     target's Imports(), substituting any nested router modules with their
//     prefixed clones. DI declarations from the inner module are forwarded
//     unchanged so controllers, services, and entities still resolve
//     normally.
//
// Each call returns a fresh wrapping Module; importing the same target with
// two different prefixes registers two independent prefixed copies.
func Mount(prefix string, mod types.Module) types.Module {
	switch m := mod.(type) {
	case *staticRouter:
		return &staticRouter{routes: m.routes, prefix: prefix}
	case *dynamicRouters:
		return &dynamicRouters{routers: m.routers, prefix: prefix}
	default:
		return &mountedModule{prefix: prefix, inner: mod}
	}
}

// mountedModule wraps an arbitrary Module so its route-registering children
// are prefixed. It forwards DI declarations from the inner module and
// recurses into the inner's Imports(), wrapping each sub-module so the
// prefix propagates through the entire subtree.
type mountedModule struct {
	prefix string
	inner  types.Module

	once    sync.Once
	imports []types.Module
}

func (m *mountedModule) Imports() []types.Module {
	m.once.Do(func() {
		subs := m.inner.Imports()
		m.imports = make([]types.Module, 0, len(subs))
		for _, sub := range subs {
			m.imports = append(m.imports, Mount(m.prefix, sub))
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
