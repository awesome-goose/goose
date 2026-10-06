package router

import (
	"testing"

	"github.com/awesome-goose/goose/types"
)

type silentParent struct{ subs []types.Module }

func (p silentParent) Imports() []types.Module { return p.subs }
func (p silentParent) Exports() []any          { return []any{"exported"} }
func (p silentParent) Declarations() []any     { return []any{"declared"} }

func TestSilentRemovesRoutesAtEveryDepthButKeepsTheDISurface(t *testing.T) {
	leaf := ForRoutes(Get("/leaf", func() {}))
	child := silentParent{subs: []types.Module{leaf, ForRoutes(Get("/other", func() {}))}}
	parent := silentParent{subs: []types.Module{child, ForRoutes(Get("/top", func() {}))}}

	m := Silent(parent)
	if len(m.Exports()) != 1 || len(m.Declarations()) != 1 {
		t.Fatal("the DI surface must pass through")
	}
	var count func(types.Module) int
	count = func(mod types.Module) int {
		n := 0
		if r, ok := mod.(interface{ Routes() (types.Routes, error) }); ok {
			rs, _ := r.Routes()
			n += len(rs)
		}
		for _, s := range mod.Imports() {
			n += count(s)
		}
		return n
	}
	if got := count(m); got != 0 {
		t.Fatalf("Silent left %d route(s) registered", got)
	}
	if count(parent) == 0 {
		t.Fatal("test is vacuous: the unwrapped tree has routes")
	}
}

func TestSilentOnARouterIsEmpty(t *testing.T) {
	r := Silent(ForRoutes(Get("/x", func() {})))
	rs, err := r.(interface{ Routes() (types.Routes, error) }).Routes()
	if err != nil || len(rs) != 0 {
		t.Fatalf("%v %v", rs, err)
	}
}
