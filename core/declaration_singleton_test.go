package core_test

import (
	"testing"

	"github.com/awesome-goose/goose/core"
	"github.com/awesome-goose/goose/types"
)

// bootFlagController is a minimal Bootable declaration: Boot sets a field a
// later caller can observe, exactly the shape gox-apps/libs/identity uses
// (AuthorizeController.Boot wires an OAuth registry field; a sibling
// AuthController.Boot constructs an *otp.Service field) to do one-time
// request-independent setup on a controller.
type bootFlagController struct {
	booted bool
}

func (c *bootFlagController) Boot(_ types.Kernel) error {
	c.booted = true
	return nil
}

type bootFlagModule struct{}

func (m *bootFlagModule) Imports() []types.Module { return nil }
func (m *bootFlagModule) Exports() []any          { return nil }
func (m *bootFlagModule) Declarations() []any     { return []any{&bootFlagController{}} }

// TestValueBasedCreateReturnsTheBootedSingleton reproduces a real production
// bug found while implementing Origine PLAN M0-27a: goose's own router
// registers handlers as router.Post("/x", []any{SomeController{}, "Method"})
// — a bare struct VALUE, not a pointer — and kernel.processHandler resolves
// it on every request via container.Create(value). That must return the
// exact singleton the registry constructed (and ran Boot() on) while
// traversing the module tree; otherwise every Boot() hook that wires
// request-time state onto a controller field is silently invisible to real
// request handling, even though Boot() itself runs (and appears to work)
// once, harmlessly, on a throwaway instance nothing ever serves a request
// with.
//
// Before the fix: container.Create(bootFlagController{}) (struct value) took
// the "no existing binding" branch, allocated a brand-new zero-value
// instance via reflect.New, injected it fresh, and returned THAT — never the
// registry's booted singleton — so `got.booted` was false.
func TestValueBasedCreateReturnsTheBootedSingleton(t *testing.T) {
	tr := core.NewTraverser()

	if err := tr.Traverse(&bootFlagModule{}); err != nil {
		t.Fatalf("Traverse: %v", err)
	}

	// Mirrors kernel.runSingle: Boot hooks run once, after traversal, before
	// the platform starts serving requests.
	if err := tr.OnBootHooks().ExecuteAll(func(fn func(types.Kernel) error) error {
		return fn(nil)
	}); err != nil {
		t.Fatalf("OnBootHooks: %v", err)
	}

	// Mirrors kernel.processHandler's per-request resolution of a
	// []any{SomeController{}, "Method"} route handler.
	resolved, err := tr.Container().Create(bootFlagController{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, ok := resolved.(*bootFlagController)
	if !ok {
		t.Fatalf("Create returned %T, want *bootFlagController", resolved)
	}
	if !got.booted {
		t.Fatal("value-based Create() returned an instance Boot() never ran on — " +
			"got a fresh, unbooted copy instead of the registry's singleton")
	}
}
