package tests

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/awesome-goose/goose/platforms/spa"
	test "github.com/awesome-goose/goose/testing"
)

func TestSpaPipeline(t *testing.T) {
	test.NewSuiteRunner(t, &SpaPipelineSuite{}).Run()
}

type SpaPipelineSuite struct {
	test.Suite
}

func (s *SpaPipelineSuite) TestDefaultsAddNoHeadersAndNoCompression() {
	app, _, _ := bootSpa(s.T.T())

	res := serveOnce(app, "GET", "/assets/app.js", spaHeaders{"Accept-Encoding": "gzip", "Origin": "https://a.example"}, nil)

	s.T.Expect(res.Code).ToEqual(200)
	for _, h := range []string{"Content-Encoding", "Vary", "Access-Control-Allow-Origin", "Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy"} {
		s.T.Expect(res.Header().Get(h)).ToEqual("")
	}
}

func (s *SpaPipelineSuite) TestDefaultPanicStillPropagates() {
	app, _, _ := bootSpa(s.T.T(), spa.WithHandler("/boom", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("kaboom")
	})))

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		serveOnce(app, "GET", "/boom", nil, nil)
	}()

	s.T.Expect(recovered).ToEqual("kaboom")
}

// --- WithHandler -----------------------------------------------------------

func (s *SpaPipelineSuite) TestMountedHandlersWinOverAPIAndStatic() {
	app, _, calls := bootSpa(s.T.T(),
		spa.WithHandler("/ws", text("ws")),
		spa.WithHandler("/api/stream", text("stream")),
		spa.WithHandler("/events/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "events:"+r.URL.Path)
		})),
		spa.WithHandler("/events/special/", text("special")),
	)

	s.T.Expect(serveOnce(app, "GET", "/ws", nil, nil).Body.String()).ToEqual("ws")
	// A mounted path is not subject to the "non-GET outside /api is 405" rule,
	// and it beats the API prefix.
	s.T.Expect(serveOnce(app, "POST", "/api/stream", nil, nil).Body.String()).ToEqual("stream")
	s.T.Expect(*calls).ToEqual(0)
	// A trailing slash mounts the subtree; the original path is preserved.
	s.T.Expect(serveOnce(app, "GET", "/events/a/b", nil, nil).Body.String()).ToEqual("events:/events/a/b")
	// The longest match wins.
	s.T.Expect(serveOnce(app, "GET", "/events/special/x", nil, nil).Body.String()).ToEqual("special")
	// No trailing slash means exact: /wsx is not /ws, so it falls through to the SPA.
	s.T.Expect(serveOnce(app, "GET", "/wsx", spaHeaders{"Accept": "text/html"}, nil).Body.String()).ToEqual("SPA INDEX")
	// Other API routes and static files are untouched.
	s.T.Expect(serveOnce(app, "GET", "/api/ping", nil, nil).Body.String()).ToEqual(`{"path":"ping"}`)
	s.T.Expect(serveOnce(app, "GET", "/assets/app.js", nil, nil).Body.String()).ToEqual(spaJS)
}

func (s *SpaPipelineSuite) TestMountedPathAndHandlerAreValidated() {
	s.T.Expect(bootSpaErr(spa.WithHandler("ws", text("x")))).Not().ToBeNil()
	s.T.Expect(bootSpaErr(spa.WithHandler("/ws", nil))).Not().ToBeNil()
	s.T.Expect(bootSpaErr(spa.WithHandler("/ws", text("x")))).ToBeNil()
}

// --- WithMiddleware --------------------------------------------------------

func (s *SpaPipelineSuite) TestMiddlewareRunsFirstToLastAndWrapsEveryRoute() {
	var order []string
	tag := func(name string) spa.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+":"+r.URL.Path)
				w.Header().Add("X-Seen-By", name)
				next.ServeHTTP(w, r)
			})
		}
	}
	app, _, _ := bootSpa(s.T.T(), spa.WithMiddleware(tag("a"), tag("b")), spa.WithHandler("/ws", text("ws")))

	for _, path := range []string{"/api/ping", "/assets/app.js", "/ws"} {
		order = nil
		res := serveOnce(app, "GET", path, nil, nil)
		s.T.Expect(res.Header().Values("X-Seen-By")).ToEqual([]string{"a", "b"})
		// Middleware sees the original path, before the API prefix is stripped.
		s.T.Expect(order).ToEqual([]string{"a:" + path, "b:" + path})
	}
}

func (s *SpaPipelineSuite) TestMiddlewareCanShortCircuit() {
	deny := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "nope", http.StatusUnauthorized)
		})
	}
	app, _, calls := bootSpa(s.T.T(), spa.WithMiddleware(deny))

	res := serveOnce(app, "GET", "/api/secret", nil, nil)

	s.T.Expect(res.Code).ToEqual(401)
	s.T.Expect(*calls).ToEqual(0)
}

// --- WithBodyLimit ---------------------------------------------------------

func (s *SpaPipelineSuite) TestBodyLimitRejectsADeclaredOversizeBodyBeforeTheHandler() {
	reached := false
	app, _, _ := bootSpa(s.T.T(), spa.WithBodyLimit(5), spa.WithHandler("/upload", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	})))

	res := serveOnce(app, "POST", "/upload", nil, strings.NewReader("0123456789"))

	s.T.Expect(res.Code).ToEqual(413)
	s.T.Expect(reached).ToEqual(false)
}

func (s *SpaPipelineSuite) TestBodyLimitCutsAnUndeclaredBodyAtTheLimit() {
	var readErr error
	var got string
	app, _, _ := bootSpa(s.T.T(), spa.WithBodyLimit(10), spa.WithHandler("/upload", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		got, readErr = string(b), err
	})))

	req := newRequest("POST", "/upload", strings.NewReader(strings.Repeat("x", 100)))
	req.ContentLength = -1 // as with chunked transfer encoding
	app.ServeHTTP(newRecorder(), req)

	var tooLarge *http.MaxBytesError
	s.T.Expect(errors.As(readErr, &tooLarge)).ToEqual(true)
	s.T.Expect(len(got) <= 10).ToEqual(true)
}

func (s *SpaPipelineSuite) TestBodyWithinTheLimitPasses() {
	app, _, _ := bootSpa(s.T.T(), spa.WithBodyLimit(10), spa.WithHandler("/upload", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_, _ = w.Write(b)
	})))

	res := serveOnce(app, "POST", "/upload", nil, strings.NewReader("hello"))

	s.T.Expect(res.Code).ToEqual(200)
	s.T.Expect(res.Body.String()).ToEqual("hello")
}

// --- WithCORS --------------------------------------------------------------

func corsApp(t *testing.T, cors spa.CORS) (*spa.App, *int) {
	app, _, calls := bootSpa(t, spa.WithCORS(cors))
	return app, calls
}

func (s *SpaPipelineSuite) TestCORSAnswersOnlyListedOrigins() {
	app, _ := corsApp(s.T.T(), spa.CORS{
		AllowedOrigins:   []string{"https://app.example"},
		AllowCredentials: true,
		ExposedHeaders:   []string{"X-Total"},
	})

	ok := serveOnce(app, "GET", "/api/x", spaHeaders{"Origin": "https://app.example"}, nil)
	s.T.Expect(ok.Header().Get("Access-Control-Allow-Origin")).ToEqual("https://app.example")
	s.T.Expect(ok.Header().Get("Access-Control-Allow-Credentials")).ToEqual("true")
	s.T.Expect(ok.Header().Get("Access-Control-Expose-Headers")).ToEqual("X-Total")
	s.T.Expect(ok.Header().Values("Vary")).ToContain("Origin")

	other := serveOnce(app, "GET", "/api/x", spaHeaders{"Origin": "https://evil.example"}, nil)
	s.T.Expect(other.Code).ToEqual(200) // still served; the browser enforces CORS
	s.T.Expect(other.Header().Get("Access-Control-Allow-Origin")).ToEqual("")
	s.T.Expect(other.Header().Get("Access-Control-Allow-Credentials")).ToEqual("")
	s.T.Expect(other.Header().Values("Vary")).ToContain("Origin")

	none := serveOnce(app, "GET", "/api/x", nil, nil)
	s.T.Expect(none.Header().Get("Access-Control-Allow-Origin")).ToEqual("")
}

func (s *SpaPipelineSuite) TestCORSPreflight() {
	app, calls := corsApp(s.T.T(), spa.CORS{
		AllowedOrigins: []string{"https://app.example"},
		MaxAge:         10 * time.Minute,
	})
	preflight := func(origin, method, headers string) *responseRecord {
		h := spaHeaders{"Origin": origin, "Access-Control-Request-Method": method}
		if headers != "" {
			h["Access-Control-Request-Headers"] = headers
		}
		return serveOnce(app, "OPTIONS", "/api/x", h, nil)
	}

	ok := preflight("https://app.example", "DELETE", "Authorization")
	s.T.Expect(ok.Code).ToEqual(204)
	s.T.Expect(ok.Header().Get("Access-Control-Allow-Origin")).ToEqual("https://app.example")
	s.T.Expect(ok.Header().Get("Access-Control-Allow-Methods")).ToContainString("DELETE")
	s.T.Expect(ok.Header().Get("Access-Control-Allow-Headers")).ToContainString("Authorization")
	s.T.Expect(ok.Header().Get("Access-Control-Max-Age")).ToEqual("600")

	s.T.Expect(preflight("https://evil.example", "GET", "").Code).ToEqual(403)
	s.T.Expect(preflight("https://app.example", "TRACE", "").Code).ToEqual(403)
	s.T.Expect(preflight("https://app.example", "GET", "X-Not-Allowed").Code).ToEqual(403)
	// The kernel never saw a preflight.
	s.T.Expect(*calls).ToEqual(0)
}

func (s *SpaPipelineSuite) TestCORSWildcardAllowsAnyOriginWithoutCredentials() {
	app, _ := corsApp(s.T.T(), spa.CORS{AllowedOrigins: []string{"*"}})

	res := serveOnce(app, "GET", "/api/x", spaHeaders{"Origin": "https://any.example"}, nil)

	s.T.Expect(res.Header().Get("Access-Control-Allow-Origin")).ToEqual("*")
}

func (s *SpaPipelineSuite) TestCORSWildcardWithCredentialsIsRejectedAtBoot() {
	err := bootSpaErr(spa.WithCORS(spa.CORS{AllowedOrigins: []string{"*"}, AllowCredentials: true}))

	if err == nil {
		s.T.FailNow("Boot accepted a wildcard origin combined with credentials")
	}
	s.T.Expect(err.Error()).ToContainString("credentials")
}

// --- WithSecurityHeaders ---------------------------------------------------

func (s *SpaPipelineSuite) TestDefaultSecurityHeaders() {
	d := spa.DefaultSecurityHeaders()

	s.T.Expect(d.NoSniff).ToEqual(true)
	s.T.Expect(d.ReferrerPolicy).ToEqual("strict-origin-when-cross-origin")
	s.T.Expect(d.FrameOptions).ToEqual("SAMEORIGIN")
	s.T.Expect(d.ContentSecurityPolicy).ToEqual("") // app-specific, never guessed
}

func (s *SpaPipelineSuite) TestSecurityHeadersOnEveryRouteAndOverridableByTheHandler() {
	headers := spa.DefaultSecurityHeaders()
	headers.ContentSecurityPolicy = "default-src 'self'"
	app, _, _ := bootSpa(s.T.T(), spa.WithSecurityHeaders(headers),
		spa.WithHandler("/framed", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Frame-Options", "DENY")
		})),
		spa.WithHandler("/ws", text("ws")))

	for _, path := range []string{"/api/ping", "/assets/app.js", "/ws"} {
		res := serveOnce(app, "GET", path, nil, nil)
		s.T.Expect(res.Header().Get("Content-Security-Policy")).ToEqual("default-src 'self'")
		s.T.Expect(res.Header().Get("X-Content-Type-Options")).ToEqual("nosniff")
		s.T.Expect(res.Header().Get("Referrer-Policy")).ToEqual("strict-origin-when-cross-origin")
		s.T.Expect(res.Header().Get("X-Frame-Options")).ToEqual("SAMEORIGIN")
	}
	s.T.Expect(serveOnce(app, "GET", "/framed", nil, nil).Header().Get("X-Frame-Options")).ToEqual("DENY")
}

// --- WithRecovery ----------------------------------------------------------

func (s *SpaPipelineSuite) TestRecoveryAnswers500WithoutLeakingThePanic() {
	var gotValue any
	var gotStack []byte
	app, _, _ := bootSpa(s.T.T(),
		spa.WithRecovery(func(r *http.Request, v any, stack []byte) { gotValue, gotStack = v, stack }),
		spa.WithHandler("/boom", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("secret-detail")
		})))

	res := serveOnce(app, "GET", "/boom", nil, nil)

	s.T.Expect(res.Code).ToEqual(500)
	s.T.Expect(res.Header().Get("Content-Type")).ToContainString("application/json")
	s.T.Expect(strings.Contains(res.Body.String(), "secret-detail")).ToEqual(false)
	s.T.Expect(gotValue).ToEqual("secret-detail")
	s.T.Expect(len(gotStack) > 0).ToEqual(true)
}

func (s *SpaPipelineSuite) TestRecoveryWorksWithoutACallback() {
	app, _, _ := bootSpa(s.T.T(), spa.WithRecovery(nil), spa.WithHandler("/boom", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("x")
	})))

	s.T.Expect(serveOnce(app, "GET", "/boom", nil, nil).Code).ToEqual(500)
}

// Once the response has started there is no status left to change: recovery must
// abort the connection (the net/http convention) rather than append an error body.
func (s *SpaPipelineSuite) TestRecoveryAbortsAResponseThatAlreadyStarted() {
	app, _, _ := bootSpa(s.T.T(), spa.WithRecovery(nil), spa.WithHandler("/half", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "partial")
		panic("late")
	})))

	var recovered any
	rec := newRecorder()
	func() {
		defer func() { recovered = recover() }()
		app.ServeHTTP(rec, newRequest("GET", "/half", nil))
	}()

	s.T.Expect(recovered).ToEqual(http.ErrAbortHandler)
	s.T.Expect(rec.Body.String()).ToEqual("partial")
}
