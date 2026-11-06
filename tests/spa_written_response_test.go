package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/platforms/spa"
	"github.com/awesome-goose/goose/types"
)

// The SPA platform must leave an already-written API response alone when a
// middleware answers (an auth 401) and returns an error to stop the chain.
func TestSPADoesNotRewriteAnAlreadyWrittenAPIResponse(t *testing.T) {
	app, _, _ := bootSpa(t, spa.WithAPIPrefix("/api"))
	app.SetHandler(func(c types.Context) error {
		_ = c.Response().Write(types.SerialTypeObject, []byte(`{"success":false}`), http.StatusUnauthorized)
		return errors.ErrRouteNotFound
	})
	rec := serveOnce(app, "GET", "/api/x", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "route") || strings.Count(rec.Body.String(), "{") != 1 {
		t.Fatalf("body was rewritten: %q", rec.Body.String())
	}
}

// A 304 from a handler is bodyless and must not become a second 500.
func TestSPABodylessStatusesStayBodyless(t *testing.T) {
	app, _, _ := bootSpa(t, spa.WithAPIPrefix("/api"))
	app.SetHandler(func(c types.Context) error {
		return c.Response().Write(types.SerialTypeObject, []byte(`{"notModified":true}`), http.StatusNotModified)
	})
	rec := serveOnce(app, "GET", "/api/x", nil, nil)
	if rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
		t.Fatalf("got %d %q, want a bodyless 304", rec.Code, rec.Body.String())
	}
}
