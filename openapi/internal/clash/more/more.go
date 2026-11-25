// Package more has a third Thing, for the three-way clash test.
package more

// Thing has the same name as openapi_test.Thing and clash.Thing.
type Thing struct {
	More bool `json:"more"`
}
