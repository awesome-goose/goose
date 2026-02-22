package core

import (
	"errors"
	"reflect"
	"sync"
	"unsafe"

	"github.com/awesome-goose/goose/types"
)

// ============================================================================
// VALIDATION ERRORS
// ============================================================================

var (
	// ErrCircularImport: Module A imports B which imports A (directly or indirectly)
	ErrCircularImport = errors.New("registry: circular module import detected")

	// ErrInvalidExport: Module exports something not in its Declarations
	ErrInvalidExport = errors.New("registry: cannot export declaration not owned by module")

	// ErrDuplicateDeclaration: Same type declared in multiple modules
	ErrDuplicateDeclaration = errors.New("registry: declaration type already registered")

	// ErrModuleNotFound: Attempting to resolve in an unregistered module
	ErrModuleNotFound = errors.New("registry: module not found in registry")

	// ErrDeclarationNotInScope: Field type not available in module's scope
	ErrDeclarationNotInScope = errors.New("registry: declaration not available in module scope")

	// ErrDeclarationNotFound: Get() couldn't find the type anywhere
	ErrDeclarationNotFound = errors.New("registry: declaration not found in any module")
)

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

// ModuleMetadata holds metadata extracted from a module.
type ModuleMetadata struct {
	Name     string
	IsGlobal bool
}

// resolvedModule holds the processed state of a module after hydration.
type resolvedModule struct {
	module   types.Module
	metadata ModuleMetadata

	// ownDeclarations: declarations defined in this module's Declarations
	ownDeclarations map[reflect.Type]*types.DeclarationInfo

	// importedDeclarations: declarations available via Imports (from other modules' Exports)
	importedDeclarations map[reflect.Type]*types.DeclarationInfo

	// exports: subset of ownDeclarations that are exported
	exports map[reflect.Type]*types.DeclarationInfo

	// availableDeclarations: union of own + imported + global (used for resolution)
	availableDeclarations map[reflect.Type]*types.DeclarationInfo
}

// ============================================================================
// REGISTRY SERVICE
// ============================================================================

// Registry wraps Container with module-aware dependency resolution.
type Registry struct {
	container *Container

	// moduleRegistry maps each module to its resolved metadata
	moduleRegistry map[reflect.Type]*resolvedModule

	// globalDeclarations holds all globally available declarations
	globalDeclarations map[reflect.Type]*types.DeclarationInfo

	// declarationIndex maps declaration type to its owning module (for Get)
	declarationIndex map[reflect.Type]*types.DeclarationInfo

	mu sync.RWMutex
}

// ============================================================================
// CONSTRUCTOR
// ============================================================================

// NewRegistry creates a new Registry instance.
func NewRegistry(container *Container) *Registry {
	return &Registry{
		container:          container,
		moduleRegistry:     make(map[reflect.Type]*resolvedModule),
		globalDeclarations: make(map[reflect.Type]*types.DeclarationInfo),
		declarationIndex:   make(map[reflect.Type]*types.DeclarationInfo),
	}
}

// ============================================================================
// CORE METHODS
// ============================================================================

// Hydrate walks the module tree starting from the root module and:
// 1. Validates module configurations (no invalid exports, no circular imports)
// 2. Builds the module registry with resolved declarations
// 3. Registers all declarations with the underlying Container
// 4. Returns error if any validation fails
//
// Algorithm:
//  1. Topological sort of module graph (detect cycles)
//  2. For each module (in dependency order):
//     a. Validate: exports ⊆ declarations
//     b. Register declarations with Container
//     c. Collect exports from imported modules
//     d. Build availableDeclarations = own ∪ imported ∪ global
//  3. Build global declaration index
func (r *Registry) Hydrate(rootModule types.Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Step 1: Topological sort with cycle detection
	sorted, err := r.topologicalSort(rootModule)
	if err != nil {
		return err
	}

	// Step 2: Process modules in dependency order
	for _, mod := range sorted {
		if err := r.processModule(mod); err != nil {
			return err
		}
	}

	// Step 3: Build available declarations for each module (includes globals)
	for _, rm := range r.moduleRegistry {
		r.buildAvailableDeclarations(rm)
	}

	return nil
}

// topologicalSort performs a topological sort of the module graph with cycle detection.
func (r *Registry) topologicalSort(root types.Module) ([]types.Module, error) {
	var result []types.Module
	visited := make(map[reflect.Type]bool)
	inStack := make(map[reflect.Type]bool)

	var visit func(mod types.Module) error
	visit = func(mod types.Module) error {
		modType := reflect.TypeOf(mod)

		if inStack[modType] {
			return ErrCircularImport
		}

		if visited[modType] {
			return nil
		}

		inStack[modType] = true
		visited[modType] = true

		// Visit all imports first (dependencies)
		for _, imp := range mod.Imports() {
			if err := visit(imp); err != nil {
				return err
			}
		}

		inStack[modType] = false
		result = append(result, mod)
		return nil
	}

	if err := visit(root); err != nil {
		return nil, err
	}

	return result, nil
}

// processModule processes a single module: validates, registers declarations, and builds exports.
func (r *Registry) processModule(mod types.Module) error {
	modType := reflect.TypeOf(mod)

	// Check if module is global (optional, defaults to false)
	isGlobal := false
	if g, ok := mod.(types.Global); ok {
		isGlobal = g.IsGlobal()
	}

	// Create resolved module
	rm := &resolvedModule{
		module: mod,
		metadata: ModuleMetadata{
			Name:     modType.String(),
			IsGlobal: isGlobal,
		},
		ownDeclarations:       make(map[reflect.Type]*types.DeclarationInfo),
		importedDeclarations:  make(map[reflect.Type]*types.DeclarationInfo),
		exports:               make(map[reflect.Type]*types.DeclarationInfo),
		availableDeclarations: make(map[reflect.Type]*types.DeclarationInfo),
	}

	// Register declarations
	for _, decl := range mod.Declarations() {
		declType := r.getDeclarationType(decl)

		// Check for duplicate declarations across modules
		if existing, exists := r.declarationIndex[declType]; exists {
			if reflect.TypeOf(existing.Module) != modType {
				return ErrDuplicateDeclaration
			}
		}

		// Create instance via container
		instance, err := r.container.Create(decl)
		if err != nil {
			return err
		}

		info := &types.DeclarationInfo{
			Instance: instance,
			Type:     declType,
			Module:   mod,
			IsGlobal: isGlobal,
		}

		rm.ownDeclarations[declType] = info
		r.declarationIndex[declType] = info

		// Track global declarations
		if isGlobal {
			r.globalDeclarations[declType] = info
		}
	}

	// Validate exports: all exports must be in declarations
	exportSet := make(map[reflect.Type]bool)
	for _, exp := range mod.Exports() {
		expType := r.getDeclarationType(exp)
		exportSet[expType] = true

		if _, found := rm.ownDeclarations[expType]; !found {
			return ErrInvalidExport
		}

		rm.exports[expType] = rm.ownDeclarations[expType]
	}

	// Collect imported declarations from imported modules' exports
	for _, imp := range mod.Imports() {
		impType := reflect.TypeOf(imp)
		if importedMod, exists := r.moduleRegistry[impType]; exists {
			for expType, expInfo := range importedMod.exports {
				rm.importedDeclarations[expType] = expInfo
			}
			// Also inherit globals from imported modules
			if importedMod.metadata.IsGlobal {
				for declType, declInfo := range importedMod.ownDeclarations {
					rm.importedDeclarations[declType] = declInfo
				}
			}
		}
	}

	r.moduleRegistry[modType] = rm
	return nil
}

// buildAvailableDeclarations builds the union of own + imported + global declarations.
func (r *Registry) buildAvailableDeclarations(rm *resolvedModule) {
	// Add own declarations
	for t, info := range rm.ownDeclarations {
		rm.availableDeclarations[t] = info
	}

	// Add imported declarations
	for t, info := range rm.importedDeclarations {
		rm.availableDeclarations[t] = info
	}

	// Add global declarations
	for t, info := range r.globalDeclarations {
		if _, exists := rm.availableDeclarations[t]; !exists {
			rm.availableDeclarations[t] = info
		}
	}
}

// getDeclarationType returns the reflect.Type for a declaration.
// Handles both pointer and non-pointer types, normalizing to pointer type.
func (r *Registry) getDeclarationType(decl any) reflect.Type {
	t := reflect.TypeOf(decl)
	if t.Kind() == reflect.Ptr {
		return t
	}
	return reflect.PointerTo(t)
}

// Resolve resolves a struct's injectable fields within a module's scope.
// Fields tagged with `inject:""` are resolved using:
//  1. Module's own declarations
//  2. Declarations imported from other modules (via Imports + Exports)
//  3. Global declarations
//
// Returns error if:
//   - Field type is not available in the module's scope
//   - Module is not registered (not part of the hydrated tree)
func (r *Registry) Resolve(target any, module types.Module) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modType := reflect.TypeOf(module)
	rm, exists := r.moduleRegistry[modType]
	if !exists {
		return ErrModuleNotFound
	}

	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("registry: target must be a pointer to struct")
	}

	s := v.Elem()
	sType := s.Type()

	for i := 0; i < s.NumField(); i++ {
		field := s.Field(i)
		fieldType := field.Type()

		if _, exist := sType.Field(i).Tag.Lookup("inject"); exist {
			// Look up in the module's available declarations
			info, found := rm.availableDeclarations[fieldType]
			if !found {
				// Try without pointer for interface types
				if fieldType.Kind() == reflect.Interface {
					// Search for implementations
					for declType, declInfo := range rm.availableDeclarations {
						if declType.Implements(fieldType) {
							info = declInfo
							found = true
							break
						}
					}
				}
			}

			if !found {
				return ErrDeclarationNotInScope
			}

			// Set the field value
			ptr := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
			ptr.Set(reflect.ValueOf(info.Instance))
		}
	}

	return nil
}

// Get retrieves a declaration by type from anywhere in the registry.
// Traverses all modules to find the declaration and returns its info.
//
// Returns:
//   - DeclarationInfo with instance, type, and owning module
//   - Error if declaration not found in any module
//
// Use this instead of container.Create() when you need module context.
func (r *Registry) Get(declarationType any) (*types.DeclarationInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	declType := r.getDeclarationType(declarationType)

	if info, exists := r.declarationIndex[declType]; exists {
		return info, nil
	}

	return nil, ErrDeclarationNotFound
}

// ============================================================================
// ADDITIONAL METHODS
// ============================================================================

// GetModule returns the resolved module info for a given module type.
func (r *Registry) GetModule(module types.Module) (*resolvedModule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modType := reflect.TypeOf(module)
	if rm, exists := r.moduleRegistry[modType]; exists {
		return rm, nil
	}

	return nil, ErrModuleNotFound
}

// IsAvailable checks if a declaration type is available within a module's scope.
func (r *Registry) IsAvailable(declarationType any, module types.Module) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modType := reflect.TypeOf(module)
	rm, exists := r.moduleRegistry[modType]
	if !exists {
		return false
	}

	declType := r.getDeclarationType(declarationType)
	_, found := rm.availableDeclarations[declType]

	if !found && reflect.TypeOf(declarationType).Kind() == reflect.Interface {
		// Check if any declaration implements the interface
		for t := range rm.availableDeclarations {
			if t.Implements(reflect.TypeOf(declarationType).Elem()) {
				return true
			}
		}
	}

	return found
}

// ListModules returns all registered modules.
func (r *Registry) ListModules() []types.Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modules := make([]types.Module, 0, len(r.moduleRegistry))
	for _, rm := range r.moduleRegistry {
		modules = append(modules, rm.module)
	}
	return modules
}

// ListDeclarations returns all declarations in a module's scope.
func (r *Registry) ListDeclarations(module types.Module) ([]*types.DeclarationInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modType := reflect.TypeOf(module)
	rm, exists := r.moduleRegistry[modType]
	if !exists {
		return nil, ErrModuleNotFound
	}

	declarations := make([]*types.DeclarationInfo, 0, len(rm.availableDeclarations))
	for _, info := range rm.availableDeclarations {
		declarations = append(declarations, info)
	}
	return declarations, nil
}

// ListGlobals returns all globally available declarations.
func (r *Registry) ListGlobals() []*types.DeclarationInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	globals := make([]*types.DeclarationInfo, 0, len(r.globalDeclarations))
	for _, info := range r.globalDeclarations {
		globals = append(globals, info)
	}
	return globals
}
