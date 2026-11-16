package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/awesome-goose/goose/io/input"
	"github.com/awesome-goose/goose/platforms/api"
	"github.com/awesome-goose/goose/platforms/cli"
	"github.com/awesome-goose/goose/platforms/web"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

type ctxKey string

type ctxDto struct {
	ID  string `param:"id"`
	Ctx context.Context
}

// A handler's DTO gets the request's own context.Context: cancelled when the
// client goes away, with its deadline, carrying the values middleware set.
// Handlers used to have no way to reach it, so cancellation never got to the
// database.
func TestInput_ContextField_IsTheRequestContext(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	r := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(parent)

	for name, c := range map[string]types.Context{
		"api": api.NewContext(httptest.NewRecorder(), r),
		"web": web.NewContext(httptest.NewRecorder(), r),
	} {
		c.SetValue(ctxKey("tenant"), "ws-a") // as a middleware does, before the handler
		var dto ctxDto
		if err := input.NewInput(c).Populate(&dto); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if dto.Ctx == nil {
			t.Fatalf("%s: context field was not bound", name)
		}
		if dto.Ctx.Value(ctxKey("tenant")) != "ws-a" {
			t.Errorf("%s: the bound context does not carry SetValue's values", name)
		}
		if dto.Ctx.Err() != nil {
			t.Errorf("%s: context is done before the request ended", name)
		}
		if got := types.RequestContext(c); got.Value(ctxKey("tenant")) != "ws-a" {
			t.Errorf("%s: RequestContext does not carry SetValue's values", name)
		}
	}

	cancel() // the client disconnects
	for name, c := range map[string]types.Context{
		"api": api.NewContext(httptest.NewRecorder(), r),
		"web": web.NewContext(httptest.NewRecorder(), r),
	} {
		select {
		case <-types.RequestContext(c).Done():
		case <-time.After(time.Second):
			t.Errorf("%s: RequestContext is not cancelled with the request", name)
		}
	}
}

func TestRequestContext_FallsBackToBackground(t *testing.T) {
	if types.RequestContext(nil) == nil {
		t.Fatal("nil Context gave a nil context")
	}
	if ctx := types.RequestContext(cli.NewContext()); ctx == nil || ctx.Err() != nil {
		t.Fatalf("cli context = %v", ctx)
	}
	// a required field check does not count the context as something the caller sends
	var dto struct {
		Ctx  context.Context `binding:"required"`
		Name string          `json:"name" binding:"required"`
	}
	m := test.NewMockContext()
	m.MockRequest().WithJSONBody(map[string]any{"name": "x"})
	if err := input.NewInput(m).Populate(&dto); err != nil {
		t.Fatalf("a context field was reported missing: %v", err)
	}
}

func TestMockContext_WithContext(t *testing.T) {
	deadline := time.Now().Add(time.Hour)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	m := test.NewMockContext().WithContext(ctx)
	var dto ctxDto
	if err := input.NewInput(m).Populate(&dto); err != nil {
		t.Fatal(err)
	}
	if d, ok := dto.Ctx.Deadline(); !ok || !d.Equal(deadline) {
		t.Fatalf("deadline = %v %v, want %v", d, ok, deadline)
	}
	if d, ok := types.RequestContext(test.NewMockContext()).Deadline(); ok {
		t.Fatalf("a fresh mock context has a deadline: %v", d)
	}
}
