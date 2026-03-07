package testing

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// HTTPTest provides HTTP testing utilities
type HTTPTest struct {
	t       *testing.T
	T       *T
	handler http.Handler
	server  *httptest.Server
	client  *http.Client
}

// NewHTTPTest creates a new HTTP test helper
func NewHTTPTest(t *testing.T, handler http.Handler) *HTTPTest {
	t.Helper()
	return &HTTPTest{
		t:       t,
		T:       New(t),
		handler: handler,
		client:  &http.Client{},
	}
}

// WithServer starts a test server (for integration/e2e tests)
func (h *HTTPTest) WithServer() *HTTPTest {
	h.server = httptest.NewServer(h.handler)
	h.client = h.server.Client()
	return h
}

// WithTLSServer starts a TLS test server
func (h *HTTPTest) WithTLSServer() *HTTPTest {
	h.server = httptest.NewTLSServer(h.handler)
	h.client = h.server.Client()
	return h
}

// Close closes the test server
func (h *HTTPTest) Close() {
	if h.server != nil {
		h.server.Close()
	}
}

// URL returns the test server URL
func (h *HTTPTest) URL() string {
	if h.server != nil {
		return h.server.URL
	}
	return ""
}

// Request creates a new HTTP request builder
func (h *HTTPTest) Request(method, path string) *HTTPRequest {
	return &HTTPRequest{
		test:    h,
		method:  method,
		path:    path,
		headers: make(http.Header),
	}
}

// GET creates a GET request
func (h *HTTPTest) GET(path string) *HTTPRequest {
	return h.Request("GET", path)
}

// POST creates a POST request
func (h *HTTPTest) POST(path string) *HTTPRequest {
	return h.Request("POST", path)
}

// PUT creates a PUT request
func (h *HTTPTest) PUT(path string) *HTTPRequest {
	return h.Request("PUT", path)
}

// DELETE creates a DELETE request
func (h *HTTPTest) DELETE(path string) *HTTPRequest {
	return h.Request("DELETE", path)
}

// PATCH creates a PATCH request
func (h *HTTPTest) PATCH(path string) *HTTPRequest {
	return h.Request("PATCH", path)
}

// HTTPRequest represents an HTTP request builder
type HTTPRequest struct {
	test    *HTTPTest
	method  string
	path    string
	body    io.Reader
	headers http.Header
	queries map[string]string
}

// WithBody sets the request body
func (r *HTTPRequest) WithBody(body io.Reader) *HTTPRequest {
	r.body = body
	return r
}

// WithJSON sets JSON body
func (r *HTTPRequest) WithJSON(v any) *HTTPRequest {
	data, err := json.Marshal(v)
	if err != nil {
		r.test.t.Fatalf("failed to marshal JSON: %v", err)
	}
	r.body = bytes.NewReader(data)
	r.headers.Set("Content-Type", "application/json")
	return r
}

// WithString sets string body
func (r *HTTPRequest) WithString(body string) *HTTPRequest {
	r.body = strings.NewReader(body)
	return r
}

// WithHeader adds a header
func (r *HTTPRequest) WithHeader(key, value string) *HTTPRequest {
	r.headers.Add(key, value)
	return r
}

// WithHeaders adds multiple headers
func (r *HTTPRequest) WithHeaders(headers map[string]string) *HTTPRequest {
	for k, v := range headers {
		r.headers.Add(k, v)
	}
	return r
}

// WithQuery adds a query parameter
func (r *HTTPRequest) WithQuery(key, value string) *HTTPRequest {
	if r.queries == nil {
		r.queries = make(map[string]string)
	}
	r.queries[key] = value
	return r
}

// WithBearer adds Bearer token authorization
func (r *HTTPRequest) WithBearer(token string) *HTTPRequest {
	r.headers.Set("Authorization", "Bearer "+token)
	return r
}

// WithBasicAuth adds Basic authentication
func (r *HTTPRequest) WithBasicAuth(user, pass string) *HTTPRequest {
	r.headers.Set("Authorization", "Basic "+basicAuth(user, pass))
	return r
}

// Do executes the request and returns the response
func (r *HTTPRequest) Do() *HTTPResponse {
	r.test.t.Helper()
	var req *http.Request
	var err error

	url := r.path
	if r.test.server != nil {
		url = r.test.server.URL + r.path
	}

	if r.queries != nil {
		q := ""
		for k, v := range r.queries {
			if q != "" {
				q += "&"
			}
			q += k + "=" + v
		}
		url += "?" + q
	}

	req, err = http.NewRequest(r.method, url, r.body)
	if err != nil {
		r.test.t.Fatalf("failed to create request: %v", err)
	}
	req.Header = r.headers

	var resp *http.Response
	if r.test.server != nil {
		resp, err = r.test.client.Do(req)
	} else {
		rr := httptest.NewRecorder()
		r.test.handler.ServeHTTP(rr, req)
		resp = rr.Result()
	}

	if err != nil {
		r.test.t.Fatalf("failed to execute request: %v", err)
	}

	return &HTTPResponse{
		test:     r.test,
		Response: resp,
	}
}

// HTTPResponse wraps http.Response with assertions
type HTTPResponse struct {
	test *HTTPTest
	*http.Response
	body []byte
}

// Body returns the response body
func (r *HTTPResponse) Body() []byte {
	if r.body == nil {
		defer r.Response.Body.Close()
		body, err := io.ReadAll(r.Response.Body)
		if err != nil {
			r.test.t.Fatalf("failed to read response body: %v", err)
		}
		r.body = body
	}
	return r.body
}

// BodyString returns the response body as string
func (r *HTTPResponse) BodyString() string {
	return string(r.Body())
}

// JSON unmarshals the response body
func (r *HTTPResponse) JSON(v any) *HTTPResponse {
	if err := json.Unmarshal(r.Body(), v); err != nil {
		r.test.t.Fatalf("failed to unmarshal JSON response: %v", err)
	}
	return r
}

// ExpectStatus asserts the status code
func (r *HTTPResponse) ExpectStatus(code int) *HTTPResponse {
	r.test.t.Helper()
	r.test.T.Expect(r.StatusCode).ToEqual(code)
	return r
}

// ExpectOK asserts status 200
func (r *HTTPResponse) ExpectOK() *HTTPResponse {
	return r.ExpectStatus(http.StatusOK)
}

// ExpectCreated asserts status 201
func (r *HTTPResponse) ExpectCreated() *HTTPResponse {
	return r.ExpectStatus(http.StatusCreated)
}

// ExpectNoContent asserts status 204
func (r *HTTPResponse) ExpectNoContent() *HTTPResponse {
	return r.ExpectStatus(http.StatusNoContent)
}

// ExpectBadRequest asserts status 400
func (r *HTTPResponse) ExpectBadRequest() *HTTPResponse {
	return r.ExpectStatus(http.StatusBadRequest)
}

// ExpectUnauthorized asserts status 401
func (r *HTTPResponse) ExpectUnauthorized() *HTTPResponse {
	return r.ExpectStatus(http.StatusUnauthorized)
}

// ExpectForbidden asserts status 403
func (r *HTTPResponse) ExpectForbidden() *HTTPResponse {
	return r.ExpectStatus(http.StatusForbidden)
}

// ExpectNotFound asserts status 404
func (r *HTTPResponse) ExpectNotFound() *HTTPResponse {
	return r.ExpectStatus(http.StatusNotFound)
}

// ExpectInternalServerError asserts status 500
func (r *HTTPResponse) ExpectInternalServerError() *HTTPResponse {
	return r.ExpectStatus(http.StatusInternalServerError)
}

// ExpectHeader asserts a header value
func (r *HTTPResponse) ExpectHeader(key, value string) *HTTPResponse {
	r.test.t.Helper()
	r.test.T.Expect(r.Header.Get(key)).ToEqual(value)
	return r
}

// ExpectContentType asserts Content-Type header
func (r *HTTPResponse) ExpectContentType(contentType string) *HTTPResponse {
	return r.ExpectHeader("Content-Type", contentType)
}

// ExpectJSON asserts Content-Type is application/json
func (r *HTTPResponse) ExpectJSON() *HTTPResponse {
	r.test.t.Helper()
	ct := r.Header.Get("Content-Type")
	r.test.T.Expect(strings.HasPrefix(ct, "application/json")).ToBeTrue()
	return r
}

// ExpectBody asserts the response body
func (r *HTTPResponse) ExpectBody(expected string) *HTTPResponse {
	r.test.t.Helper()
	r.test.T.Expect(r.BodyString()).ToEqual(expected)
	return r
}

// ExpectBodyContains asserts the body contains a substring
func (r *HTTPResponse) ExpectBodyContains(substr string) *HTTPResponse {
	r.test.t.Helper()
	r.test.T.Expect(r.BodyString()).ToContainString(substr)
	return r
}

// ExpectJSONEqual asserts JSON equality
func (r *HTTPResponse) ExpectJSONEqual(expected any) *HTTPResponse {
	r.test.t.Helper()
	var actual any
	r.JSON(&actual)
	r.test.T.Expect(actual).ToDeepEqual(expected)
	return r
}

func basicAuth(user, pass string) string {
	auth := user + ":" + pass
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
