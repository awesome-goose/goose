package types

type Context interface {
	Request() Request
	Response() Response

	SetValue(key any, value any)
	GetValue(key any) any
}
