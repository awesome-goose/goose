package tests

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/awesome-goose/goose"
	"github.com/awesome-goose/goose/io/output"
	"github.com/awesome-goose/goose/platforms/spa"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

type streamTestParams struct{}

type streamOutputModule struct{}

func (m *streamOutputModule) Imports() []types.Module { return nil }
func (m *streamOutputModule) Exports() []any          { return nil }
func (m *streamOutputModule) Declarations() []any     { return nil }

func (m *streamOutputModule) Boot(k types.Kernel) error {
	_, err := k.AppendRoutes(types.Route{
		Method: types.GET,
		Path:   "stream",
		Handler: func(p *streamTestParams) types.Output {
			ch := make(chan output.SSEEvent)
			go func() {
				defer close(ch)
				for i := 0; i < 20; i++ {
					ch <- output.SSEEvent{ID: fmt.Sprintf("%d", i), Data: []byte(fmt.Sprintf("event-%d", i))}
					time.Sleep(20 * time.Millisecond)
				}
			}()
			return output.SSE(ch)
		},
	})
	return err
}

func TestStreamOutput(t *testing.T) {
	test.NewSuiteRunner(t, &StreamOutputSuite{}).Run()
}

type StreamOutputSuite struct{ test.Suite }

// TestSSE_DeliversAllEventsInOrder_PastAFixedWriteTimeout is U-G2's own Done
// criterion ("A handler streaming 100 events over 60s delivers all, in
// order"), scaled down the same way M0-11's write-deadline test was (600ms
// of ticks against a 300ms timeout, not 3 minutes against 30s — the
// mechanism being proven is identical): 20 events 20ms apart (400ms total)
// against a 100ms WithWriteTimeout.
//
// Before this fix (F3 in the TRD): types.StreamOutput was declared but the
// kernel's createHandler had no branch for it at all — every handler's
// output went through one buffer-then-write-once path — so there was no
// supported way to stream in the first place, and nothing cleared the write
// deadline for one, so a real stream longer than the configured timeout
// would have been cut short exactly like F2 documented for the pre-fix
// custom platform.
func (s *StreamOutputSuite) TestSSE_DeliversAllEventsInOrder_PastAFixedWriteTimeout() {
	port := 18199
	platform := spa.NewPlatform(
		spa.WithStaticDir(s.T.T().TempDir()),
		spa.WithHost("127.0.0.1"),
		spa.WithPort(port),
		spa.WithAPIPrefix("/api"),
		spa.WithWriteTimeout(100*time.Millisecond),
	)

	// goose.Start blocks on the platform's ListenAndServe for as long as the
	// server runs, so it must be started in a goroutine — mirrors
	// cloud/server/unmount_test.go's established pattern for the same
	// reason. Stopping it means sending the process the SIGTERM its own
	// signal handler (registered inside kernel.runSingle) already listens
	// for; the returned stop func only becomes available once Start itself
	// returns (i.e. after that SIGTERM already unblocked it), so it's
	// captured here and called in the cleanup below purely to run the
	// kernel's own shutdown/reset — see kernel_introspection_test.go's
	// comment for why that matters once more than one test in this package
	// calls goose.Start.
	stopCh := make(chan func() error, 1)
	errCh := make(chan error, 1)
	go func() {
		stop, err := goose.Start(goose.SPA(platform, &streamOutputModule{}, nil))
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
		select {
		case stop := <-stopCh:
			_ = stop()
		default:
		}
	}()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	var resp *http.Response
	var err error
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-errCh:
			s.T.T().Fatalf("server exited before boot completed: %v", err)
		default:
		}
		resp, err = http.Get(base + "/api/stream")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	s.T.Expect(err).ToBeNil()
	defer func() { _ = resp.Body.Close() }()

	s.T.Expect(resp.Header.Get("Content-Type")).ToContainString("text/event-stream")

	var got []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if line := scanner.Text(); strings.HasPrefix(line, "data: ") {
			got = append(got, strings.TrimPrefix(line, "data: "))
		}
	}
	s.T.Expect(scanner.Err()).ToBeNil()

	want := make([]string, 20)
	for i := range want {
		want[i] = fmt.Sprintf("event-%d", i)
	}
	s.T.Expect(got).ToEqual(want)
}
