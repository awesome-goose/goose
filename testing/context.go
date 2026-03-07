package testing

import (
	"encoding/json"

	"github.com/awesome-goose/goose/types"
)

// MockContext implements types.Context for testing
type MockContext struct {
	request  *MockRequest
	response *MockResponse
	values   map[any]any
}

// NewMockContext creates a new mock context
func NewMockContext() *MockContext {
	return &MockContext{
		request:  NewMockRequest(),
		response: NewMockResponse(),
		values:   make(map[any]any),
	}
}

// Request returns the mock request
func (c *MockContext) Request() types.Request {
	return c.request
}

// Response returns the mock response
func (c *MockContext) Response() types.Response {
	return c.response
}

// SetValue stores a value in context
func (c *MockContext) SetValue(key any, value any) {
	c.values[key] = value
}

// GetValue retrieves a value from context
func (c *MockContext) GetValue(key any) any {
	return c.values[key]
}

// MockRequest returns the underlying mock request for configuration
func (c *MockContext) MockRequest() *MockRequest {
	return c.request
}

// MockResponse returns the underlying mock response for assertions
func (c *MockContext) MockResponse() *MockResponse {
	return c.response
}

// --- Mock Request ---

// MockRequest implements types.Request for testing
type MockRequest struct {
	method  types.Method
	paths   []string
	queries map[string]string
	params  map[string]string
	headers map[string][]string
	body    []byte
}

// NewMockRequest creates a new mock request
func NewMockRequest() *MockRequest {
	return &MockRequest{
		queries: make(map[string]string),
		params:  make(map[string]string),
		headers: make(map[string][]string),
	}
}

// Headers returns request headers
func (r *MockRequest) Headers() map[string][]string {
	return r.headers
}

// Method returns the HTTP method
func (r *MockRequest) Method() types.Method {
	return r.method
}

// Paths returns the path segments
func (r *MockRequest) Paths() []string {
	return r.paths
}

// Queries returns query parameters
func (r *MockRequest) Queries() map[string]string {
	return r.queries
}

// Params returns route parameters
func (r *MockRequest) Params() map[string]string {
	return r.params
}

// Body returns the request body
func (r *MockRequest) Body() ([]byte, error) {
	return r.body, nil
}

// PopulateParams sets route parameters
func (r *MockRequest) PopulateParams(params map[string]string) {
	r.params = params
}

// WithMethod sets the HTTP method
func (r *MockRequest) WithMethod(method types.Method) *MockRequest {
	r.method = method
	return r
}

// WithPaths sets the path segments
func (r *MockRequest) WithPaths(paths ...string) *MockRequest {
	r.paths = paths
	return r
}

// WithQueries sets the query parameters
func (r *MockRequest) WithQueries(queries map[string]string) *MockRequest {
	r.queries = queries
	return r
}

// WithParams sets the route parameters
func (r *MockRequest) WithParams(params map[string]string) *MockRequest {
	r.params = params
	return r
}

// WithHeaders sets the request headers
func (r *MockRequest) WithHeaders(headers map[string][]string) *MockRequest {
	r.headers = headers
	return r
}

// WithHeader adds a header
func (r *MockRequest) WithHeader(key string, values ...string) *MockRequest {
	r.headers[key] = values
	return r
}

// WithBody sets the request body
func (r *MockRequest) WithBody(body []byte) *MockRequest {
	r.body = body
	return r
}

// WithJSONBody sets the request body from a JSON object
func (r *MockRequest) WithJSONBody(v any) *MockRequest {
	data, _ := json.Marshal(v)
	r.body = data
	return r
}

// --- Mock Response ---

// MockResponse implements types.Response for testing
type MockResponse struct {
	statusCode int
	headers    map[string]string
	body       []byte
	serialType types.SerialType
	raw        any
}

// NewMockResponse creates a new mock response
func NewMockResponse() *MockResponse {
	return &MockResponse{
		headers: make(map[string]string),
	}
}

// Write writes the response body with the given serial type and status code
func (r *MockResponse) Write(serialType types.SerialType, outputData []byte, statusCode int) error {
	r.serialType = serialType
	r.body = outputData
	r.statusCode = statusCode
	return nil
}

// SetHeader sets a single response header
func (r *MockResponse) SetHeader(key, value string) {
	r.headers[key] = value
}

// SetHeaders sets multiple response headers
func (r *MockResponse) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		r.headers[k] = v
	}
}

// Raw returns the underlying response writer
func (r *MockResponse) Raw() any {
	return r.raw
}

// StatusCode returns the status code
func (r *MockResponse) StatusCode() int {
	return r.statusCode
}

// Headers returns the response headers
func (r *MockResponse) GetHeaders() map[string]string {
	return r.headers
}

// Body returns the response body
func (r *MockResponse) Body() []byte {
	return r.body
}

// SerialType returns the serial type
func (r *MockResponse) SerialType() types.SerialType {
	return r.serialType
}
