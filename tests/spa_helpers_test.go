package tests

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/awesome-goose/goose/platforms/spa"
	"github.com/awesome-goose/goose/types"
)

// spaJS is a compressible static asset served in the SPA tests.
var spaJS = strings.Repeat("console.log('goose spa platform');\n", 200)

// spaHeaders is a request's header set.
type spaHeaders map[string]string

// bootSpa boots an SPA app over a temp static dir holding index.html and
// assets/app.js. Its kernel handler answers every API request with
// {"path":"<paths>"} as application/json and counts the calls in *calls.
func bootSpa(t *testing.T, options ...spa.Option) (app *spa.App, dir string, calls *int) {
	t.Helper()
	dir = t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("SPA INDEX"), 0o644))
	must(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte(spaJS), 0o644))

	platform := spa.NewPlatform(append([]spa.Option{spa.WithStaticDir(dir)}, options...)...)
	booted, err := platform.Boot(nil)
	must(t, err)

	app = booted.(*spa.App)
	calls = new(int)
	app.SetHandler(func(c types.Context) error {
		*calls++
		body := `{"path":"` + strings.Join(c.Request().Paths(), "/") + `"}`
		return c.Response().Write(types.SerialTypeObject, []byte(body), 200)
	})
	return app, dir, calls
}

// bootSpaErr boots an SPA platform and returns the boot error.
func bootSpaErr(options ...spa.Option) error {
	_, err := spa.NewPlatform(options...).Boot(nil)
	return err
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// serveOnce runs the request through app.ServeHTTP and returns the recording.
func serveOnce(app http.Handler, method, target string, headers spaHeaders, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

// listen serves the app's real http.Server (with its timeouts) on a free
// loopback port and returns its base URL.
func listen(t *testing.T, app *spa.App) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	must(t, err)
	srv := app.HTTPServer()
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return "http://" + ln.Addr().String()
}

// rawClient never adds or removes Accept-Encoding or decompresses, so a test
// sees exactly what the server sent.
func rawClient() *http.Client {
	return &http.Client{Transport: &http.Transport{DisableCompression: true}}
}

func text(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, body)
	})
}

type responseRecord = httptest.ResponseRecorder

func newRequest(method, target string, body io.Reader) *http.Request {
	return httptest.NewRequest(method, target, body)
}

func newRecorder() *httptest.ResponseRecorder { return httptest.NewRecorder() }
