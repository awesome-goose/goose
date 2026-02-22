package web

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

func (c *Context) SetValue(key any, value any) {
	c.request.raw = c.request.raw.WithContext(context.WithValue(c.request.raw.Context(), key, value))
}

func (c *Context) GetValue(key any) any {
	return c.request.raw.Context().Value(key)
}
