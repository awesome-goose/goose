package spa

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
)

// pipeline assembles the request handler. Outermost first: recovery, security
// headers, CORS, compression, body limit, the caller's middleware (first is
// outermost), then routing. With no options set it is just routing.
func (a *App) pipeline() http.Handler {
	var h http.Handler = http.HandlerFunc(a.route)

	for i := len(a.config.Middlewares) - 1; i >= 0; i-- {
		h = a.config.Middlewares[i](h)
	}
	if a.config.BodyLimit > 0 {
		h = bodyLimit(a.config.BodyLimit, h)
	}
	if a.config.Compression {
		h = compression(h)
	}
	if a.config.CORS != nil {
		h = newCORS(a.config.CORS)(h)
	}
	if a.config.SecurityHeaders != nil {
		h = securityHeaders(a.config.SecurityHeaders, h)
	}
	if a.config.Recover {
		h = recovery(a.config.OnPanic, h)
	}
	return h
}

// validate rejects configuration that cannot work, at boot rather than at the
// first request.
func (c *Config) validate() error {
	for _, m := range c.Handlers {
		if !strings.HasPrefix(m.Path, "/") {
			return fmt.Errorf("spa: WithHandler path %q must start with \"/\"", m.Path)
		}
		if m.Handler == nil {
			return fmt.Errorf("spa: WithHandler(%q) has a nil handler", m.Path)
		}
	}
	if c.CORS != nil {
		if len(c.CORS.AllowedOrigins) == 0 {
			return errors.New("spa: WithCORS needs at least one allowed origin")
		}
		if c.CORS.AllowCredentials && contains(c.CORS.AllowedOrigins, "*") {
			return errors.New("spa: WithCORS cannot combine the \"*\" origin with credentials; list the origins")
		}
	}
	if c.BodyLimit < 0 {
		return errors.New("spa: WithBodyLimit must not be negative")
	}
	if c.WriteTimeoutSet && c.WriteTimeout < 0 {
		return errors.New("spa: WithWriteTimeout must not be negative")
	}
	return nil
}

// mounted returns the handler mounted for path, the longest match, or nil.
func (a *App) mounted(path string) http.Handler {
	var best http.Handler
	bestLen := -1
	for _, m := range a.config.Handlers {
		exact := m.Path == path
		subtree := strings.HasSuffix(m.Path, "/") && strings.HasPrefix(path, m.Path)
		if (exact || subtree) && len(m.Path) > bestLen {
			best, bestLen = m.Handler, len(m.Path)
		}
	}
	return best
}

// --- recovery --------------------------------------------------------------

func recovery(onPanic RecoveryFunc, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tw := &trackedWriter{ResponseWriter: w}
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v) // deliberate abort: net/http closes the connection quietly
			}
			if onPanic != nil {
				onPanic(r, v, debug.Stack())
			}
			if tw.started {
				// A status is already on the wire; the only honest answer left is
				// to cut the connection so the client sees a failed response.
				panic(http.ErrAbortHandler)
			}
			// Drop headers a layer set for a body that will never be sent.
			h := w.Header()
			h.Del("Content-Encoding")
			h.Del("Content-Length")
			h.Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"message":"internal server error"}`)
		}()
		next.ServeHTTP(tw, r)
	})
}

// --- security headers ------------------------------------------------------

func securityHeaders(s *SecurityHeaders, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		set := func(key, value string) {
			if value != "" {
				h.Set(key, value)
			}
		}
		set("Content-Security-Policy", s.ContentSecurityPolicy)
		set("Referrer-Policy", s.ReferrerPolicy)
		set("X-Frame-Options", s.FrameOptions)
		if s.NoSniff {
			h.Set("X-Content-Type-Options", "nosniff")
		}
		next.ServeHTTP(w, r)
	})
}

// --- CORS ------------------------------------------------------------------

var (
	defaultCORSMethods = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"}
	defaultCORSHeaders = []string{"Content-Type", "Authorization"}
)

func newCORS(c *CORS) func(http.Handler) http.Handler {
	methods := c.AllowedMethods
	if len(methods) == 0 {
		methods = defaultCORSMethods
	}
	headers := c.AllowedHeaders
	if len(headers) == 0 {
		headers = defaultCORSHeaders
	}
	anyOrigin := contains(c.AllowedOrigins, "*")
	originAllowed := func(origin string) bool { return anyOrigin || contains(c.AllowedOrigins, origin) }
	methodAllowed := func(m string) bool { return containsFold(methods, m) }
	headersAllowed := func(requested string) bool {
		for _, name := range strings.Split(requested, ",") {
			if name = strings.TrimSpace(name); name != "" && !containsFold(headers, name) {
				return false
			}
		}
		return true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			if !anyOrigin {
				addVary(h, "Origin") // the answer depends on the caller's Origin
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			allowed := originAllowed(origin)

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				addVary(h, "Access-Control-Request-Method")
				addVary(h, "Access-Control-Request-Headers")
				if !allowed ||
					!methodAllowed(r.Header.Get("Access-Control-Request-Method")) ||
					!headersAllowed(r.Header.Get("Access-Control-Request-Headers")) {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				setCORSOrigin(h, c, anyOrigin, origin)
				h.Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))
				h.Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
				if c.MaxAge > 0 {
					h.Set("Access-Control-Max-Age", strconv.Itoa(int(c.MaxAge.Seconds())))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			if allowed {
				setCORSOrigin(h, c, anyOrigin, origin)
				if len(c.ExposedHeaders) > 0 {
					h.Set("Access-Control-Expose-Headers", strings.Join(c.ExposedHeaders, ", "))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func setCORSOrigin(h http.Header, c *CORS, anyOrigin bool, origin string) {
	if anyOrigin {
		h.Set("Access-Control-Allow-Origin", "*")
	} else {
		h.Set("Access-Control-Allow-Origin", origin)
	}
	if c.AllowCredentials {
		h.Set("Access-Control-Allow-Credentials", "true")
	}
}

// --- body limit ------------------------------------------------------------

func bodyLimit(limit int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > limit {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Connection", "close")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_, _ = io.WriteString(w, `{"message":"request body too large"}`)
			return
		}
		if r.Body != nil {
			// Covers bodies of unknown length (chunked) and clients that lie.
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func containsFold(list []string, v string) bool {
	for _, s := range list {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}
