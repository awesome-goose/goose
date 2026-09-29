package tests

import (
	"fmt"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/awesome-goose/goose"
	"github.com/awesome-goose/goose/io/output"
	"github.com/awesome-goose/goose/platforms/spa"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

type kernelIntrospectionParams struct{}

type kernelIntrospectionModule struct{}

func (m *kernelIntrospectionModule) Imports() []types.Module { return nil }
func (m *kernelIntrospectionModule) Exports() []any          { return nil }
func (m *kernelIntrospectionModule) Declarations() []any     { return nil }

func (m *kernelIntrospectionModule) Boot(k types.Kernel) error {
	_, err := k.AppendRoutes(types.Route{
		Method: types.GET,
		Path:   "introspect-me",
		Handler: func(p *kernelIntrospectionParams) types.Output {
			return output.OK("ok")
		},
	})
	return err
}

func TestKernelIntrospection(t *testing.T) {
	test.NewSuiteRunner(t, &KernelIntrospectionSuite{}).Run()
}

type KernelIntrospectionSuite struct{ test.Suite }

// Before goose.Kernel() existed, Start's only return value was a stop func —
// there was no way for a consumer to get the route table back at all, which
// blocked any tooling built on it: an IDOR test harness that needs to replay
// every registered route (PLAN M1-09) can't discover "every route" without
// this, and neither can the OpenAPI emitter U-G3 plans to build from the
// same table.
func (s *KernelIntrospectionSuite) TestKernel_RoutesReflectsWhatJustBooted() {
	port := 18299
	platform := spa.NewPlatform(
		spa.WithStaticDir(s.T.T().TempDir()),
		spa.WithHost("127.0.0.1"),
		spa.WithPort(port),
		spa.WithAPIPrefix("/api"),
	)

	stopCh := make(chan func() error, 1)
	errCh := make(chan error, 1)
	go func() {
		stop, err := goose.Start(goose.SPA(platform, &kernelIntrospectionModule{}, nil))
		stopCh <- stop
		errCh <- err
	}()
	defer func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		select {
		case <-errCh:
		case <-time.After(5 * time.Second):
			s.T.T().Log("server did not shut down within 5s of SIGTERM")
		}
		// Start's own shutdown/reset (routes, the traverser's boot-hook
		// graph) only runs when the returned stop func is actually called —
		// SIGTERM above just unblocks the SPA platform's own Run() loop.
		// Without calling stop() here, this shared defaultKernel would
		// carry this test's route and boot hook into every later test in
		// this package that also calls goose.Start, and — since
		// AppendRoutes correctly rejects a route re-registered with a real
		// handler on both sides as a genuine duplicate — the *next* such
		// test would fail to boot at all.
		select {
		case stop := <-stopCh:
			_ = stop()
		default:
		}
	}()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-errCh:
			s.T.T().Fatalf("server exited before boot completed: %v", err)
		default:
		}
		if resp, err := http.Get(base + "/api/introspect-me"); err == nil {
			_ = resp.Body.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Other tests in this package also call goose.Start against the same
	// package-level kernel, and routes accumulate across Start calls within
	// one process (Boot appends, shutdown never resets k.routes) — so this
	// asserts our route is present, not that it's the only one.
	found := false
	var walk func(routes types.Routes, prefix string)
	walk = func(routes types.Routes, prefix string) {
		for _, r := range routes {
			full := prefix + "/" + r.Path
			if r.Handler != nil && r.IsMethod("GET") && full == "/introspect-me" {
				found = true
			}
			if r.HasChildren() {
				walk(r.Children, full)
			}
		}
	}
	walk(goose.Kernel().Routes(), "")
	s.T.Expect(found).ToEqual(true)
}
