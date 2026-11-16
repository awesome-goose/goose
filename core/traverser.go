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

// ownedInstances returns the declaration instances a module itself owns (not
// ones it merely imports), in creation order.
func (t *traverser) ownedInstances(module types.Module) []any {
	declarations, err := t.registry.ListDeclarations(module)
	if err != nil {
		return nil
	}
	var out []any
	for _, info := range declarations {
		if info.Instance == nil || info.Module != module {
			continue
		}
		out = append(out, info.Instance)
	}
	return out
}

// collectHooks gathers boot/shutdown hooks in a deterministic,
// dependency-respecting order.
//
// Hooks execute LIFO (see stack.ExecuteAll), so the push order is the reverse
// of the desired execution order:
//
//   - Boot: dependencies must boot before the modules that import them, and
//     within one module its declarations boot before the module itself. So
//     modules are visited dependents-first (reverse topological order), and
//     inside a module the module hook is pushed first, then its declarations
//     last-to-first.
//   - Shutdown is the mirror image: dependents stop before their
//     dependencies, and a module stops before its declarations. So modules
//     are visited dependencies-first (topological order), declarations are
//     pushed first-to-last, then the module hook.
//
// Each instance's hooks are registered once, by the module that owns it.
func (t *traverser) collectHooks() {
	modules := t.registry.ListModules() // topological: dependencies first

	seen := make(map[any]bool)
	for i := len(modules) - 1; i >= 0; i-- {
		module := modules[i]
		if !seen[module] {
			seen[module] = true
			if bootable, ok := module.(types.Bootable); ok {
				t.onBootStack.Push(bootable.Boot)
			}
		}
		owned := t.ownedInstances(module)
		for j := len(owned) - 1; j >= 0; j-- {
			if seen[owned[j]] {
				continue
			}
			seen[owned[j]] = true
			if bootable, ok := owned[j].(types.Bootable); ok {
				t.onBootStack.Push(bootable.Boot)
			}
		}
	}

	seenShutdown := make(map[any]bool)
	for _, module := range modules {
		owned := t.ownedInstances(module)
		for _, inst := range owned {
			if seenShutdown[inst] {
				continue
			}
			seenShutdown[inst] = true
			if shutdownable, ok := inst.(types.Shutdownable); ok {
				t.onShutdownStack.Push(shutdownable.Shutdown)
			}
		}
		if !seenShutdown[module] {
			seenShutdown[module] = true
			if shutdownable, ok := module.(types.Shutdownable); ok {
				t.onShutdownStack.Push(shutdownable.Shutdown)
			}
		}
	}
}
