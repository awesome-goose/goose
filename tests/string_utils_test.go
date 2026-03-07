package tests

import (
	"testing"

	test "github.com/awesome-goose/goose/testing"
	str "github.com/awesome-goose/goose/utils/string"
)

func TestStringUtils(t *testing.T) {
	test.NewSuiteRunner(t, &StringUtilsSuite{}).Run()
}

type StringUtilsSuite struct {
	test.Suite
}

func (s *StringUtilsSuite) TestIsValidHTTPMethod_ValidMethods() {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	for _, method := range validMethods {
		s.T.Expect(str.IsValidHTTPMethod(method)).ToBeTrue()
	}
}

func (s *StringUtilsSuite) TestIsValidHTTPMethod_LowercaseValid() {
	validMethods := []string{"get", "post", "put", "delete", "patch", "options", "head"}
	for _, method := range validMethods {
		s.T.Expect(str.IsValidHTTPMethod(method)).ToBeTrue()
	}
}

func (s *StringUtilsSuite) TestIsValidHTTPMethod_MixedCaseValid() {
	validMethods := []string{"Get", "Post", "Put", "Delete", "Patch", "Options", "Head"}
	for _, method := range validMethods {
		s.T.Expect(str.IsValidHTTPMethod(method)).ToBeTrue()
	}
}

func (s *StringUtilsSuite) TestIsValidHTTPMethod_InvalidMethods() {
	invalidMethods := []string{"INVALID", "TRACE", "CONNECT", ""}
	for _, method := range invalidMethods {
		s.T.Expect(str.IsValidHTTPMethod(method)).ToBeFalse()
	}
}

func (s *StringUtilsSuite) TestSplitPath_SimplePath() {
	result := str.SplitPath("users/123/posts")
	s.T.Expect(result).ToHaveLength(3)
	s.T.Expect(result[0]).ToEqual("users")
	s.T.Expect(result[1]).ToEqual("123")
	s.T.Expect(result[2]).ToEqual("posts")
}

func (s *StringUtilsSuite) TestSplitPath_LeadingSlash() {
	result := str.SplitPath("/users/123")
	s.T.Expect(result).ToHaveLength(2)
	s.T.Expect(result[0]).ToEqual("users")
	s.T.Expect(result[1]).ToEqual("123")
}

func (s *StringUtilsSuite) TestSplitPath_TrailingSlash() {
	result := str.SplitPath("users/123/")
	s.T.Expect(result).ToHaveLength(2)
	s.T.Expect(result[0]).ToEqual("users")
	s.T.Expect(result[1]).ToEqual("123")
}

func (s *StringUtilsSuite) TestSplitPath_BothSlashes() {
	result := str.SplitPath("/users/123/")
	s.T.Expect(result).ToHaveLength(2)
	s.T.Expect(result[0]).ToEqual("users")
	s.T.Expect(result[1]).ToEqual("123")
}

func (s *StringUtilsSuite) TestSplitPath_SingleSegment() {
	result := str.SplitPath("users")
	s.T.Expect(result).ToHaveLength(1)
	s.T.Expect(result[0]).ToEqual("users")
}

func (s *StringUtilsSuite) TestSplitPath_EmptyString() {
	result := str.SplitPath("")
	s.T.Expect(result).ToHaveLength(0)
}

func (s *StringUtilsSuite) TestSplitPath_OnlySlash() {
	result := str.SplitPath("/")
	s.T.Expect(result).ToHaveLength(0)
}

func (s *StringUtilsSuite) TestSplit_BySlash() {
	result := str.Split("a/b/c", '/')
	s.T.Expect(result).ToHaveLength(3)
	s.T.Expect(result[0]).ToEqual("a")
	s.T.Expect(result[1]).ToEqual("b")
	s.T.Expect(result[2]).ToEqual("c")
}

func (s *StringUtilsSuite) TestSplit_ByComma() {
	result := str.Split("one,two,three", ',')
	s.T.Expect(result).ToHaveLength(3)
	s.T.Expect(result[0]).ToEqual("one")
	s.T.Expect(result[1]).ToEqual("two")
	s.T.Expect(result[2]).ToEqual("three")
}

func (s *StringUtilsSuite) TestSplit_NoSeparator() {
	result := str.Split("noseparator", '/')
	s.T.Expect(result).ToHaveLength(1)
	s.T.Expect(result[0]).ToEqual("noseparator")
}

func (s *StringUtilsSuite) TestSplit_EmptyString() {
	result := str.Split("", '/')
	s.T.Expect(result).ToHaveLength(0)
}

func (s *StringUtilsSuite) TestSplit_UnicodeString() {
	result := str.Split("你好/世界", '/')
	s.T.Expect(result).ToHaveLength(2)
	s.T.Expect(result[0]).ToEqual("你好")
	s.T.Expect(result[1]).ToEqual("世界")
}
