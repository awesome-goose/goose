package router

import (
	"errors"
	"sync"

	"github.com/awesome-goose/goose/types"
)

type routerModule struct {
	routers   []types.Router
	processed map[types.Router]bool
	mu        sync.RWMutex
}

func (m *routerModule) Imports() []types.Module {
	return []types.Module{}
}

func (m *routerModule) Exports() []any {
	return []any{}
}

func (m *routerModule) Declarations() []any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]any, len(m.routers))

	for i, r := range m.routers {
		result[i] = r
	}

	return result
}

func (m *routerModule) Boot(k types.Kernel) error {
	m.mu.Lock()
	if m.processed == nil {
		m.processed = make(map[types.Router]bool)
	}
	m.mu.Unlock()

	for _, router := range m.routers {
		m.mu.RLock()
		alreadyProcessed := m.processed[router]
		m.mu.RUnlock()

		if alreadyProcessed {
			continue
		}

		if _, ok := router.(*staticRouter); !ok {
			resolvedRouter, err := k.Registry().Get(router)
			if err != nil {
				return err
			}

			router, ok = resolvedRouter.Instance.(types.Router)
			if !ok {
				return errors.New("invalid router instance")
			}
		}

		routes, err := router.Routes()
		if err != nil {
			return err
		}

		_, err = k.AppendRoutes(routes...)
		if err != nil {
			return err
		}

		m.mu.Lock()
		m.processed[router] = true
		m.mu.Unlock()
	}

	return nil
}

func (m *routerModule) Append(routers ...types.Router) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, r := range routers {
		m.routers = append(m.routers, r)
	}
}
