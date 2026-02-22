package cli

import (
	"os"

	"github.com/awesome-goose/goose/types"
)

type Context struct {
	request  *Request
	response *Response

	values map[any]any
}

func NewContext() *Context {
	return &Context{
		request:  NewRequest(),
		response: NewResponse(os.Stdout),
	}
}

func (c *Context) Request() types.Request {
	return c.request
}

func (c *Context) Response() types.Response {
	return c.response
}

func (c *Context) SetValue(key any, value any) {
	if c.values == nil {
		c.values = make(map[any]any)
	}

	c.values[key] = value
}

func (c *Context) GetValue(key any) any {
	if c.values == nil {
		return nil
	}

	return c.values[key]
}
