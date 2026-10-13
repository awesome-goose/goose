package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/platforms/api"
	"github.com/awesome-goose/goose/types"
)

// A middleware that writes its own complete response and returns an error
// only to stop the chain must not get a second WriteHeader / the error text
// appended to its body.
func TestAPIDoesNotRewriteAnAlreadyWrittenResponse(t *testing.T) {
	app := api.NewApp(&api.Config{Host: "127.0.0.1", Port: 0})
	app.SetHandler(func(c types.Context) error {
		_ = c.Response().Write(types.SerialTypeObject, []byte(`{"success":false}`), http.StatusUnauthorized)
		return errors.ErrRouteNotFound // any error: the response is already out
	})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))

	body, _ := io.ReadAll(rec.Result().Body)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if strings.Contains(string(body), "route") || strings.Count(string(body), "{") != 1 {
		t.Fatalf("body was rewritten after the handler had responded: %q", body)
	}
}
