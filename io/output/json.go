package output

import (
	"net/http"
)

// JSONOutput represents a JSON API response with standardized structure.
// It provides a consistent response format: { "success": bool, "data": any, "message": string, "meta": any }
type JSONOutput struct {
	data        any
	code        int
	isSuccess   bool
	message     string
	meta        any
	headers     map[string]string
	contentType string
}

// JSONOutputOption is a function that configures a JSONOutput
type JSONOutputOption func(*JSONOutput)

// WithMeta adds metadata to the response (pagination, debug info, etc.)
func WithMeta(meta any) JSONOutputOption {
	return func(j *JSONOutput) {
		j.meta = meta
	}
}

// WithHeaders adds custom headers to the response
func WithHeaders(headers map[string]string) JSONOutputOption {
	return func(j *JSONOutput) {
		for k, v := range headers {
			j.headers[k] = v
		}
	}
}

// WithHeader adds a single custom header to the response
func WithHeader(key, value string) JSONOutputOption {
	return func(j *JSONOutput) {
		j.headers[key] = value
	}
}

// WithContentType overrides the default content type
func WithContentType(contentType string) JSONOutputOption {
	return func(j *JSONOutput) {
		j.contentType = contentType
	}
}

// NewJSONOutput creates a new JSON output with the given parameters
func NewJSONOutput(data any, code int, isSuccess bool, message string, meta any) *JSONOutput {
	return &JSONOutput{
		data:        data,
		code:        code,
		isSuccess:   isSuccess,
		message:     message,
		meta:        meta,
		headers:     make(map[string]string),
		contentType: "application/json; charset=utf-8",
	}
}

// JSON creates a success JSON response with data and optional configuration
func JSON(data any, opts ...JSONOutputOption) *JSONOutput {
	j := &JSONOutput{
		data:        data,
		code:        http.StatusOK,
		isSuccess:   true,
		message:     "",
		meta:        nil,
		headers:     make(map[string]string),
		contentType: "application/json; charset=utf-8",
	}
	for _, opt := range opts {
		opt(j)
	}
	return j
}

// JSONWithCode creates a success JSON response with a specific status code
func JSONWithCode(data any, code int, opts ...JSONOutputOption) *JSONOutput {
	j := &JSONOutput{
		data:        data,
		code:        code,
		isSuccess:   code >= 200 && code < 400,
		message:     "",
		meta:        nil,
		headers:     make(map[string]string),
		contentType: "application/json; charset=utf-8",
	}
	for _, opt := range opts {
		opt(j)
	}
	return j
}

// JSONError creates an error JSON response
func JSONError(message string, code int, opts ...JSONOutputOption) *JSONOutput {
	j := &JSONOutput{
		data:        nil,
		code:        code,
		isSuccess:   false,
		message:     message,
		meta:        nil,
		headers:     make(map[string]string),
		contentType: "application/json; charset=utf-8",
	}
	for _, opt := range opts {
		opt(j)
	}
	return j
}

// JSONErrorWithData creates an error JSON response that includes data (e.g., validation errors)
func JSONErrorWithData(message string, data any, code int, opts ...JSONOutputOption) *JSONOutput {
	j := &JSONOutput{
		data:        data,
		code:        code,
		isSuccess:   false,
		message:     message,
		meta:        nil,
		headers:     make(map[string]string),
		contentType: "application/json; charset=utf-8",
	}
	for _, opt := range opts {
		opt(j)
	}
	return j
}

// Data returns the structured JSON response
func (j *JSONOutput) Data() any {
	response := map[string]any{
		"success": j.isSuccess,
		"data":    j.data,
	}

	if j.message != "" {
		response["message"] = j.message
	}

	if j.meta != nil {
		response["meta"] = j.meta
	}

	return response
}

// Code returns the HTTP status code
func (j *JSONOutput) Code() int {
	return j.code
}

// Headers returns any custom headers
func (j *JSONOutput) Headers() map[string]string {
	return j.headers
}

// ContentType returns the content type for JSON
func (j *JSONOutput) ContentType() string {
	return j.contentType
}

// SetMessage sets the response message
func (j *JSONOutput) SetMessage(message string) *JSONOutput {
	j.message = message
	return j
}

// SetMeta sets the response metadata
func (j *JSONOutput) SetMeta(meta any) *JSONOutput {
	j.meta = meta
	return j
}

// SetHeader adds a header to the response
func (j *JSONOutput) SetHeader(key, value string) *JSONOutput {
	j.headers[key] = value
	return j
}
