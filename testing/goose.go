package testing

import (
	"testing"

	"github.com/awesome-goose/goose/types"
)

// GooseTest provides Goose-specific testing utilities
type GooseTest struct {
	*T
	t         *testing.T
	container *TestContainer
	ctx       *MockContext
}

// NewGooseTest creates a new Goose test helper
func NewGooseTest(t *testing.T) *GooseTest {
	t.Helper()
	return &GooseTest{
		T:         New(t),
		t:         t,
		container: NewTestContainer(),
	}
}

// WithContainer sets up a test container with registrations
func (g *GooseTest) WithContainer(setup func(c *TestContainer)) *GooseTest {
	setup(g.container)
	return g
}

// Container returns the test container
func (g *GooseTest) Container() *TestContainer {
	return g.container
}

// Context returns a new mock context
func (g *GooseTest) Context() *MockContext {
	if g.ctx == nil {
		g.ctx = NewMockContext()
	}
	return g.ctx
}

// NewContext creates a fresh mock context
func (g *GooseTest) NewContext() *MockContext {
	g.ctx = NewMockContext()
	return g.ctx
}

// Service creates a service test helper
func (g *GooseTest) Service(service any) *ServiceTest {
	return NewServiceTest(g.t, service, g.container)
}

// Controller creates a controller test helper
func (g *GooseTest) Controller(controller any) *ControllerTest {
	return NewControllerTest(g.t, controller, g.container)
}

// Route creates a route test helper
func (g *GooseTest) Route(routes []types.Route) *RouteTest {
	return NewRouteTest(g.t, routes)
}

// Module creates a module test helper
func (g *GooseTest) Module(module types.Module) *ModuleTest {
	return NewModuleTest(g.t, module, g.container)
}

// Entity creates an entity test helper
func Entity[T any](g *GooseTest, entity types.Entity[T]) *EntityTest[T] {
	return NewEntityTest(g.t, entity)
}
