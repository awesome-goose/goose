package types

type Request interface {
	Headers() map[string][]string
	Method() Method
	Paths() []string
	Queries() map[string]string
	Params() map[string]string
	Body() ([]byte, error)

	PopulateParams(params map[string]string)
}
