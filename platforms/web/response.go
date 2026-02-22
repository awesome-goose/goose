package web

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/props"
)

type HTMLResponse struct {
	Functions props.Props
	Templates []string
	Data      any
}

type Response struct {
	raw http.ResponseWriter
}

func NewResponse(raw http.ResponseWriter) *Response {
	return &Response{raw}
}

func (r *Response) Write(serialType types.SerialType, outputData []byte, statusCode int) error {
	htmlResp, ok := any(outputData).(HTMLResponse)
	if ok {
		if r.raw.Header().Get("Content-Type") == "" {
			r.raw.Header().Set("Content-Type", "text/html; charset=utf-8")
		}

		t, err := template.New("platform_web.html").Funcs(template.FuncMap(htmlResp.Functions)).ParseFiles(htmlResp.Templates...)
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, htmlResp.Data); err != nil {
			return err
		}

		outputData = buf.Bytes()
	} else {
		// Set default content type if not already set
		if r.raw.Header().Get("Content-Type") == "" {
			switch serialType {
			case types.SerialTypeString:
				r.raw.Header().Set("Content-Type", "text/plain; charset=utf-8")
			case types.SerialTypeBool, types.SerialTypeNumber:
				r.raw.Header().Set("Content-Type", "text/plain; charset=utf-8")
			case types.SerialTypeBinary:
				r.raw.Header().Set("Content-Type", "application/octet-stream")
			case types.SerialTypeObject:
				r.raw.Header().Set("Content-Type", "application/json; charset=utf-8")
			case types.SerialTypeHTML:
				r.raw.Header().Set("Content-Type", "text/html; charset=utf-8")
			case types.SerialTypeFile:
				r.raw.Header().Set("Content-Type", "application/octet-stream")
			case types.SerialTypeNil:
				r.raw.Header().Set("Content-Type", "application/json; charset=utf-8")
			default:
				r.raw.Header().Set("Content-Type", "application/json; charset=utf-8")
			}
		}
	}

	r.raw.WriteHeader(statusCode)
	if len(outputData) > 0 {
		_, err := r.raw.Write(outputData)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetHeader sets a single response header
func (r *Response) SetHeader(key, value string) {
	r.raw.Header().Set(key, value)
}

// SetHeaders sets multiple response headers
func (r *Response) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		r.raw.Header().Set(k, v)
	}
}

// Raw returns the underlying http.ResponseWriter
func (r *Response) Raw() any {
	return r.raw
}
