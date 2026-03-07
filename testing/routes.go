package testing

import (
	"testing"

	"github.com/awesome-goose/goose/core"
	"github.com/awesome-goose/goose/types"
)

// RouteTest provides utilities for testing routes
type RouteTest struct {
	t       *testing.T
	tHelper *T
	routes  types.Routes
	router  types.RouterFinder
}

// NewRouteTest creates a new route test helper
func NewRouteTest(t *testing.T, routes []types.Route) *RouteTest {
	return &RouteTest{
		t:       t,
		tHelper: New(t),
		routes:  routes,
		router:  core.NewRouter(),
	}
}

// T returns the test helper for assertions
func (rt *RouteTest) T() *T {
	return rt.tHelper
}

// Routes returns the routes being tested
func (rt *RouteTest) Routes() types.Routes {
	return rt.routes
}

// Find tests route matching
func (rt *RouteTest) Find(method string, paths []string) (*types.Route, map[string]string, error) {
	rt.t.Helper()
	return rt.router.Find(rt.routes, method, paths)
}

// TestRouteExists verifies a route exists for the given method and path
func (rt *RouteTest) TestRouteExists(method string, paths []string) {
	rt.t.Helper()
	route, _, err := rt.router.Find(rt.routes, method, paths)
	rt.tHelper.Expect(err).ToBeNil()
	rt.tHelper.Expect(route).Not().ToBeNil()
}

// TestRouteNotFound verifies no route exists for the given method and path
func (rt *RouteTest) TestRouteNotFound(method string, paths []string) {
	rt.t.Helper()
	_, _, err := rt.router.Find(rt.routes, method, paths)
	rt.tHelper.Expect(err).Not().ToBeNil()
}

// TestRouteParams verifies route parameters are extracted correctly
func (rt *RouteTest) TestRouteParams(method string, paths []string, expectedParams map[string]string) {
	rt.t.Helper()
	_, params, err := rt.router.Find(rt.routes, method, paths)
	rt.tHelper.Expect(err).ToBeNil()
	rt.tHelper.Expect(params).ToEqual(expectedParams)
}
