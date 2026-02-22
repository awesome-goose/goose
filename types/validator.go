package types

type Validator interface {
	Validate(data any) error
	ValidateContext(ctx Context) error
}
type Validators []Validator
