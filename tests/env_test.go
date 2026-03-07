package tests

import (
	"os"
	"testing"

	"github.com/awesome-goose/goose/env"
	test "github.com/awesome-goose/goose/testing"
)

func TestEnv(t *testing.T) {
	test.NewSuiteRunner(t, &EnvSuite{}).Run()
}

type EnvSuite struct {
	test.Suite
	env *env.Env
}

func (s *EnvSuite) SetupTest() {
	// NewEnv loads from OS, but that's fine for testing
	// We'll override values in each test
	s.env = env.NewEnv()
}

func (s *EnvSuite) TestSetAndGet() {
	s.env.Set("TEST_KEY", "test_value")
	s.T.Expect(s.env.Get("TEST_KEY")).ToEqual("test_value")
}

func (s *EnvSuite) TestGetNonExistent() {
	result := s.env.Get("NON_EXISTENT_KEY")
	s.T.Expect(result).ToEqual("")
}

func (s *EnvSuite) TestGetWithDefault_ValueExists() {
	s.env.Set("EXISTING_KEY", "existing_value")
	result := s.env.GetWithDefault("EXISTING_KEY", "default_value")
	s.T.Expect(result).ToEqual("existing_value")
}

func (s *EnvSuite) TestGetWithDefault_ValueNotExists() {
	result := s.env.GetWithDefault("NON_EXISTENT_KEY", "default_value")
	s.T.Expect(result).ToEqual("default_value")
}

func (s *EnvSuite) TestGetInt() {
	s.env.Set("INT_KEY", "42")
	result := s.env.GetInt("INT_KEY")
	s.T.Expect(result).ToEqual(42)
}

func (s *EnvSuite) TestGetInt_EmptyValue() {
	result := s.env.GetInt("EMPTY_KEY")
	s.T.Expect(result).ToEqual(0)
}

func (s *EnvSuite) TestGetInt_InvalidValue() {
	s.env.Set("INVALID_INT", "not_a_number")
	result := s.env.GetInt("INVALID_INT")
	s.T.Expect(result).ToEqual(0)
}

func (s *EnvSuite) TestGetInt_NegativeValue() {
	s.env.Set("NEGATIVE_INT", "-10")
	result := s.env.GetInt("NEGATIVE_INT")
	s.T.Expect(result).ToEqual(-10)
}

func (s *EnvSuite) TestGetBool_True() {
	s.env.Set("BOOL_KEY", "true")
	s.T.Expect(s.env.GetBool("BOOL_KEY")).ToBeTrue()
}

func (s *EnvSuite) TestGetBool_False() {
	s.env.Set("BOOL_KEY", "false")
	s.T.Expect(s.env.GetBool("BOOL_KEY")).ToBeFalse()
}

func (s *EnvSuite) TestGetBool_EmptyValue() {
	result := s.env.GetBool("EMPTY_KEY")
	s.T.Expect(result).ToBeFalse()
}

func (s *EnvSuite) TestGetBool_InvalidValue() {
	s.env.Set("INVALID_BOOL", "not_a_bool")
	result := s.env.GetBool("INVALID_BOOL")
	s.T.Expect(result).ToBeFalse()
}

func (s *EnvSuite) TestGetFloat() {
	s.env.Set("FLOAT_KEY", "3.14159")
	result := s.env.GetFloat("FLOAT_KEY")
	s.T.Expect(result).ToEqual(3.14159)
}

func (s *EnvSuite) TestGetFloat_EmptyValue() {
	result := s.env.GetFloat("EMPTY_KEY")
	s.T.Expect(result).ToEqual(0.0)
}

func (s *EnvSuite) TestGetFloat_InvalidValue() {
	s.env.Set("INVALID_FLOAT", "not_a_float")
	result := s.env.GetFloat("INVALID_FLOAT")
	s.T.Expect(result).ToEqual(0.0)
}

func (s *EnvSuite) TestGetFloat_NegativeValue() {
	s.env.Set("NEGATIVE_FLOAT", "-2.5")
	result := s.env.GetFloat("NEGATIVE_FLOAT")
	s.T.Expect(result).ToEqual(-2.5)
}

func (s *EnvSuite) TestOverwriteValue() {
	s.env.Set("OVERWRITE_KEY", "first_value")
	s.T.Expect(s.env.Get("OVERWRITE_KEY")).ToEqual("first_value")

	s.env.Set("OVERWRITE_KEY", "second_value")
	s.T.Expect(s.env.Get("OVERWRITE_KEY")).ToEqual("second_value")
}

func (s *EnvSuite) TestSpecialCharacters() {
	s.env.Set("SPECIAL_KEY", "value with spaces & symbols!")
	result := s.env.Get("SPECIAL_KEY")
	s.T.Expect(result).ToEqual("value with spaces & symbols!")
}

func (s *EnvSuite) TestNewEnvLoadsOsVars() {
	os.Setenv("GOOSE_TEST_ENV_VAR", "test_value_from_os")
	defer os.Unsetenv("GOOSE_TEST_ENV_VAR")

	newEnv := env.NewEnv()
	result := newEnv.Get("GOOSE_TEST_ENV_VAR")
	s.T.Expect(result).ToEqual("test_value_from_os")
}

func (s *EnvSuite) TestUnicodeValues() {
	s.env.Set("UNICODE_KEY", "你好世界")
	result := s.env.Get("UNICODE_KEY")
	s.T.Expect(result).ToEqual("你好世界")
}
