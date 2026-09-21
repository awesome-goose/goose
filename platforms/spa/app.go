package spa

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/awesome-goose/goose/platforms/api"
	"github.com/awesome-goose/goose/types"
)

// DefaultReadTimeout is the default read timeout for HTTP requests
const DefaultReadTimeout = 30 * time.Second

// DefaultWriteTimeout is the default write timeout for HTTP responses
const DefaultWriteTimeout = 30 * time.Second

// DefaultIdleTimeout is the default idle timeout for keep-alive connections
const DefaultIdleTimeout = 120 * time.Second

// DefaultShutdownTimeout is the default graceful shutdown timeout
const DefaultShutdownTimeout = 30 * time.Second

type App struct {
	config     *Config
	fn         func(c types.Context) error
	server     *http.Server
	apiPrefix  string
	staticRoot string
	handler    http.Handler
}

func NewApp(config *Config) *App {
	prefix := "/" + strings.Trim(config.APIPrefix, "/")
	if prefix == "/" {
		prefix = "/api"
	}

	// Resolve against the working directory, not the module root, so a
	// deployed binary next to its public/ folder keeps working.
	root, err := filepath.Abs(config.StaticDir)
	if err != nil {
		root = config.StaticDir
	}

	a := &App{config: config, apiPrefix: prefix, staticRoot: root}
	a.handler = a.pipeline()
	return a
}

// SetHandler sets the kernel handler. Run calls this automatically; it is
// exported so tests can exercise ServeHTTP without starting a server.
func (a *App) SetHandler(fn func(c types.Context) error) {
	a.fn = fn
}

// HTTPServer returns the configured http.Server without starting it. Run uses
// it; a caller with its own listener (or a test that needs the real timeouts)
// can Serve on it directly. The same server is returned on every call.
func (a *App) HTTPServer() *http.Server {
	if a.server != nil {
		return a.server
	}

	// Calculate timeouts
	readTimeout := DefaultReadTimeout
	writeTimeout := DefaultWriteTimeout
	if a.config.Timeout > 0 {
		timeout := time.Duration(a.config.Timeout) * time.Second
		readTimeout = timeout
		writeTimeout = timeout
	}
	if a.config.WriteTimeoutSet {
		writeTimeout = a.config.WriteTimeout
	}

	a.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", a.config.Host, a.config.Port),
		Handler:      a,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  DefaultIdleTimeout,
	}
	return a.server
}

func (a *App) Run(fn func(c types.Context) error) error {
	a.SetHandler(fn)
	a.HTTPServer()

	// Channel to signal server errors
	errChan := make(chan error, 1)

	// Start server in goroutine
	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case <-quit:
		return a.Shutdown()
	}
}

// Shutdown gracefully shuts down the server
func (a *App) Shutdown() error {
	if a.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()

	return a.server.Shutdown(ctx)
}

// ServeHTTP runs the request through the pipeline built from the platform's options.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}

// route sends a request to a mounted handler, the kernel (API prefix) or the static files.
func (a *App) route(w http.ResponseWriter, r *http.Request) {
	if h := a.mounted(r.URL.Path); h != nil {
		h.ServeHTTP(w, r)
		return
	}

	if p := r.URL.Path; p == a.apiPrefix || strings.HasPrefix(p, a.apiPrefix+"/") {
		a.serveAPI(w, r)
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.serveStatic(w, r)
}

// serveAPI strips the API prefix and hands the request to the kernel
// handler, so app modules declare routes exactly like an api application.
func (a *App) serveAPI(w http.ResponseWriter, r *http.Request) {
	stripped := strings.TrimPrefix(r.URL.Path, a.apiPrefix)
	if stripped == "" {
		stripped = "/"
	}

	r2 := new(http.Request)
	*r2 = *r
	u := *r.URL
	u.Path = stripped
	u.RawPath = ""
	r2.URL = &u

	ctx := api.NewContext(w, r2)
	if err := a.fn(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) serveStatic(w http.ResponseWriter, r *http.Request) {
	// Rooted clean prevents escaping staticRoot; http.ServeFile
	// additionally rejects any path containing "..".
	upath := path.Clean("/" + r.URL.Path)
	fsPath := filepath.Join(a.staticRoot, filepath.FromSlash(upath))

	if info, err := os.Stat(fsPath); err == nil && !info.IsDir() {
		if a.config.Compression && a.servePrecompressed(w, r, fsPath) {
			return
		}
		http.ServeFile(w, r, fsPath)
		return
	}

	if a.wantsHTML(r, upath) {
		indexPath := filepath.Join(a.staticRoot, a.config.IndexFile)
		if _, err := os.Stat(indexPath); err == nil {
			// The entry file must never be cached, or deploys won't propagate.
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFile(w, r, indexPath)
			return
		}
	}

	http.NotFound(w, r)
}

// wantsHTML reports whether the request looks like a client-side route:
// an extensionless path (or "/") from a client that accepts HTML. A missing
// asset like /logo.png stays a real 404 instead of returning index.html.
func (a *App) wantsHTML(r *http.Request, upath string) bool {
	if upath != "/" && path.Ext(upath) != "" {
		return false
	}

	accept := r.Header.Get("Accept")
	return accept == "" || strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*")
}
