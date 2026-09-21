package tests

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/awesome-goose/goose/platforms/spa"
	test "github.com/awesome-goose/goose/testing"
)

func TestSpaCompression(t *testing.T) {
	test.NewSuiteRunner(t, &SpaCompressionSuite{}).Run()
}

type SpaCompressionSuite struct {
	test.Suite
}

// everything enables every pipeline option at once, so a test through it proves
// that no layer in the chain drops Flush or Hijack.
func everything(extra ...spa.Option) []spa.Option {
	headers := spa.DefaultSecurityHeaders()
	headers.ContentSecurityPolicy = "default-src 'self'"
	return append([]spa.Option{
		spa.WithCompression(),
		spa.WithRecovery(nil),
		spa.WithBodyLimit(1 << 20),
		spa.WithCORS(spa.CORS{AllowedOrigins: []string{"https://app.example"}}),
		spa.WithSecurityHeaders(headers),
		spa.WithMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) })
		}),
	}, extra...)
}

func gunzip(t *testing.T, b []byte) string {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(b))
	must(t, err)
	out, err := io.ReadAll(zr)
	must(t, err)
	return string(out)
}

func gzipAccepted() spaHeaders { return spaHeaders{"Accept-Encoding": "gzip"} }

// --- gzip ------------------------------------------------------------------

func (s *SpaCompressionSuite) TestGzipsCompressibleAPIAndStaticResponses() {
	app, _, _ := bootSpa(s.T.T(), spa.WithCompression())

	api := serveOnce(app, "GET", "/api/ping", gzipAccepted(), nil)
	s.T.Expect(api.Header().Get("Content-Encoding")).ToEqual("gzip")
	s.T.Expect(api.Header().Values("Vary")).ToContain("Accept-Encoding")
	s.T.Expect(api.Header().Get("Content-Length")).ToEqual("")
	s.T.Expect(gunzip(s.T.T(), api.Body.Bytes())).ToEqual(`{"path":"ping"}`)

	js := serveOnce(app, "GET", "/assets/app.js", gzipAccepted(), nil)
	s.T.Expect(js.Header().Get("Content-Encoding")).ToEqual("gzip")
	s.T.Expect(js.Header().Get("Content-Length")).ToEqual("")
	s.T.Expect(gunzip(s.T.T(), js.Body.Bytes())).ToEqual(spaJS)
	s.T.Expect(js.Body.Len() < len(spaJS)/4).ToEqual(true)
}

func (s *SpaCompressionSuite) TestNoCompressionWhenTheClientCannotTakeGzip() {
	app, _, _ := bootSpa(s.T.T(), spa.WithCompression())

	for _, enc := range []string{"", "identity", "gzip;q=0", "br"} {
		res := serveOnce(app, "GET", "/assets/app.js", spaHeaders{"Accept-Encoding": enc}, nil)
		s.T.Expect(res.Header().Get("Content-Encoding")).ToEqual("")
		s.T.Expect(res.Body.String()).ToEqual(spaJS)
		// Caches must still key on Accept-Encoding for a compressible type.
		s.T.Expect(res.Header().Values("Vary")).ToContain("Accept-Encoding")
	}
}

func (s *SpaCompressionSuite) TestLeavesAloneWhatMustNotOrNeedNotBeCompressed() {
	app, _, _ := bootSpa(s.T.T(), spa.WithCompression(),
		spa.WithHandler("/img", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(bytes.Repeat([]byte{0x89}, 2000))
		})),
		spa.WithHandler("/encoded", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "br")
			_, _ = io.WriteString(w, "already-br")
		})),
		spa.WithHandler("/empty", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})))

	img := serveOnce(app, "GET", "/img", gzipAccepted(), nil)
	s.T.Expect(img.Header().Get("Content-Encoding")).ToEqual("")
	s.T.Expect(img.Body.Len()).ToEqual(2000)

	encoded := serveOnce(app, "GET", "/encoded", gzipAccepted(), nil)
	s.T.Expect(encoded.Header().Get("Content-Encoding")).ToEqual("br")
	s.T.Expect(encoded.Body.String()).ToEqual("already-br")

	empty := serveOnce(app, "GET", "/empty", gzipAccepted(), nil)
	s.T.Expect(empty.Code).ToEqual(204)
	s.T.Expect(empty.Header().Get("Content-Encoding")).ToEqual("")

	head := serveOnce(app, "HEAD", "/assets/app.js", gzipAccepted(), nil)
	s.T.Expect(head.Header().Get("Content-Encoding")).ToEqual("")

	rng := serveOnce(app, "GET", "/assets/app.js", spaHeaders{"Accept-Encoding": "gzip", "Range": "bytes=0-9"}, nil)
	s.T.Expect(rng.Code).ToEqual(206)
	s.T.Expect(rng.Header().Get("Content-Encoding")).ToEqual("")
	s.T.Expect(rng.Body.String()).ToEqual(spaJS[:10])
}

func (s *SpaCompressionSuite) TestSniffsTheContentTypeWhenTheHandlerSetsNone() {
	page := "<html><body>" + strings.Repeat("hello ", 300) + "</body></html>"
	app, _, _ := bootSpa(s.T.T(), spa.WithCompression(), spa.WithHandler("/page", text(page)))

	res := serveOnce(app, "GET", "/page", gzipAccepted(), nil)

	s.T.Expect(res.Header().Get("Content-Type")).ToContainString("text/html")
	s.T.Expect(res.Header().Get("Content-Encoding")).ToEqual("gzip")
	s.T.Expect(gunzip(s.T.T(), res.Body.Bytes())).ToEqual(page)
}

// --- precompressed siblings ------------------------------------------------

func (s *SpaCompressionSuite) TestServesPrecompressedSiblings() {
	app, dir, _ := bootSpa(s.T.T(), spa.WithCompression())
	must(s.T.T(), os.WriteFile(filepath.Join(dir, "assets", "app.js.br"), []byte("BR-BYTES"), 0o644))
	must(s.T.T(), os.WriteFile(filepath.Join(dir, "assets", "app.js.gz"), []byte("GZ-BYTES"), 0o644))

	br := serveOnce(app, "GET", "/assets/app.js", spaHeaders{"Accept-Encoding": "gzip, br"}, nil)
	s.T.Expect(br.Body.String()).ToEqual("BR-BYTES")
	s.T.Expect(br.Header().Get("Content-Encoding")).ToEqual("br")
	s.T.Expect(br.Header().Get("Content-Type")).ToContainString("javascript")
	s.T.Expect(br.Header().Values("Vary")).ToContain("Accept-Encoding")

	gz := serveOnce(app, "GET", "/assets/app.js", gzipAccepted(), nil)
	s.T.Expect(gz.Body.String()).ToEqual("GZ-BYTES")
	s.T.Expect(gz.Header().Get("Content-Encoding")).ToEqual("gzip")
	s.T.Expect(gz.Header().Get("Content-Type")).ToContainString("javascript")

	plain := serveOnce(app, "GET", "/assets/app.js", spaHeaders{"Accept-Encoding": "identity"}, nil)
	s.T.Expect(plain.Body.String()).ToEqual(spaJS)
	s.T.Expect(plain.Header().Get("Content-Encoding")).ToEqual("")

	// A range applies to the identity file, never to a sibling.
	rng := serveOnce(app, "GET", "/assets/app.js", spaHeaders{"Accept-Encoding": "br", "Range": "bytes=0-9"}, nil)
	s.T.Expect(rng.Body.String()).ToEqual(spaJS[:10])
}

func (s *SpaCompressionSuite) TestSiblingsAreIgnoredWithoutCompression() {
	app, dir, _ := bootSpa(s.T.T())
	must(s.T.T(), os.WriteFile(filepath.Join(dir, "assets", "app.js.br"), []byte("BR-BYTES"), 0o644))

	res := serveOnce(app, "GET", "/assets/app.js", spaHeaders{"Accept-Encoding": "br"}, nil)

	s.T.Expect(res.Body.String()).ToEqual(spaJS)
	s.T.Expect(res.Header().Get("Content-Encoding")).ToEqual("")
}

// --- Flush and Hijack survive the chain (TRD F1) ---------------------------

func (s *SpaCompressionSuite) TestWrappedWriterKeepsFlusherAndHijacker() {
	var flusher, hijacker, controllerFlush bool
	app, _, _ := bootSpa(s.T.T(), everything(spa.WithHandler("/probe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, flusher = w.(http.Flusher)
		_, hijacker = w.(http.Hijacker)
		controllerFlush = http.NewResponseController(w).Flush() == nil
	})))...)

	serveOnce(app, "GET", "/probe", gzipAccepted(), nil)

	s.T.Expect(flusher).ToEqual(true)
	s.T.Expect(hijacker).ToEqual(true)
	s.T.Expect(controllerFlush).ToEqual(true)
}

// A browser sends Accept-Encoding: gzip on the WebSocket upgrade. The custom
// platform's gzip writer answered it with 500 "bad handshake" (F1).
func (s *SpaCompressionSuite) TestWebSocketUpgradeSurvivesEveryOption() {
	app, _, _ := bootSpa(s.T.T(), everything(spa.WithHandler("/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker) // the assertion gorilla/websocket makes
		if !ok {
			http.Error(w, "not a hijacker", http.StatusInternalServerError)
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.WriteString(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\nhello")
	})))...)
	base := listen(s.T.T(), app)

	conn, err := net.Dial("tcp", strings.TrimPrefix(base, "http://"))
	must(s.T.T(), err)
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = io.WriteString(conn, "GET /ws HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"+
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\nAccept-Encoding: gzip\r\n\r\n")
	must(s.T.T(), err)

	status, err := bufio.NewReader(conn).ReadString('\n')
	must(s.T.T(), err)

	s.T.Expect(strings.TrimSpace(status)).ToEqual("HTTP/1.1 101 Switching Protocols")
}

// The first event must reach the client while the handler is still running; the
// custom platform held it until the handler returned (F1).
func (s *SpaCompressionSuite) TestSSEFirstEventArrivesBeforeTheHandlerReturns() {
	release := make(chan struct{})
	app, _, _ := bootSpa(s.T.T(), everything(spa.WithHandler("/sse", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: one\n\n")
		_ = http.NewResponseController(w).Flush()
		select {
		case <-release:
		case <-time.After(5 * time.Second):
		}
		_, _ = io.WriteString(w, "data: two\n\n")
	})))...)
	base := listen(s.T.T(), app)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", base+"/sse", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	res, err := rawClient().Do(req)
	must(s.T.T(), err)
	defer res.Body.Close()

	s.T.Expect(res.Header.Get("Content-Encoding")).ToEqual("") // events are never compressed
	r := bufio.NewReader(res.Body)
	first, err := r.ReadString('\n')
	must(s.T.T(), err)
	s.T.Expect(first).ToEqual("data: one\n")

	close(release)
	r.ReadString('\n') // blank line ending event one
	second, err := r.ReadString('\n')
	must(s.T.T(), err)
	s.T.Expect(second).ToEqual("data: two\n")
}

// Flushing a compressed stream must flush the gzip block too, or the client waits.
func (s *SpaCompressionSuite) TestFlushDeliversACompressedChunkBeforeTheHandlerReturns() {
	release := make(chan struct{})
	app, _, _ := bootSpa(s.T.T(), everything(spa.WithHandler("/lines", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"n":1}`+"\n")
		_ = http.NewResponseController(w).Flush()
		select {
		case <-release:
		case <-time.After(5 * time.Second):
		}
		_, _ = io.WriteString(w, `{"n":2}`+"\n")
	})))...)
	base := listen(s.T.T(), app)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", base+"/lines", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	res, err := rawClient().Do(req)
	must(s.T.T(), err)
	defer res.Body.Close()
	s.T.Expect(res.Header.Get("Content-Encoding")).ToEqual("gzip")

	zr, err := gzip.NewReader(res.Body)
	must(s.T.T(), err)
	r := bufio.NewReader(zr)
	first, err := r.ReadString('\n')
	must(s.T.T(), err)
	s.T.Expect(first).ToEqual(`{"n":1}` + "\n")

	close(release)
	second, err := r.ReadString('\n')
	must(s.T.T(), err)
	s.T.Expect(second).ToEqual(`{"n":2}` + "\n")
}
