package output

import (
	"net/http"
)

// Output is a generic output that can hold any data
// Use more specific output types (JSONOutput, HTMLOutput, etc.) for better features
type Output struct {
	data        any
	code        int
	isError     bool
	headers     map[string]string
	contentType string
}

// OutputOption configures an Output
type OutputOption func(*Output)

// NewOutput creates a basic output (legacy constructor)
func NewOutput(data any, code int, isError bool) *Output {
	return &Output{
		data:        data,
		code:        code,
		isError:     isError,
		headers:     make(map[string]string),
		contentType: "",
	}
}

// Raw creates a raw output with any data type
func Raw(data any, opts ...OutputOption) *Output {
	o := &Output{
		data:        data,
		code:        http.StatusOK,
		isError:     false,
		headers:     make(map[string]string),
		contentType: "",
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// RawWithCode creates a raw output with a specific status code
func RawWithCode(data any, code int, opts ...OutputOption) *Output {
	o := &Output{
		data:        data,
		code:        code,
		isError:     code >= 400,
		headers:     make(map[string]string),
		contentType: "",
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Text creates a plain text response
func Text(content string, opts ...OutputOption) *Output {
	o := &Output{
		data:        content,
		code:        http.StatusOK,
		isError:     false,
		headers:     make(map[string]string),
		contentType: "text/plain; charset=utf-8",
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Empty creates an empty response with a status code
func Empty(code int) *Output {
	return &Output{
		data:        nil,
		code:        code,
		isError:     code >= 400,
		headers:     make(map[string]string),
		contentType: "",
	}
}

// WithRawHeaders adds custom headers to raw output
func WithRawHeaders(headers map[string]string) OutputOption {
	return func(o *Output) {
		for k, v := range headers {
			o.headers[k] = v
		}
	}
}

// WithRawContentType sets the content type for raw output
func WithRawContentType(contentType string) OutputOption {
	return func(o *Output) {
		o.contentType = contentType
	}
}

// Data returns the output data
func (o *Output) Data() any {
	return o.data
}

// Code returns the HTTP status code
func (o *Output) Code() int {
	return o.code
}

// Headers returns custom headers
func (o *Output) Headers() map[string]string {
	return o.headers
}

// ContentType returns the content type
func (o *Output) ContentType() string {
	return o.contentType
}

// SetData updates the output data
func (o *Output) SetData(data any) *Output {
	o.data = data
	return o
}

// SetCode updates the status code
func (o *Output) SetCode(code int) *Output {
	o.code = code
	return o
}

// SetHeader adds a header
func (o *Output) SetHeader(key, value string) *Output {
	o.headers[key] = value
	return o
}

// SetContentType sets the content type
func (o *Output) SetContentType(contentType string) *Output {
	o.contentType = contentType
	return o
}
