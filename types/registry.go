package types

import (
	"reflect"
)

// DeclarationInfo holds resolved information about a declaration.
type DeclarationInfo struct {
	// Instance is the resolved singleton instance.
	Instance any

	// Type is the reflect.Type of the declaration.
	Type reflect.Type

	// Module is the module that owns this declaration.
	Module Module

	// IsGlobal indicates if this declaration is globally available.
	IsGlobal bool
}

// Registry provides module-aware dependency resolution.
type Registry interface {
	// Hydrate walks the module tree starting from the root module and:
	// 1. Validates module configurations (no invalid exports, no circular imports)
	// 2. Builds the module registry with resolved declarations
	// 3. Registers all declarations with the underlying Container
	// 4. Returns error if any validation fails
	Hydrate(rootModule Module) error

	// Resolve resolves a struct's injectable fields within a module's scope.
	// Fields tagged with `inject:""` are resolved using:
	//   1. Module's own declarations
	//   2. Declarations imported from other modules (via Imports + Exports)
	//   3. Global declarations
	Resolve(target any, module Module) error

	// Get retrieves a declaration by type from anywhere in the registry.
	// Returns DeclarationInfo with instance, type, and owning module.
	Get(declarationType any) (*DeclarationInfo, error)

	// IsAvailable checks if a declaration type is available within a module's scope.
	IsAvailable(declarationType any, module Module) bool

	// ListModules returns all registered modules.
	ListModules() []Module

	// ListDeclarations returns all declarations in a module's scope.
	ListDeclarations(module Module) ([]*DeclarationInfo, error)

	// ListGlobals returns all globally available declarations.
	ListGlobals() []*DeclarationInfo
}
