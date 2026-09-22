package web

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/awesome-goose/goose/errors"
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
	config *Config
	fn     func(c types.Context) error
	server *http.Server
}

func NewApp(config *Config) *App {
	return &App{config: config}
}

// SetHandler sets the kernel handler. Run calls this automatically; it is
// exported so tests can exercise ServeHTTP without starting a server.
func (a *App) SetHandler(fn func(c types.Context) error) {
	a.fn = fn
}

func (a *App) Run(fn func(c types.Context) error) error {
	a.SetHandler(fn)

	// Calculate timeouts
	readTimeout := DefaultReadTimeout
	writeTimeout := DefaultWriteTimeout
	if a.config.Timeout > 0 {
		timeout := time.Duration(a.config.Timeout) * time.Second
		readTimeout = timeout
		writeTimeout = timeout
	}

	a.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", a.config.Host, a.config.Port),
		Handler:      a,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  DefaultIdleTimeout,
	}

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

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(w, r)
	err := a.fn(ctx)
	if err != nil {
		http.Error(w, err.Error(), statusFor(err))
		return
	}
}

// statusFor maps a kernel error to the HTTP status a caller should see.
// errors.ErrRouteNotFound — what core/router.go returns for a path with no
// matching route — is the one kernel error a live request can trigger simply
// by asking for something that doesn't exist, so it maps to 404. Every other
// error here represents an actual fault (a failed handler, a DI/config
// problem that slipped past boot) and stays 500, unchanged from before.
func statusFor(err error) int {
	if stderrors.Is(err, errors.ErrRouteNotFound) {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}
