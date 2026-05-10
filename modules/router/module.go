package router

import (
	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/types"
)

// dynamicRouters is the module returned by ForRouters. It holds references
// to one or more types.Router implementations that are resolved through the
// DI registry at Boot time. Each ForRouters call returns a fresh instance.
type dynamicRouters struct {
	routers []types.Router
	prefix  string // optional — applied during Boot when non-empty
}

func (d *dynamicRouters) Imports() []types.Module { return nil }
func (d *dynamicRouters) Exports() []any          { return nil }

// Declarations exposes the router instances so the DI container knows about
// them. Preserves the behavior of the previous singleton routerModule.
func (d *dynamicRouters) Declarations() []any {
	out := make([]any, len(d.routers))
	for i, r := range d.routers {
		out[i] = r
	}
	return out
}

// Boot resolves each router through the registry (if needed), collects its
// routes, optionally wraps them under d.prefix, then appends them to the
// kernel.
func (d *dynamicRouters) Boot(k types.Kernel) error {
	for _, r := range d.routers {
		if _, ok := r.(*staticRouter); !ok {
			resolved, err := k.Registry().Get(r)
			if err != nil {
				return err
			}
			rr, ok := resolved.Instance.(types.Router)
			if !ok {
				return errors.ErrInvalidRouterInstance
			}
			r = rr
		}

		routes, err := r.Routes()
		if err != nil {
			return err
		}

		routes = wrapWithPrefix(d.prefix, routes)

		if _, err = k.AppendRoutes(routes...); err != nil {
			return err
		}
	}
	return nil
}
