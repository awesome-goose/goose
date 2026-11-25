// Package clash is a second package named clash, so qualifying by the last element is not enough.
package clash

// Thing has the same name and the same package name as internal/clash.Thing.
type Thing struct {
	Again int `json:"again"`
}
