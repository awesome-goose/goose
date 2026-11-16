package api

import (
	"context"
	"net/http"

	"github.com/awesome-goose/goose/types"
)

type Context struct {
	request  *Request
	response *Response
}

func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		request:  NewRequest(r),
		response: NewResponse(w),
	}
}

func (c *Context) Request() types.Request {
	return c.request
}

func (c *Context) Response() types.Response {
	return c.response
}

// Context returns the request's context.Context (types.ContextCarrier): it is
// cancelled when the client disconnects or the server shuts down, and carries
// the values set with SetValue.
func (c *Context) Context() context.Context {
	return c.request.raw.Context()
}

func (c *Context) SetValue(key any, value any) {
	c.request.raw = c.request.raw.WithContext(context.WithValue(c.request.raw.Context(), key, value))
}

func (c *Context) GetValue(key any) any {
	return c.request.raw.Context().Value(key)
}
