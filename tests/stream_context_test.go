package tests

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/awesome-goose/goose"
	"github.com/awesome-goose/goose/io/output"
	"github.com/awesome-goose/goose/platforms/spa"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

// holdEnded receives the reason a context-aware stream handler was released.
var holdEnded = make(chan error, 4)

type holdModule struct{}

func (m *holdModule) Imports() []types.Module { return nil }
func (m *holdModule) Exports() []any          { return nil }
func (m *holdModule) Declarations() []any     { return nil }

func (m *holdModule) Boot(k types.Kernel) error {
	_, err := k.AppendRoutes(types.Route{
		Method: types.GET,
		Path:   "hold",
		Handler: func(p *streamTestParams) types.Output {
			return output.StreamContext(func(ctx context.Context, write func([]byte) error) error {
				if err := write([]byte(": open\n\n")); err != nil {
					return err
				}
				// Holds the connection open while writing NOTHING — the shape
				// of a live event feed between events.
				<-ctx.Done()
				holdEnded <- ctx.Err()
				return nil
			}, output.WithStreamContentType("text/event-stream"))
		},
	})
	return err
}

func TestStreamContext(t *testing.T) {
	test.NewSuiteRunner(t, &StreamContextSuite{}).Run()
}

type StreamContextSuite struct{ test.Suite }

var holdMu sync.Mutex // the tests share one port + the holdEnded channel

func startHold(s *StreamContextSuite, port int) (base string, shutdown func() (time.Duration, bool)) {
	platform := spa.NewPlatform(
		spa.WithStaticDir(s.T.T().TempDir()),
		spa.WithHost("127.0.0.1"),
		spa.WithPort(port),
		spa.WithAPIPrefix("/api"),
	)
	stopCh := make(chan func() error, 1)
	errCh := make(chan error, 1)
	go func() {
		stop, err := goose.Start(goose.SPA(platform, &holdModule{}, nil))
		stopCh <- stop
		errCh <- err
	}()
	base = fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(base + "/api/nonexistent"); err == nil {
			_ = resp.Body.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	return base, func() (time.Duration, bool) {
		start := time.Now()
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		ok := true
		select {
		case <-errCh:
		case <-time.After(5 * time.Second):
			ok = false
		}
		select {
		case stop := <-stopCh:
			_ = stop()
		default:
		}
		return time.Since(start), ok
	}
}

// A client that goes away must release the handler even though the handler
// never writes again — otherwise an idle SSE feed leaks a goroutine (and a
// database poll loop) per dropped connection until its next heartbeat.
func (s *StreamContextSuite) TestStreamContext_CancelledWhenClientDisconnects() {
	holdMu.Lock()
	defer holdMu.Unlock()
	for len(holdEnded) > 0 {
		<-holdEnded
	}
	base, shutdown := startHold(s, 18201)
	defer shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/hold", nil)
	resp, err := http.DefaultClient.Do(req)
	s.T.Expect(err).ToBeNil()
	cancel() // client hangs up
	_ = resp.Body.Close()

	select {
	case got := <-holdEnded:
		s.T.Expect(got).ToEqual(context.Canceled)
	case <-time.After(2 * time.Second):
		s.T.T().Fatal("handler was not released when the client disconnected")
	}
}

// Graceful shutdown must not wait (DefaultShutdownTimeout is 30s) for a live
// stream to finish on its own: the kernel cancels every open stream's
// context first, so the HTTP server can drain.
func (s *StreamContextSuite) TestStreamContext_CancelledOnShutdown_SoShutdownIsPrompt() {
	holdMu.Lock()
	defer holdMu.Unlock()
	for len(holdEnded) > 0 {
		<-holdEnded
	}
	base, shutdown := startHold(s, 18202)

	resp, err := http.Get(base + "/api/hold")
	s.T.Expect(err).ToBeNil()
	defer func() { _ = resp.Body.Close() }()

	took, ok := shutdown()
	s.T.Expect(ok).ToEqual(true)
	if took > 3*time.Second {
		s.T.T().Fatalf("shutdown took %v with one idle stream open", took)
	}
	select {
	case <-holdEnded:
	case <-time.After(time.Second):
		s.T.T().Fatal("stream handler was never released")
	}
}

var slowEnded = make(chan error, 2)

type slowModule struct{}

func (m *slowModule) Imports() []types.Module { return nil }
func (m *slowModule) Exports() []any          { return nil }
func (m *slowModule) Declarations() []any     { return nil }

func (m *slowModule) Boot(k types.Kernel) error {
	_, err := k.AppendRoutes(types.Route{
		Method: types.GET,
		Path:   "flood",
		Handler: func(p *streamTestParams) types.Output {
			return output.StreamContext(func(ctx context.Context, write func([]byte) error) error {
				chunk := make([]byte, 1<<20)
				for ctx.Err() == nil {
					if err := write(chunk); err != nil {
						slowEnded <- err
						return err
					}
				}
				return nil
			}, output.WithStreamWriteTimeout(300*time.Millisecond))
		},
	})
	return err
}

func TestStreamSlowClient(t *testing.T) {
	test.NewSuiteRunner(t, &StreamSlowSuite{}).Run()
}

type StreamSlowSuite struct{ test.Suite }

// A client that connects and never reads must not pin its handler forever:
// with a per-write timeout the blocked write fails and the handler returns
// (PLAN M1-14 "slow-client drop"; TRD §6.1 "dropped after a bounded buffer
// and reconnect with since"). Without one, a stalled TCP peer blocks the
// write indefinitely and the stream's goroutine, DB poll loop and
// per-workspace connection slot leak.
func (s *StreamSlowSuite) TestStreamWriteTimeout_DropsAClientThatStopsReading() {
	port := 18203
	platform := spa.NewPlatform(
		spa.WithStaticDir(s.T.T().TempDir()), spa.WithHost("127.0.0.1"),
		spa.WithPort(port), spa.WithAPIPrefix("/api"),
	)
	stopCh := make(chan func() error, 1)
	errCh := make(chan error, 1)
	go func() {
		stop, err := goose.Start(goose.SPA(platform, &slowModule{}, nil))
		stopCh <- stop
		errCh <- err
	}()
	defer func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		select {
		case <-errCh:
		case <-time.After(5 * time.Second):
		}
		select {
		case stop := <-stopCh:
			_ = stop()
		default:
		}
	}()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if resp, err := http.Get(base + "/api/nonexistent"); err == nil {
			_ = resp.Body.Close()
			break
		}
	}

	resp, err := http.Get(base + "/api/flood") // headers arrive; body is never read
	s.T.Expect(err).ToBeNil()
	defer func() { _ = resp.Body.Close() }()

	select {
	case got := <-slowEnded:
		s.T.Expect(got != nil).ToEqual(true)
	case <-time.After(8 * time.Second):
		s.T.T().Fatal("a client that stopped reading was never dropped")
	}
}
