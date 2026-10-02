// Package openapi builds an OpenAPI 3.1 document from Goose's route table and the
// tags of the handlers' input structs (TRD §11 U-G3, PLAN M2-11).
//
// A handler is `func(*Dto) types.Output` or a controller method `[]any{Controller{}, "Method"}`.
// The Dto's field tags say where each value comes from, which is exactly what the
// request reader uses (io/input), so the document cannot say something the server does not do:
//
//	param:"id"       path parameter
//	query:"name"     query parameter (a []string is comma separated, as the reader splits it)
//	queries:"all"    any other query parameters (a map[string]string)
//	header:"name"    header parameter
//	json:"name"      property of the JSON body (POST, PUT, PATCH, DELETE)
//	form:"name"      property of a form body
//	raw:"body"       the body as is (octet stream)
//	context:"..."    filled by the server, not by the caller: left out
//	binding:"required,min=1,max=9,oneof=a b"   required, bounds and enum; "-" leaves the field out
//	doc:"words"      the field's description
//
// The output type of a handler is the interface types.Output, so responses are
// described as an object any JSON can fill, unless a Dto says more with an
// Operation method:
//
//	func (Dto) OpenAPI() openapi.Operation { return openapi.Operation{Summary: "...", Response: ResponseType{}} }
//
// Everything is deterministic: the same routes give the same bytes.
package openapi
