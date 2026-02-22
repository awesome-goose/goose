package cli

import (
	"os"

	"github.com/awesome-goose/goose/types"
)

type Response struct {
	raw     *os.File
	headers map[string]string
}

func NewResponse(raw *os.File) *Response {
	return &Response{
		raw:     raw,
		headers: make(map[string]string),
	}
}

func (r *Response) Write(serialType types.SerialType, data []byte, statusCode int) error {
	var output []byte
	switch serialType {
	case types.SerialTypeString:
		output = data
	case types.SerialTypeBool, types.SerialTypeNumber:
		output = data
	case types.SerialTypeBinary:
		output = data
	case types.SerialTypeObject:
		output = data
	case types.SerialTypeNil:
		output = []byte{}
	case types.SerialTypeError:
		output = data
	default:
		output = data
	}
	if len(output) > 0 {
		_, err := r.raw.Write(output)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetHeader stores a header (no-op for CLI, but maintains interface compatibility)
func (r *Response) SetHeader(key, value string) {
	r.headers[key] = value
}

// SetHeaders stores multiple headers
func (r *Response) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		r.headers[k] = v
	}
}

// Raw returns the underlying os.File
func (r *Response) Raw() any {
	return r.raw
}
