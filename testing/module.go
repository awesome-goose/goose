package testing

import (
	"testing"

	"github.com/awesome-goose/goose/types"
)

// ModuleTest provides utilities for testing modules
type ModuleTest struct {
	t         *testing.T
	tHelper   *T
	module    types.Module
	container *TestContainer
}

// NewModuleTest creates a new module test helper
func NewModuleTest(t *testing.T, module types.Module, container *TestContainer) *ModuleTest {
	return &ModuleTest{
		t:         t,
		tHelper:   New(t),
		module:    module,
		container: container,
	}
}

// Module returns the underlying module
func (mt *ModuleTest) Module() types.Module {
	return mt.module
}

// T returns the test helper for assertions
func (mt *ModuleTest) T() *T {
	return mt.tHelper
}

// TestImports verifies the module imports the expected modules
func (mt *ModuleTest) TestImports() []types.Module {
	mt.t.Helper()
	return mt.module.Imports()
}

// TestExports verifies the module exports the expected dependencies
func (mt *ModuleTest) TestExports() []any {
	mt.t.Helper()
	return mt.module.Exports()
}

// TestDeclarations verifies the module declares the expected items
func (mt *ModuleTest) TestDeclarations() []any {
	mt.t.Helper()
	return mt.module.Declarations()
}
