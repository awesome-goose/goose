package tests

import (
	"testing"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/platforms/api"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

func TestApiPlatform(t *testing.T) {
	test.NewSuiteRunner(t, &ApiPlatformSuite{}).Run()
}

type ApiPlatformSuite struct {
	test.Suite
}

func (s *ApiPlatformSuite) newApp(handler func(c types.Context) error) *api.App {
	app := api.NewApp(&api.Config{Host: "127.0.0.1", Port: 0})
	app.SetHandler(handler)
	return app
}

// App.ServeHTTP hard-mapped every kernel error to 500, including
// errors.ErrRouteNotFound — what core/router.go returns for a path with no
// matching route. A client hitting a route that doesn't exist got "500
// Internal Server Error" instead of 404, indistinguishable from a real fault.
func (s *ApiPlatformSuite) TestRouteNotFoundIs404() {
	h := test.NewHTTPTest(s.T.T(), s.newApp(func(c types.Context) error {
		return errors.ErrRouteNotFound.WithMeta(map[string]any{"method": "GET", "path": "nope"})
	}))

	h.GET("/nope").Do().ExpectStatus(404)
}

// Every other kernel error keeps its prior behavior.
func (s *ApiPlatformSuite) TestOtherErrorsStayInternalServerError() {
	h := test.NewHTTPTest(s.T.T(), s.newApp(func(c types.Context) error {
		return errors.ErrRuntimeError.WithError(nil)
	}))

	h.GET("/boom").Do().ExpectStatus(500)
}
