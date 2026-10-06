package router

import (
	"sync"

	"github.com/awesome-goose/goose/types"
)

// Silent imports a module for its services and entities only: every router in
// its Imports() subtree is replaced by an empty one, so none of its routes are
// registered. Use it when an app depends on another app's DI surface (a queue
// service, an entity) but must not re-expose that app's API under its own
// prefix:
//
//	&queueapp.AppModule{}              // services AND the queue API, under this app
//	router.Silent(&queueapp.AppModule{}) // services only
//
// Declarations, exports and Boot/Shutdown forward unchanged, exactly as for
// Mount; only route registration is removed.
func Silent(mod types.Module) types.Module {
	switch mod.(type) {
	case *staticRouter, *dynamicRouters:
		return &staticRouter{}
	}
	return &silentModule{inner: mod}
}

type silentModule struct {
	inner types.Module

	once    sync.Once
	imports []types.Module
}

func (m *silentModule) Imports() []types.Module {
	m.once.Do(func() {
		subs := m.inner.Imports()
		m.imports = make([]types.Module, 0, len(subs))
		for _, sub := range subs {
			m.imports = append(m.imports, Silent(sub))
		}
	})
	return m.imports
}

func (m *silentModule) Exports() []any      { return m.inner.Exports() }
func (m *silentModule) Declarations() []any { return m.inner.Declarations() }

func (m *silentModule) Boot(k types.Kernel) error {
	if b, ok := m.inner.(types.Bootable); ok {
		return b.Boot(k)
	}
	return nil
}

func (m *silentModule) Shutdown(k types.Kernel) error {
	if s, ok := m.inner.(types.Shutdownable); ok {
		return s.Shutdown(k)
	}
	return nil
}
