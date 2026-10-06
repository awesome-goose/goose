package tests

import (
	"fmt"
	"io"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/awesome-goose/goose"
	"github.com/awesome-goose/goose/io/output"
	"github.com/awesome-goose/goose/platforms/api"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

type mwParams struct{}

// replayMW answers /mw/replay itself, never reaching the handler.
type replayMW struct{ called *bool }

func (m replayMW) Handle(c types.Context) error {
	if c.Request().Paths()[len(c.Request().Paths())-1] == "mw-replay" {
		return &types.Respond{Output: headerOut{data: "replayed", code: 202, headers: map[string]string{"X-Replayed": "true"}}}
	}
	return nil
}

// afterMW observes the handler's output and adds a header.
type afterMW struct{}

func (afterMW) Handle(types.Context) error { return nil }
func (afterMW) After(_ types.Context, out types.Output) (types.Output, error) {
	return headerOut{data: out.Data(), code: out.Code(), headers: map[string]string{"X-After": "yes"}}, nil
}

type headerOut struct {
	data    any
	code    int
	headers map[string]string
}

func (h headerOut) Data() any                  { return h.data }
func (h headerOut) Code() int                  { return h.code }
func (h headerOut) Headers() map[string]string { return h.headers }
func (h headerOut) ContentType() string        { return "" }

type mwModule struct{ handlerRan *bool }

func (m *mwModule) Imports() []types.Module { return nil }
func (m *mwModule) Exports() []any          { return nil }
func (m *mwModule) Declarations() []any     { return nil }
func (m *mwModule) Boot(k types.Kernel) error {
	handler := func(p *mwParams) types.Output {
		*m.handlerRan = true
		return output.OK("from-handler")
	}
	_, err := k.AppendRoutes(
		types.Route{Method: types.GET, Path: "mw-replay", Handler: handler, Middlewares: types.Middlewares{replayMW{}}},
		types.Route{Method: types.GET, Path: "mw-after", Handler: handler, Middlewares: types.Middlewares{afterMW{}}},
	)
	return err
}

func TestMiddlewareRespondAndAfter(t *testing.T) {
	test.NewSuiteRunner(t, &MiddlewareSuite{}).Run()
}

type MiddlewareSuite struct{ test.Suite }

func (s *MiddlewareSuite) TestRespondShortCircuitsAndAfterObservesOutput() {
	port := 18311
	ran := false
	stopCh := make(chan func() error, 1)
	errCh := make(chan error, 1)
	go func() {
		stop, err := goose.Start(goose.API(api.NewPlatform(api.WithHost("127.0.0.1"), api.WithPort(port)), &mwModule{handlerRan: &ran}, nil))
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
	deadline := time.Now().Add(5 * time.Second)
	var resp *http.Response
	var err error
	for time.Now().Before(deadline) {
		if resp, err = http.Get(base + "/mw-replay"); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	s.T.Expect(err).ToBeNil()
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	s.T.Expect(resp.StatusCode).ToEqual(202)
	s.T.Expect(resp.Header.Get("X-Replayed")).ToEqual("true")
	s.T.Expect(string(body)).ToContain("replayed")
	s.T.Expect(ran).ToEqual(false)

	resp2, err := http.Get(base + "/mw-after")
	s.T.Expect(err).ToBeNil()
	body2, _ := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	s.T.Expect(resp2.StatusCode).ToEqual(200)
	s.T.Expect(resp2.Header.Get("X-After")).ToEqual("yes")
	s.T.Expect(string(body2)).ToContain("from-handler")
	s.T.Expect(ran).ToEqual(true)
}
