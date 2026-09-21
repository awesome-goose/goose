package spa

import (
	"bufio"
	"net"
	"net/http"
)

// trackedWriter forwards everything a handler may ask of the connection: Flush
// for streams, Hijack for WebSocket upgrades, and Unwrap so
// http.NewResponseController can reach the real writer (per-request deadlines).
// It records whether the response has started, which decides whether a panic can
// still be answered with a 500.
//
// Every wrapper in this package must keep all three: dropping one breaks
// streaming, WebSockets or deadlines without any error.
type trackedWriter struct {
	http.ResponseWriter
	started bool
}

func (w *trackedWriter) WriteHeader(code int) {
	// 1xx responses other than 101 are interim; the final response is still to come.
	if code < 100 || code >= 200 || code == http.StatusSwitchingProtocols {
		w.started = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *trackedWriter) Write(p []byte) (int, error) {
	w.started = true
	return w.ResponseWriter.Write(p)
}

func (w *trackedWriter) Flush() {
	w.started = true
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *trackedWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	w.started = true
	return h.Hijack()
}

func (w *trackedWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
