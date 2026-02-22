package core

import (
	"github.com/awesome-goose/goose/types"
)

type traverser struct {
	container       *Container
	registry        *Registry
	onBootStack     types.Stack[func(types.Kernel) error]
	onShutdownStack types.Stack[func(types.Kernel) error]
}

func NewTraverser() *traverser {
	container := NewContainer()
	return &traverser{
		container:       container,
		registry:        NewRegistry(container),
		onBootStack:     NewStack[func(types.Kernel) error](),
		onShutdownStack: NewStack[func(types.Kernel) error](),
	}
}

func (t *traverser) Traverse(root types.Module) error {
	// Hydrate the registry with the module tree
	if err := t.registry.Hydrate(root); err != nil {
		return err
	}

	// Collect hooks from all registered modules and declarations
	t.collectHooks()

	return nil
}

func (t *traverser) Container() types.Container {
	return t.container
}

func (t *traverser) Registry() types.Registry {
	return t.registry
}

func (t *traverser) OnBootHooks() types.Stack[func(types.Kernel) error] {
	return t.onBootStack
}

func (t *traverser) OnShutdownHooks() types.Stack[func(types.Kernel) error] {
	return t.onShutdownStack
}

// collectHooks iterates through all modules and declarations to collect boot/shutdown hooks.
func (t *traverser) collectHooks() {
	seen := make(map[any]bool)

	for _, module := range t.registry.ListModules() {
		// Collect hooks from the module itself
		if !seen[module] {
			seen[module] = true
			if bootable, ok := module.(types.Bootable); ok {
				t.onBootStack.Push(bootable.Boot)
			}
			if shutdownable, ok := module.(types.Shutdownable); ok {
				t.onShutdownStack.Push(shutdownable.Shutdown)
			}
		}

		// Collect hooks from all declarations in this module's scope
		declarations, err := t.registry.ListDeclarations(module)
		if err != nil {
			continue
		}

		for _, info := range declarations {
			if info.Instance == nil || seen[info.Instance] {
				continue
			}
			seen[info.Instance] = true

			if bootable, ok := info.Instance.(types.Bootable); ok {
				t.onBootStack.Push(bootable.Boot)
			}
			if shutdownable, ok := info.Instance.(types.Shutdownable); ok {
				t.onShutdownStack.Push(shutdownable.Shutdown)
			}
		}
	}
}
