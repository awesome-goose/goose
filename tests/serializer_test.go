package tests

import (
	"errors"
	"testing"

	"github.com/awesome-goose/goose/core"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

func TestSerializer(t *testing.T) {
	test.NewSuiteRunner(t, &SerializerSuite{}).Run()
}

type SerializerSuite struct {
	test.Suite
	serializer types.Serializer
}

func (s *SerializerSuite) SetupTest() {
	s.serializer = core.NewSerializer()
}

func (s *SerializerSuite) TestSerialize_Nil() {
	typ, data, err := s.serializer.Serialize(nil)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeNil)
	s.T.Expect(data).ToBeNil()
}

func (s *SerializerSuite) TestSerialize_NilPointer() {
	var ptr *string = nil
	typ, data, err := s.serializer.Serialize(ptr)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeNil)
	s.T.Expect(data).ToBeNil()
}

func (s *SerializerSuite) TestSerialize_String() {
	typ, data, err := s.serializer.Serialize("hello world")

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeString)
	s.T.Expect(string(data)).ToEqual("hello world")
}

func (s *SerializerSuite) TestSerialize_EmptyString() {
	typ, data, err := s.serializer.Serialize("")

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeString)
	s.T.Expect(string(data)).ToEqual("")
}

func (s *SerializerSuite) TestSerialize_BoolTrue() {
	typ, data, err := s.serializer.Serialize(true)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeBool)
	s.T.Expect(string(data)).ToEqual("true")
}

func (s *SerializerSuite) TestSerialize_BoolFalse() {
	typ, data, err := s.serializer.Serialize(false)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeBool)
	s.T.Expect(string(data)).ToEqual("false")
}

func (s *SerializerSuite) TestSerialize_Int() {
	typ, data, err := s.serializer.Serialize(42)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeNumber)
	s.T.Expect(string(data)).ToEqual("42")
}

func (s *SerializerSuite) TestSerialize_NegativeInt() {
	typ, data, err := s.serializer.Serialize(-100)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeNumber)
	s.T.Expect(string(data)).ToEqual("-100")
}

func (s *SerializerSuite) TestSerialize_Int64() {
	typ, data, err := s.serializer.Serialize(int64(9223372036854775807))

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeNumber)
	s.T.Expect(string(data)).ToEqual("9223372036854775807")
}

func (s *SerializerSuite) TestSerialize_Uint() {
	typ, data, err := s.serializer.Serialize(uint(42))

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeNumber)
	s.T.Expect(string(data)).ToEqual("42")
}

func (s *SerializerSuite) TestSerialize_Float64() {
	typ, data, err := s.serializer.Serialize(3.14159265358979)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeNumber)
	s.T.Expect(string(data)).ToContainString("3.14159")
}

func (s *SerializerSuite) TestSerialize_NegativeFloat() {
	typ, data, err := s.serializer.Serialize(-2.5)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeNumber)
	s.T.Expect(string(data)).ToEqual("-2.5")
}

func (s *SerializerSuite) TestSerialize_ByteSlice() {
	input := []byte{0x01, 0x02, 0x03, 0xAB, 0xCD}
	typ, data, err := s.serializer.Serialize(input)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeBinary)
	s.T.Expect(data).ToEqual(input)
}

func (s *SerializerSuite) TestSerialize_EmptyByteSlice() {
	input := []byte{}
	typ, data, err := s.serializer.Serialize(input)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeBinary)
	s.T.Expect(data).ToHaveLength(0)
}

type TestStructForSerialization struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func (s *SerializerSuite) TestSerialize_Struct() {
	input := TestStructForSerialization{Name: "test", Value: 42}
	typ, data, err := s.serializer.Serialize(input)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeObject)
	s.T.Expect(string(data)).ToContainString("name")
	s.T.Expect(string(data)).ToContainString("test")
}

func (s *SerializerSuite) TestSerialize_Map() {
	input := map[string]int{"a": 1, "b": 2}
	typ, data, err := s.serializer.Serialize(input)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeObject)
	s.T.Expect(string(data)).ToContainString("a")
	s.T.Expect(string(data)).ToContainString("b")
}

func (s *SerializerSuite) TestSerialize_EmptyMap() {
	input := map[string]int{}
	typ, data, err := s.serializer.Serialize(input)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeObject)
	s.T.Expect(string(data)).ToEqual("{}")
}

func (s *SerializerSuite) TestSerialize_IntSlice() {
	input := []int{1, 2, 3}
	typ, data, err := s.serializer.Serialize(input)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeObject)
	s.T.Expect(string(data)).ToEqual("[1,2,3]")
}

func (s *SerializerSuite) TestSerialize_EmptySlice() {
	input := []int{}
	typ, data, err := s.serializer.Serialize(input)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeObject)
	s.T.Expect(string(data)).ToEqual("[]")
}

func (s *SerializerSuite) TestSerialize_Array() {
	input := [3]int{1, 2, 3}
	typ, data, err := s.serializer.Serialize(input)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeObject)
	s.T.Expect(string(data)).ToEqual("[1,2,3]")
}

func (s *SerializerSuite) TestSerialize_Error() {
	input := errors.New("test error message")
	typ, data, err := s.serializer.Serialize(input)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeError)
	s.T.Expect(string(data)).ToEqual("test error message")
}

func (s *SerializerSuite) TestSerialize_PointerToString() {
	str := "pointed"
	typ, data, err := s.serializer.Serialize(&str)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeString)
	s.T.Expect(string(data)).ToEqual("pointed")
}

func (s *SerializerSuite) TestSerialize_PointerToInt() {
	num := 42
	typ, data, err := s.serializer.Serialize(&num)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeNumber)
	s.T.Expect(string(data)).ToEqual("42")
}

func (s *SerializerSuite) TestSerialize_PointerToStruct() {
	input := &TestStructForSerialization{Name: "ptr", Value: 99}
	typ, data, err := s.serializer.Serialize(input)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(typ).ToEqual(types.SerialTypeObject)
	s.T.Expect(string(data)).ToContainString("ptr")
}
