package spa

import (
	"net/http"
	"time"
)

type Config struct {
	Name        string
	Version     string
	Author      string
	Description string

	Host    string
	Port    int
	Timeout int

	// StaticDir is the directory served for non-API requests. Relative
	// paths resolve against the working directory of the running binary.
	StaticDir string
	// IndexFile is the SPA entry file within StaticDir served as a
	// fallback for client-side routes.
	IndexFile string
	// APIPrefix is the URL prefix routed to the goose kernel. Routes are
	// declared without the prefix.
	APIPrefix string

	// WriteTimeout, when WriteTimeoutSet, overrides the server write timeout
	// (0 disables it). See WithWriteTimeout.
	WriteTimeout    time.Duration
	WriteTimeoutSet bool

	// BodyLimit caps request bodies in bytes; 0 means no limit.
	BodyLimit int64
	// CORS is nil unless WithCORS was used: no CORS headers are sent by default.
	CORS *CORS
	// SecurityHeaders is nil unless WithSecurityHeaders was used.
	SecurityHeaders *SecurityHeaders
	// Recover turns handler panics into a 500 response; OnPanic is optional.
	Recover bool
	OnPanic RecoveryFunc
	// Compression enables gzip responses and precompressed static siblings.
	Compression bool
	// Middlewares wrap the whole request pipeline, first is outermost.
	Middlewares []Middleware
	// Handlers are mounted ahead of the API prefix and the static files.
	Handlers []Mount
}

type Option func(*Config)

// Middleware wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Mount is an http.Handler mounted at a path. A path ending in "/" matches that
// whole subtree; any other path matches exactly. The longest match wins.
type Mount struct {
	Path    string
	Handler http.Handler
}

// RecoveryFunc is called with the recovered value and the stack of a panicking
// handler, before the 500 response is written.
type RecoveryFunc func(r *http.Request, recovered any, stack []byte)

// CORS configures cross-origin access. Only origins listed in AllowedOrigins are
// answered; "*" allows any origin but cannot be combined with AllowCredentials.
type CORS struct {
	AllowedOrigins   []string
	AllowedMethods   []string // default: GET, HEAD, POST, PUT, PATCH, DELETE
	AllowedHeaders   []string // default: Content-Type, Authorization
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration // preflight cache lifetime; 0 omits the header
}

// SecurityHeaders lists response headers set on every response. Empty fields are
// not sent. A handler may still override a header it sets itself.
type SecurityHeaders struct {
	ContentSecurityPolicy string
	ReferrerPolicy        string
	FrameOptions          string
	// NoSniff sends X-Content-Type-Options: nosniff.
	NoSniff bool
}

// DefaultSecurityHeaders is a conservative set with no Content-Security-Policy:
// a policy depends on the app's assets and must be supplied by it.
func DefaultSecurityHeaders() SecurityHeaders {
	return SecurityHeaders{
		ReferrerPolicy: "strict-origin-when-cross-origin",
		FrameOptions:   "SAMEORIGIN",
		NoSniff:        true,
	}
}

// WithName sets the name of the .
func WithName(name string) Option {
	return func(Config *Config) {
		Config.Name = name
	}
}

// WithVersion sets the version of the .
func WithVersion(version string) Option {
	return func(Config *Config) {
		Config.Version = version
	}
}

// WithAuthor sets the author of the .
func WithAuthor(author string) Option {
	return func(Config *Config) {
		Config.Author = author
	}
}

// WithDescription sets the description of the .
func WithDescription(description string) Option {
	return func(Config *Config) {
		Config.Description = description
	}
}

func WithHost(host string) Option {
	return func(Config *Config) {
		Config.Host = host
	}
}

func WithPort(port int) Option {
	return func(Config *Config) {
		Config.Port = port
	}
}

func WithTimeout(timeout int) Option {
	return func(Config *Config) {
		Config.Timeout = timeout
	}
}

func WithStaticDir(dir string) Option {
	return func(Config *Config) {
		Config.StaticDir = dir
	}
}

func WithIndexFile(file string) Option {
	return func(Config *Config) {
		Config.IndexFile = file
	}
}

func WithAPIPrefix(prefix string) Option {
	return func(Config *Config) {
		Config.APIPrefix = prefix
	}
}

// WithWriteTimeout sets the server write timeout; 0 disables it. It takes
// precedence over WithTimeout's write half. A streamed response (SSE, downloads)
// is cut when it outlives this timeout unless its handler clears the deadline
// with http.NewResponseController(w).SetWriteDeadline(time.Time{}).
func WithWriteTimeout(d time.Duration) Option {
	return func(Config *Config) {
		Config.WriteTimeout = d
		Config.WriteTimeoutSet = true
	}
}

// WithBodyLimit rejects request bodies larger than n bytes (413 when the size is
// declared up front, a read error otherwise). 0 means no limit.
func WithBodyLimit(n int64) Option {
	return func(Config *Config) {
		Config.BodyLimit = n
	}
}

// WithCORS enables CORS for the listed origins. See CORS.
func WithCORS(cors CORS) Option {
	return func(Config *Config) {
		Config.CORS = &cors
	}
}

// WithSecurityHeaders sets the given headers on every response.
func WithSecurityHeaders(headers SecurityHeaders) Option {
	return func(Config *Config) {
		Config.SecurityHeaders = &headers
	}
}

// WithRecovery turns a handler panic into a 500 JSON response that does not
// include the panic value. onPanic, if not nil, receives the value and stack.
// Without this option a panic propagates to net/http as before.
func WithRecovery(onPanic RecoveryFunc) Option {
	return func(Config *Config) {
		Config.Recover = true
		Config.OnPanic = onPanic
	}
}

// WithCompression gzips compressible responses for clients that accept it and
// serves precompressed "<file>.br" / "<file>.gz" siblings of static files.
// Server-sent events, WebSocket upgrades, range requests and responses that
// already carry a Content-Encoding are never compressed, and Flush and Hijack
// keep working through it.
func WithCompression() Option {
	return func(Config *Config) {
		Config.Compression = true
	}
}

// WithMiddleware wraps the whole pipeline (API, static files and mounted
// handlers). The first middleware is the outermost. Recovery, security headers,
// CORS, compression and the body limit run outside all of them.
func WithMiddleware(middlewares ...Middleware) Option {
	return func(Config *Config) {
		Config.Middlewares = append(Config.Middlewares, middlewares...)
	}
}

// WithHandler mounts an http.Handler (a WebSocket endpoint, an SSE stream, a
// webhook) ahead of the API prefix and the static files. A path ending in "/"
// matches its whole subtree, any other path matches exactly. Mounted handlers
// accept every method.
func WithHandler(path string, handler http.Handler) Option {
	return func(Config *Config) {
		Config.Handlers = append(Config.Handlers, Mount{Path: path, Handler: handler})
	}
}
