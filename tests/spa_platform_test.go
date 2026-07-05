package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/awesome-goose/goose"
	"github.com/awesome-goose/goose/platforms/spa"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

func TestSpaPlatform(t *testing.T) {
	test.NewSuiteRunner(t, &SpaPlatformSuite{}).Run()
}

type SpaPlatformSuite struct {
	test.Suite
	staticDir string
	seen      struct {
		method string
		paths  []string
	}
}

func (s *SpaPlatformSuite) SetupTest() {
	s.staticDir = s.T.T().TempDir()

	if err := os.WriteFile(filepath.Join(s.staticDir, "index.html"), []byte("SPA INDEX"), 0644); err != nil {
		s.T.T().Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(s.staticDir, "assets"), 0755); err != nil {
		s.T.T().Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.staticDir, "assets", "app.js"), []byte("console.log('app')"), 0644); err != nil {
		s.T.T().Fatal(err)
	}
	// A file outside the static root that traversal must never reach
	if err := os.WriteFile(filepath.Join(filepath.Dir(s.staticDir), "secret.txt"), []byte("TOP SECRET"), 0644); err != nil {
		s.T.T().Fatal(err)
	}
}

// newApp builds a booted spa.App with an echo handler that records the
// method and paths the kernel handler would see.
func (s *SpaPlatformSuite) newApp(options ...spa.Option) *spa.App {
	options = append([]spa.Option{spa.WithStaticDir(s.staticDir)}, options...)
	platform := spa.NewPlatform(options...)

	booted, err := platform.Boot(nil)
	s.T.Expect(err).ToBeNil()

	app := booted.(*spa.App)
	app.SetHandler(func(c types.Context) error {
		s.seen.method = string(c.Request().Method())
		s.seen.paths = c.Request().Paths()
		body := "METHOD=" + s.seen.method + " PATHS=" + strings.Join(s.seen.paths, ",")
		return c.Response().Write(types.SerialTypeString, []byte(body), 200)
	})

	return app
}

func (s *SpaPlatformSuite) TestStaticAssetIsServed() {
	h := test.NewHTTPTest(s.T.T(), s.newApp())

	res := h.GET("/assets/app.js").Do().ExpectOK()
	s.T.Expect(res.BodyString()).ToEqual("console.log('app')")
	s.T.Expect(res.Header.Get("Content-Type")).ToContainString("javascript")
}

func (s *SpaPlatformSuite) TestRootServesIndex() {
	h := test.NewHTTPTest(s.T.T(), s.newApp())

	res := h.GET("/").Do().ExpectOK()
	s.T.Expect(res.BodyString()).ToEqual("SPA INDEX")
}

func (s *SpaPlatformSuite) TestClientRouteFallsBackToIndex() {
	h := test.NewHTTPTest(s.T.T(), s.newApp())

	res := h.GET("/some/client/route").WithHeader("Accept", "text/html").Do().ExpectOK()
	s.T.Expect(res.BodyString()).ToEqual("SPA INDEX")
	s.T.Expect(res.Header.Get("Cache-Control")).ToEqual("no-cache")
}

func (s *SpaPlatformSuite) TestMissingAssetIs404NotIndex() {
	h := test.NewHTTPTest(s.T.T(), s.newApp())

	res := h.GET("/missing.png").Do().ExpectStatus(404)
	s.T.Expect(res.BodyString()).Not().ToContainString("SPA INDEX")
}

func (s *SpaPlatformSuite) TestNonHTMLAcceptDoesNotFallBack() {
	h := test.NewHTTPTest(s.T.T(), s.newApp())

	h.GET("/some/client/route").WithHeader("Accept", "application/json").Do().ExpectStatus(404)
}

func (s *SpaPlatformSuite) TestAPIRequestStripsPrefix() {
	h := test.NewHTTPTest(s.T.T(), s.newApp())

	res := h.GET("/api/users/42").Do().ExpectOK()
	s.T.Expect(res.BodyString()).ToEqual("METHOD=GET PATHS=users,42")
}

func (s *SpaPlatformSuite) TestAPIRootMapsToRootRoute() {
	h := test.NewHTTPTest(s.T.T(), s.newApp())

	res := h.GET("/api").Do().ExpectOK()
	s.T.Expect(res.BodyString()).ToEqual("METHOD=GET PATHS=")
}

func (s *SpaPlatformSuite) TestAPIAcceptsNonGETMethods() {
	h := test.NewHTTPTest(s.T.T(), s.newApp())

	res := h.POST("/api/users").WithString(`{}`).Do().ExpectOK()
	s.T.Expect(res.BodyString()).ToEqual("METHOD=POST PATHS=users")
}

func (s *SpaPlatformSuite) TestNonGETOutsideAPIIsMethodNotAllowed() {
	h := test.NewHTTPTest(s.T.T(), s.newApp())

	h.POST("/anything-else").WithString(`{}`).Do().ExpectStatus(405)
}

func (s *SpaPlatformSuite) TestPathTraversalNeverEscapesStaticRoot() {
	h := test.NewHTTPTest(s.T.T(), s.newApp())

	for _, path := range []string{"/../secret.txt", "/assets/../../secret.txt"} {
		res := h.GET(path).Do()
		s.T.Expect(res.StatusCode).Not().ToEqual(200)
		s.T.Expect(res.BodyString()).Not().ToContainString("TOP SECRET")
	}
}

func (s *SpaPlatformSuite) TestAPIPrefixIsNormalized() {
	h := test.NewHTTPTest(s.T.T(), s.newApp(spa.WithAPIPrefix("api/")))

	res := h.GET("/api/users").Do().ExpectOK()
	s.T.Expect(res.BodyString()).ToEqual("METHOD=GET PATHS=users")
}

func (s *SpaPlatformSuite) TestCustomIndexFile() {
	if err := os.WriteFile(filepath.Join(s.staticDir, "main.html"), []byte("CUSTOM INDEX"), 0644); err != nil {
		s.T.T().Fatal(err)
	}
	h := test.NewHTTPTest(s.T.T(), s.newApp(spa.WithIndexFile("main.html")))

	res := h.GET("/deep/route").WithHeader("Accept", "text/html").Do().ExpectOK()
	s.T.Expect(res.BodyString()).ToEqual("CUSTOM INDEX")
}

func (s *SpaPlatformSuite) TestFactorySetsPlatformType() {
	platform := spa.NewPlatform(spa.WithName("spa-test"))

	s.T.Expect(string(platform.Type())).ToEqual("spa")

	instance := goose.SPA(platform, nil, nil)
	s.T.Expect(string(instance.Type)).ToEqual(string(types.PlatformTypeSPA))
	s.T.Expect(instance.Name).ToEqual("spa-test")
}
