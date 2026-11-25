// Package clash has a type named like one in the openapi tests, for the name-clash test.
package clash

// Thing has the same name as openapi_test.Thing.
type Thing struct {
	Other string `json:"other"`
}

// Named is a Thing-like type that chooses its own component name.
type Named struct {
	Other string `json:"other"`
}

// OpenAPIName names its component.
func (Named) OpenAPIName() string { return "ClashThing" }
