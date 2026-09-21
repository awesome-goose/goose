package spa

import (
	"bufio"
	"compress/gzip"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// compression wraps next with gzip for clients that accept it. Responses that
// must not be compressed are passed through untouched (see compressWriter).
func compression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// An upgrade (WebSocket) leaves HTTP entirely; do not put a writer in front of it.
		if r.Header.Get("Upgrade") != "" {
			next.ServeHTTP(w, r)
			return
		}

		cw := &compressWriter{
			ResponseWriter: w,
			head:           r.Method == http.MethodHead,
			wantGzip:       acceptsEncoding(r.Header.Get("Accept-Encoding"), "gzip"),
		}
		next.ServeHTTP(cw, r)
		// Not deferred: after a panic there is nothing to finish, and committing
		// headers or closing the stream would stop recovery from answering 500.
		cw.finish()
	})
}

// compressWriter decides, when the response headers are about to be sent, whether
// to gzip the body. It defers WriteHeader until the first Write or Flush so it can
// see the final Content-Type (and sniff one if the handler set none, as net/http
// would, but on the uncompressed bytes).
type compressWriter struct {
	http.ResponseWriter
	head     bool
	wantGzip bool

	status   int // pending status, 0 until WriteHeader
	decided  bool
	hijacked bool
	gz       *gzip.Writer
}

func (w *compressWriter) WriteHeader(code int) {
	if w.decided {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	// Interim responses (100 Continue, 103 Early Hints) go out immediately.
	if code >= 100 && code < 200 && code != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.status = code
}

func (w *compressWriter) Write(p []byte) (int, error) {
	w.commit(p)
	if w.gz != nil {
		return w.gz.Write(p)
	}
	return w.ResponseWriter.Write(p)
}

// commit fixes the headers and sends them. first is the start of the body, used
// only to sniff a Content-Type when the handler set none.
func (w *compressWriter) commit(first []byte) {
	if w.decided {
		return
	}
	w.decided = true

	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	h := w.Header()

	ct := h.Get("Content-Type")
	if ct == "" && len(first) > 0 && bodyAllowed(status) {
		ct = http.DetectContentType(first)
		h.Set("Content-Type", ct)
	}

	if bodyAllowed(status) && !w.head &&
		h.Get("Content-Encoding") == "" && h.Get("Content-Range") == "" &&
		compressible(ct) {
		addVary(h, "Accept-Encoding")
		if w.wantGzip {
			h.Set("Content-Encoding", "gzip")
			h.Del("Content-Length")
			w.gz = gzip.NewWriter(w.ResponseWriter)
		}
	}

	w.ResponseWriter.WriteHeader(status)
}

func (w *compressWriter) Flush() {
	w.commit(nil)
	if w.gz != nil {
		_ = w.gz.Flush() // emit the pending gzip block, or the client waits for it
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *compressWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	w.hijacked = true
	return h.Hijack()
}

func (w *compressWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// finish sends headers for a handler that wrote no body and closes the gzip stream.
func (w *compressWriter) finish() {
	if w.hijacked {
		return
	}
	if !w.decided && w.status != 0 {
		w.commit(nil)
	}
	if w.gz != nil {
		_ = w.gz.Close()
	}
}

// bodyAllowed reports whether a response with this status may carry a body.
func bodyAllowed(status int) bool {
	return status >= 200 && status != http.StatusNoContent && status != http.StatusNotModified
}

// compressible reports whether a media type is worth gzipping.
func compressible(contentType string) bool {
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	switch {
	case media == "text/event-stream":
		return false // a stream is flushed event by event
	case strings.HasPrefix(media, "text/"),
		strings.HasSuffix(media, "+json"), strings.HasSuffix(media, "+xml"):
		return true
	}
	switch media {
	case "application/json", "application/javascript", "application/x-javascript",
		"application/ecmascript", "application/xml", "application/x-ndjson",
		"application/wasm", "image/svg+xml", "font/ttf", "font/otf",
		"application/vnd.ms-fontobject":
		return true
	}
	return false
}

// acceptsEncoding reports whether an Accept-Encoding header accepts enc. An
// explicit entry, including "gzip;q=0", wins over "*".
func acceptsEncoding(header, enc string) bool {
	star, explicit, seen := false, false, false
	for _, part := range strings.Split(header, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		name = strings.ToLower(strings.TrimSpace(name))
		if name != enc && name != "*" {
			continue
		}
		q := 1.0
		for _, p := range strings.Split(params, ";") {
			if v, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(p)), "q="); ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					q = f
				}
			}
		}
		if name == enc {
			seen, explicit = true, q > 0
		} else {
			star = q > 0
		}
	}
	if seen {
		return explicit
	}
	return star
}

// addVary adds a Vary value unless it is already listed.
func addVary(h http.Header, value string) {
	for _, line := range h.Values("Vary") {
		for _, v := range strings.Split(line, ",") {
			if strings.EqualFold(strings.TrimSpace(v), value) {
				return
			}
		}
	}
	h.Add("Vary", value)
}

// servePrecompressed serves "<file>.br" or "<file>.gz" in place of file when the
// client accepts that encoding, so a build that precompresses its assets is not
// recompressed on every request. It reports whether it served the request.
// Range requests are left to the identity file: a range of the encoded bytes is
// not what a client that sent one means.
func (a *App) servePrecompressed(w http.ResponseWriter, r *http.Request, fsPath string) bool {
	if r.Header.Get("Range") != "" {
		return false
	}
	accept := r.Header.Get("Accept-Encoding")

	served := false
	for _, variant := range []struct{ encoding, ext string }{{"br", ".br"}, {"gzip", ".gz"}} {
		sibling := fsPath + variant.ext
		info, err := os.Stat(sibling)
		if err != nil || info.IsDir() {
			continue
		}
		// The identity file has siblings, so caches must key on Accept-Encoding.
		addVary(w.Header(), "Accept-Encoding")
		if served || !acceptsEncoding(accept, variant.encoding) {
			continue
		}

		contentType := mime.TypeByExtension(filepath.Ext(fsPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType) // else ServeFile would sniff the compressed bytes
		w.Header().Set("Content-Encoding", variant.encoding)
		http.ServeFile(w, r, sibling)
		served = true
	}
	return served
}
