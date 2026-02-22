package types

type Serializer interface {
	Serialize(data any) (SerialType, []byte, error)
}

type SerialType string

const (
	SerialTypeString   SerialType = "string"
	SerialTypeBool     SerialType = "bool"
	SerialTypeNumber   SerialType = "number"
	SerialTypeObject   SerialType = "object"
	SerialTypeBinary   SerialType = "binary"
	SerialTypeNil      SerialType = "nil"
	SerialTypeError    SerialType = "error"
	SerialTypeHTML     SerialType = "html"
	SerialTypeFile     SerialType = "file"
	SerialTypeRedirect SerialType = "redirect"
)

func (st SerialType) String() string {
	return string(st)
}

func (st SerialType) Is(value string) bool {
	return string(st) == value
}
