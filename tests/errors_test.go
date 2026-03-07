package tests

import (
	"errors"
	"testing"

	gooseErr "github.com/awesome-goose/goose/errors"
	test "github.com/awesome-goose/goose/testing"
)

func TestErrors(t *testing.T) {
	test.NewSuiteRunner(t, &ErrorsSuite{}).Run()
}

type ErrorsSuite struct {
	test.Suite
}

func (s *ErrorsSuite) TestNew() {
	err := gooseErr.New("TEST_CODE", "Test message", "Detail info", "Suggested action")

	s.T.Expect(err.Code).ToEqual("TEST_CODE")
	s.T.Expect(err.Message).ToEqual("Test message")
	s.T.Expect(err.Detail).ToEqual("Detail info")
	s.T.Expect(err.SuggestedAction).ToEqual("Suggested action")
	s.T.Expect(err.Err).ToBeNil()
	s.T.Expect(err.Meta).ToBeNil()
}

func (s *ErrorsSuite) TestNewWithMeta() {
	meta := map[string]string{"key": "value"}
	err := gooseErr.NewWithMeta("META_CODE", "Message", "Detail", "Action", meta)

	s.T.Expect(err.Code).ToEqual("META_CODE")
	s.T.Expect(err.Meta).ToEqual(meta)
}

func (s *ErrorsSuite) TestWrap() {
	inner := errors.New("inner error")
	err := gooseErr.Wrap(inner, "WRAP_CODE", "Wrapped message", "Detail", "Action")

	s.T.Expect(err.Code).ToEqual("WRAP_CODE")
	s.T.Expect(err.Err).ToEqual(inner)
	s.T.Expect(err.Unwrap()).ToEqual(inner)
}

func (s *ErrorsSuite) TestErrorString() {
	// Test with detail
	err1 := gooseErr.New("CODE1", "Message", "Detail", "Action")
	s.T.Expect(err1.Error()).ToContainString("CODE1")
	s.T.Expect(err1.Error()).ToContainString("Message")
	s.T.Expect(err1.Error()).ToContainString("Detail")

	// Test without detail
	err2 := gooseErr.New("CODE2", "Message", "", "Action")
	s.T.Expect(err2.Error()).ToContainString("CODE2")
	s.T.Expect(err2.Error()).ToContainString("Message")

	// Test with wrapped error
	inner := errors.New("inner")
	err3 := gooseErr.Wrap(inner, "CODE3", "Message", "Detail", "Action")
	s.T.Expect(err3.Error()).ToContainString("inner")

	// Test nil error
	var nilErr *gooseErr.Error
	s.T.Expect(nilErr.Error()).ToEqual("<nil>")
}

func (s *ErrorsSuite) TestIs() {
	err1 := gooseErr.New("SAME_CODE", "Message 1", "", "")
	err2 := gooseErr.New("SAME_CODE", "Message 2", "", "")
	err3 := gooseErr.New("DIFF_CODE", "Message 3", "", "")

	s.T.Expect(err1.Is(err2)).ToBeTrue()
	s.T.Expect(err1.Is(err3)).ToBeFalse()
	s.T.Expect(err1.Is(errors.New("not goose error"))).ToBeFalse()
}

func (s *ErrorsSuite) TestWithMeta() {
	err := gooseErr.New("CODE", "Message", "", "")
	result := err.WithMeta("test meta")

	s.T.Expect(result.Meta).ToEqual("test meta")
	s.T.Expect(result).ToEqual(err) // Should return same pointer
}

func (s *ErrorsSuite) TestWithError() {
	inner := errors.New("inner error")
	err := gooseErr.New("CODE", "Message", "", "")
	result := err.WithError(inner)

	s.T.Expect(result.Err).ToEqual(inner)
	s.T.Expect(result).ToEqual(err) // Should return same pointer
}

func (s *ErrorsSuite) TestWithDetail() {
	err := gooseErr.New("CODE", "Message", "", "")
	result := err.WithDetail("new detail")

	s.T.Expect(result.Detail).ToEqual("new detail")
}

func (s *ErrorsSuite) TestWithSuggestedAction() {
	err := gooseErr.New("CODE", "Message", "", "")
	result := err.WithSuggestedAction("new action")

	s.T.Expect(result.SuggestedAction).ToEqual("new action")
}

func (s *ErrorsSuite) TestGetters() {
	err := gooseErr.NewWithMeta("CODE", "Message", "Detail", "Action", "meta")

	s.T.Expect(err.GetCode()).ToEqual("CODE")
	s.T.Expect(err.GetMessage()).ToEqual("Message")
	s.T.Expect(err.GetDetail()).ToEqual("Detail")
	s.T.Expect(err.GetSuggestedAction()).ToEqual("Action")
	s.T.Expect(err.GetMeta()).ToEqual("meta")
}

func (s *ErrorsSuite) TestGettersOnNil() {
	var nilErr *gooseErr.Error

	s.T.Expect(nilErr.GetCode()).ToEqual("")
	s.T.Expect(nilErr.GetMessage()).ToEqual("")
	s.T.Expect(nilErr.GetDetail()).ToEqual("")
	s.T.Expect(nilErr.GetSuggestedAction()).ToEqual("")
	s.T.Expect(nilErr.GetMeta()).ToBeNil()
}

func (s *ErrorsSuite) TestString() {
	err := gooseErr.New("CODE", "Message", "", "")
	s.T.Expect(err.String()).ToEqual(err.Error())
}

func (s *ErrorsSuite) TestUnwrapOnNil() {
	var nilErr *gooseErr.Error
	s.T.Expect(nilErr.Unwrap()).ToBeNil()
}

func (s *ErrorsSuite) TestBuiltinErrors() {
	s.T.Expect(gooseErr.ErrConfigFileNotFound).Not().ToBeNil()
	s.T.Expect(gooseErr.ErrFailedToReadConfigFile).Not().ToBeNil()
	s.T.Expect(gooseErr.ErrNamespaceRequired).Not().ToBeNil()
	s.T.Expect(gooseErr.ErrNoConcreteFound).Not().ToBeNil()
	s.T.Expect(gooseErr.ErrCircularDependency).Not().ToBeNil()
	s.T.Expect(gooseErr.ErrResolverMustBeFunction).Not().ToBeNil()
}

func (s *ErrorsSuite) TestMethodChaining() {
	err := gooseErr.New("CODE", "Message", "", "").
		WithMeta("meta").
		WithDetail("detail").
		WithSuggestedAction("action")

	s.T.Expect(err.Meta).ToEqual("meta")
	s.T.Expect(err.Detail).ToEqual("detail")
	s.T.Expect(err.SuggestedAction).ToEqual("action")
}
