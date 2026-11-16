package core_test

import (
	"fmt"
	"testing"

	"github.com/awesome-goose/goose/core"
	"github.com/awesome-goose/goose/types"
)

// BUGS.md #6: Boot() hooks used to be collected by iterating Go maps, so a
// module could Boot before the module it imports (e.g. a queue module using
// tables another module's Boot migrates). These tests pin the contract:
// dependencies boot before dependents, a module's own declarations boot
// before the module, shutdown is the exact mirror, and the order is the same
// on every run.

type orderLog struct{ events []string }

type orderDecl struct{ name string }

type orderDeclA struct{ orderDecl }
type orderDeclB struct{ orderDecl }

var recorder *orderLog

func (d *orderDeclA) Boot(types.Kernel) error {
	recorder.events = append(recorder.events, "boot:declA")
	return nil
}
func (d *orderDeclA) Shutdown(types.Kernel) error {
	recorder.events = append(recorder.events, "stop:declA")
	return nil
}
func (d *orderDeclB) Boot(types.Kernel) error {
	recorder.events = append(recorder.events, "boot:declB")
	return nil
}
func (d *orderDeclB) Shutdown(types.Kernel) error {
	recorder.events = append(recorder.events, "stop:declB")
	return nil
}

type orderModule struct {
	name    string
	imports []types.Module
	decls   []any
}

func (m *orderModule) Imports() []types.Module { return m.imports }
func (m *orderModule) Exports() []any          { return nil }
func (m *orderModule) Declarations() []any     { return m.decls }
func (m *orderModule) Boot(types.Kernel) error {
	recorder.events = append(recorder.events, "boot:"+m.name)
	return nil
}
func (m *orderModule) Shutdown(types.Kernel) error {
	recorder.events = append(recorder.events, "stop:"+m.name)
	return nil
}

func runOrder(t *testing.T, root types.Module) (boot, stop []string) {
	t.Helper()
	recorder = &orderLog{}
	tr := core.NewTraverser()
	if err := tr.Traverse(root); err != nil {
		t.Fatalf("Traverse: %v", err)
	}
	if err := tr.OnBootHooks().ExecuteAll(func(fn func(types.Kernel) error) error { return fn(nil) }); err != nil {
		t.Fatalf("boot: %v", err)
	}
	boot = recorder.events
	recorder = &orderLog{}
	if err := tr.OnShutdownHooks().ExecuteAll(func(fn func(types.Kernel) error) error { return fn(nil) }); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	return boot, recorder.events
}

func index(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

// A <- B <- C <- root, plus a diamond: root also imports A directly through D.
func buildTree() types.Module {
	a := &orderModule{name: "A", decls: []any{&orderDeclA{}}}
	b := &orderModule{name: "B", imports: []types.Module{a}}
	d := &orderModule{name: "D", imports: []types.Module{a}, decls: []any{&orderDeclB{}}}
	c := &orderModule{name: "C", imports: []types.Module{b, d}}
	return &orderModule{name: "root", imports: []types.Module{c}}
}

func TestBootOrderRespectsDependencies(t *testing.T) {
	boot, _ := runOrder(t, buildTree())
	for _, pair := range [][2]string{{"A", "B"}, {"A", "D"}, {"B", "C"}, {"D", "C"}, {"C", "root"}} {
		if index(boot, "boot:"+pair[0]) >= index(boot, "boot:"+pair[1]) || index(boot, "boot:"+pair[0]) < 0 {
			t.Fatalf("%s must boot before %s; order: %v", pair[0], pair[1], boot)
		}
	}
	if index(boot, "boot:declA") >= index(boot, "boot:A") {
		t.Fatalf("a module's declarations must boot before the module itself; order: %v", boot)
	}
}

func TestShutdownIsTheMirrorOfBoot(t *testing.T) {
	boot, stop := runOrder(t, buildTree())
	if len(boot) != len(stop) {
		t.Fatalf("boot ran %d hooks, shutdown %d", len(boot), len(stop))
	}
	for i, b := range boot {
		want := "stop:" + b[len("boot:"):]
		if stop[len(stop)-1-i] != want {
			t.Fatalf("shutdown must mirror boot.\nboot: %v\nstop: %v", boot, stop)
		}
	}
}

// Go randomizes map iteration per run, so a single pass proves little. Run the
// whole traversal many times and require one identical order every time.
func TestHookOrderIsDeterministic(t *testing.T) {
	want, _ := runOrder(t, buildTree())
	for i := 0; i < 300; i++ {
		got, _ := runOrder(t, buildTree())
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("run %d produced a different boot order:\nfirst: %v\n   got: %v", i, want, got)
		}
	}
}
