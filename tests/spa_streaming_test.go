package tests

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/awesome-goose/goose/platforms/spa"
	test "github.com/awesome-goose/goose/testing"
)

func TestSpaStreaming(t *testing.T) {
	test.NewSuiteRunner(t, &SpaStreamingSuite{}).Run()
}

type SpaStreamingSuite struct {
	test.Suite
}

const (
	streamTicks    = 10
	streamInterval = 60 * time.Millisecond // 10 ticks = ~600 ms of streaming
	streamTimeout  = 300 * time.Millisecond
)

// tickHandler streams streamTicks lines, one per streamInterval. With clear set it
// first removes the server's write deadline, as a long-lived stream must.
func tickHandler(clear bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if clear {
			_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for i := 1; i <= streamTicks; i++ {
			if _, err := fmt.Fprintf(w, "tick %d\n", i); err != nil {
				return
			}
			_ = http.NewResponseController(w).Flush()
			time.Sleep(streamInterval)
		}
	})
}

// receivedTicks counts the complete lines the client receives before the stream ends.
func receivedTicks(s *SpaStreamingSuite, url string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	res, err := rawClient().Do(req)
	must(s.T.T(), err)
	defer res.Body.Close()

	n := 0
	sc := bufio.NewScanner(res.Body)
	for sc.Scan() {
		n++
	}
	return n
}

// TRD F2: the write timeout cut every streamed response. Clearing the deadline
// per request is the supported way to keep a stream alive, and it must work
// through every wrapper in the chain (that needs Unwrap on each of them).
func (s *SpaStreamingSuite) TestAStreamOutlivesTheWriteTimeoutOnlyWhenItClearsItsDeadline() {
	app, _, _ := bootSpa(s.T.T(), everything(
		spa.WithWriteTimeout(streamTimeout),
		spa.WithHandler("/cut", tickHandler(false)),
		spa.WithHandler("/kept", tickHandler(true)),
	)...)
	base := listen(s.T.T(), app)

	cut := receivedTicks(s, base+"/cut")
	kept := receivedTicks(s, base+"/kept")

	s.T.Expect(cut < streamTicks).ToEqual(true) // the timeout still protects handlers that do not opt out
	s.T.Expect(kept).ToEqual(streamTicks)
}

func (s *SpaStreamingSuite) TestWriteTimeoutDefaultsAndOverrides() {
	timeouts := func(options ...spa.Option) (read, write time.Duration) {
		app, _, _ := bootSpa(s.T.T(), options...)
		srv := app.HTTPServer()
		return srv.ReadTimeout, srv.WriteTimeout
	}

	read, write := timeouts()
	s.T.Expect(read).ToEqual(spa.DefaultReadTimeout)
	s.T.Expect(write).ToEqual(spa.DefaultWriteTimeout) // unchanged default

	read, write = timeouts(spa.WithTimeout(5))
	s.T.Expect(read).ToEqual(5 * time.Second)
	s.T.Expect(write).ToEqual(5 * time.Second)

	// WithWriteTimeout beats WithTimeout's write half, whatever the option order.
	read, write = timeouts(spa.WithWriteTimeout(0), spa.WithTimeout(5))
	s.T.Expect(read).ToEqual(5 * time.Second)
	s.T.Expect(write).ToEqual(time.Duration(0))

	read, write = timeouts(spa.WithTimeout(5), spa.WithWriteTimeout(2*time.Minute))
	s.T.Expect(read).ToEqual(5 * time.Second)
	s.T.Expect(write).ToEqual(2 * time.Minute)
}

func (s *SpaStreamingSuite) TestHTTPServerIsTheSameInstanceEachTime() {
	app, _, _ := bootSpa(s.T.T())

	s.T.Expect(app.HTTPServer() == app.HTTPServer()).ToEqual(true)
}
