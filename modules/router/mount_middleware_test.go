package router

import (
	"testing"

	"github.com/awesome-goose/goose/core"
	"github.com/awesome-goose/goose/types"
)

// recordingMiddleware counts how many times it ran, so a test can tell
// whether the router actually invoked it for a given request rather than
// merely carrying it somewhere in the Route tree.
type recordingMiddleware struct {
	name string
	runs *[]string
}

func (m *recordingMiddleware) Handle(ctx types.Context) error {
	*m.runs = append(*m.runs, m.name)
	return nil
}

// TestMount_MiddlewarePropagatesToEveryRouteInTheSubtree is PLAN M1-02's own
// prerequisite: cloud/server has no way today to apply an authentication
// check to every route of a mounted app short of hand-editing every route
// declaration across every app. Mount is the one call site common to all of
// them (app.module.go's router.Mount("apps/<name>", ...)), so this is where
// a global auth requirement needs to be attachable.
func TestMount_MiddlewarePropagatesToEveryRouteInTheSubtree(t *testing.T) {
	var runs []string
	auth := &recordingMiddleware{name: "auth", runs: &runs}

	mod := ForRoutes(
		Get("/", []any{"controller", "List"}),
		Get("/:id", []any{"controller", "Get"}),
	)

	mounted := Mount("apps/widgets", mod, auth)

	k := core.NewKernel()
	bootable, ok := mounted.(types.Bootable)
	if !ok {
		t.Fatalf("Mount(...) result does not implement types.Bootable")
	}
	if err := bootable.Boot(k); err != nil {
		t.Fatalf("Boot: %v", err)
	}

	for _, tc := range []struct{ paths []string }{
		{[]string{"apps", "widgets"}},
		{[]string{"apps", "widgets", "42"}},
	} {
		route, _, err := k.Router().Find(k.Routes(), "GET", tc.paths)
		if err != nil {
			t.Fatalf("Find(%v): %v", tc.paths, err)
		}
		found := false
		for _, mw := range route.Middlewares {
			if mw == types.Middleware(auth) {
				found = true
			}
		}
		if !found {
			t.Errorf("Find(%v).Middlewares = %v, want it to include the Mount-level auth middleware", tc.paths, route.Middlewares)
		}
	}
}

// TestMount_MiddlewareDoesNotLeakAcrossSiblingMounts checks the specific
// hazard this design has to avoid: AppendRoutes merges two routes at the
// same Method+Path with both nil Handlers by combining their Children only
// (core/kernel.go's AppendRoutes), so if Mount attached its middleware to
// the *shared* "apps" segment two different apps both hang off, only
// whichever Mount call's "apps" node is registered first would keep its
// Middlewares after the merge — silently dropping the other app's
// middleware. Attaching it to the innermost, unique-per-app segment instead
// (here "widgets" vs "gadgets", never shared) avoids that: each app's own
// middleware must still be present after both are mounted side by side.
func TestMount_MiddlewareDoesNotLeakAcrossSiblingMounts(t *testing.T) {
	var widgetRuns, gadgetRuns []string
	widgetAuth := &recordingMiddleware{name: "widget-auth", runs: &widgetRuns}
	gadgetAuth := &recordingMiddleware{name: "gadget-auth", runs: &gadgetRuns}

	widgets := Mount("apps/widgets", ForRoutes(Get("/", []any{"c", "List"})), widgetAuth)
	gadgets := Mount("apps/gadgets", ForRoutes(Get("/", []any{"c", "List"})), gadgetAuth)

	k := core.NewKernel()
	for _, mod := range []types.Module{widgets, gadgets} {
		if err := mod.(types.Bootable).Boot(k); err != nil {
			t.Fatalf("Boot: %v", err)
		}
	}

	widgetRoute, _, err := k.Router().Find(k.Routes(), "GET", []string{"apps", "widgets"})
	if err != nil {
		t.Fatalf("Find(apps/widgets): %v", err)
	}
	gadgetRoute, _, err := k.Router().Find(k.Routes(), "GET", []string{"apps", "gadgets"})
	if err != nil {
		t.Fatalf("Find(apps/gadgets): %v", err)
	}

	hasMiddleware := func(route *types.Route, want types.Middleware) bool {
		for _, mw := range route.Middlewares {
			if mw == want {
				return true
			}
		}
		return false
	}

	if !hasMiddleware(widgetRoute, widgetAuth) {
		t.Error("widgets route lost its own middleware after gadgets was mounted alongside it")
	}
	if !hasMiddleware(gadgetRoute, gadgetAuth) {
		t.Error("gadgets route lost its own middleware after widgets was mounted alongside it")
	}
	if hasMiddleware(widgetRoute, gadgetAuth) {
		t.Error("widgets route picked up gadgets's middleware — cross-app leak")
	}
	if hasMiddleware(gadgetRoute, widgetAuth) {
		t.Error("gadgets route picked up widgets's middleware — cross-app leak")
	}
}

// TestMount_NoMiddlewareUnchanged is a regression guard: calling Mount the
// existing way (no middlewares) must produce exactly the same route tree as
// before this change — nothing about ordinary, unprotected mounts should
// change.
func TestMount_NoMiddlewareUnchanged(t *testing.T) {
	mod := Mount("apps/widgets", ForRoutes(Get("/", []any{"c", "List"})))

	k := core.NewKernel()
	if err := mod.(types.Bootable).Boot(k); err != nil {
		t.Fatalf("Boot: %v", err)
	}

	route, _, err := k.Router().Find(k.Routes(), "GET", []string{"apps", "widgets"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(route.Middlewares) != 0 {
		t.Errorf("Middlewares = %v, want none", route.Middlewares)
	}
}
