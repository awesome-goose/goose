package types

type Method string

var (
	GET    Method = "GET"
	POST   Method = "POST"
	PUT    Method = "PUT"
	DELETE Method = "DELETE"
	PATCH  Method = "PATCH"
)

func (m Method) String() string {
	return string(m)
}

func (m Method) Is(value string) bool {
	return string(m) == value
}
