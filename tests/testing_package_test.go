package tests

import (
	"testing"

	test "github.com/awesome-goose/goose/testing"
)

func TestTestingPackage(t *testing.T) {
	test.NewSuiteRunner(t, &TestingSuite{}).Run()
}

type TestingSuite struct {
	test.Suite
}

// Test GooseTest basic functionality
func (s *TestingSuite) TestGooseTest_New() {
	gt := test.New(s.T.T())
	s.T.Expect(gt).Not().ToBeNil()
}

// Test assertions - ToEqual
func (s *TestingSuite) TestExpect_ToEqual_String() {
	gt := test.New(s.T.T())
	gt.Expect("hello").ToEqual("hello")
}

func (s *TestingSuite) TestExpect_ToEqual_Int() {
	gt := test.New(s.T.T())
	gt.Expect(42).ToEqual(42)
}

func (s *TestingSuite) TestExpect_ToEqual_Float() {
	gt := test.New(s.T.T())
	gt.Expect(3.14).ToEqual(3.14)
}

// Test assertions - Not().ToEqual()
func (s *TestingSuite) TestExpect_NotToEqual() {
	gt := test.New(s.T.T())
	gt.Expect("hello").Not().ToEqual("world")
	gt.Expect(1).Not().ToEqual(2)
}

// Test assertions - ToBeNil / Not().ToBeNil()
func (s *TestingSuite) TestExpect_ToBeNil() {
	gt := test.New(s.T.T())
	var nilPtr *string
	gt.Expect(nilPtr).ToBeNil()
	gt.Expect(nil).ToBeNil()
}

func (s *TestingSuite) TestExpect_NotToBeNil() {
	gt := test.New(s.T.T())
	str := "hello"
	gt.Expect(&str).Not().ToBeNil()
	gt.Expect("value").Not().ToBeNil()
}

// Test assertions - ToBeTrue / ToBeFalse
func (s *TestingSuite) TestExpect_ToBeTrue() {
	gt := test.New(s.T.T())
	gt.Expect(true).ToBeTrue()
	gt.Expect(1 == 1).ToBeTrue()
}

func (s *TestingSuite) TestExpect_ToBeFalse() {
	gt := test.New(s.T.T())
	gt.Expect(false).ToBeFalse()
	gt.Expect(1 == 2).ToBeFalse()
}

// Test assertions - ToHaveLength
func (s *TestingSuite) TestExpect_ToHaveLength_Slice() {
	gt := test.New(s.T.T())
	slice := []int{1, 2, 3}
	gt.Expect(slice).ToHaveLength(3)
}

func (s *TestingSuite) TestExpect_ToHaveLength_Map() {
	gt := test.New(s.T.T())
	m := map[string]int{"a": 1, "b": 2}
	gt.Expect(m).ToHaveLength(2)
}

func (s *TestingSuite) TestExpect_ToHaveLength_String() {
	gt := test.New(s.T.T())
	gt.Expect("hello").ToHaveLength(5)
}

// Test assertions - ToContain
func (s *TestingSuite) TestExpect_ToContain_String() {
	gt := test.New(s.T.T())
	gt.Expect("hello world").ToContain("world")
}

func (s *TestingSuite) TestExpect_ToContain_Slice() {
	gt := test.New(s.T.T())
	slice := []string{"a", "b", "c"}
	gt.Expect(slice).ToContain("b")
}

// Test mock context
func (s *TestingSuite) TestMockContext_Basic() {
	ctx := test.NewMockContext()
	s.T.Expect(ctx).Not().ToBeNil()
}

func (s *TestingSuite) TestMockContext_GetSet() {
	ctx := test.NewMockContext()
	ctx.SetValue("key", "value")
	result := ctx.GetValue("key")
	s.T.Expect(result).ToEqual("value")
}

func (s *TestingSuite) TestMockContext_GetDefault() {
	ctx := test.NewMockContext()
	result := ctx.GetValue("nonexistent")
	s.T.Expect(result).ToBeNil()
}

// Test fixtures
func (s *TestingSuite) TestFixtures_RegisterAndGet() {
	fixture := test.NewFixture()
	s.T.Expect(fixture).Not().ToBeNil()
}

// Test mock implementation
type MockTestService struct {
	mock *test.Mock
}

func NewMockTestService(t *testing.T) *MockTestService {
	return &MockTestService{
		mock: test.NewMock(t),
	}
}

func (m *MockTestService) DoSomething(input string) string {
	args := m.mock.Called("DoSomething", input)
	if args == nil || len(args) == 0 {
		return ""
	}
	if result, ok := args[0].(string); ok {
		return result
	}
	return ""
}

func (s *TestingSuite) TestMock_CalledAndReturns() {
	mock := NewMockTestService(s.T.T())
	mock.mock.On("DoSomething", "output")

	result := mock.DoSomething("input")

	s.T.Expect(result).ToEqual("output")
	s.T.Expect(mock.mock.WasCalled("DoSomething")).ToBeTrue()
}

func (s *TestingSuite) TestMock_WasCalledWith() {
	mock := NewMockTestService(s.T.T())
	mock.mock.On("DoSomething", "output")

	mock.DoSomething("specific-input")

	s.T.Expect(mock.mock.WasCalledWith("DoSomething", "specific-input")).ToBeTrue()
	s.T.Expect(mock.mock.WasCalledWith("DoSomething", "other-input")).ToBeFalse()
}

func (s *TestingSuite) TestMock_WasNotCalled() {
	mock := NewMockTestService(s.T.T())
	mock.mock.On("DoSomething", "output")

	s.T.Expect(mock.mock.WasCalled("DoSomething")).ToBeFalse()
}

func (s *TestingSuite) TestMock_CallCount() {
	mock := NewMockTestService(s.T.T())
	mock.mock.On("DoSomething", "output")

	mock.DoSomething("first")
	mock.DoSomething("second")
	mock.DoSomething("third")

	s.T.Expect(mock.mock.CallCount("DoSomething")).ToEqual(3)
}

// Test suite lifecycle
type LifecycleSuite struct {
	test.Suite
	setupCalled    bool
	teardownCalled bool
}

func (s *LifecycleSuite) SetupTest() {
	s.setupCalled = true
}

func (s *LifecycleSuite) TeardownTest() {
	s.teardownCalled = true
}

func (s *LifecycleSuite) TestSetupWasCalled() {
	s.T.Expect(s.setupCalled).ToBeTrue()
}

func (s *TestingSuite) TestSuiteLifecycle() {
	// Run a nested suite to verify lifecycle
	suite := &LifecycleSuite{}
	test.NewSuiteRunner(s.T.T(), suite).Run()
	s.T.Expect(suite.teardownCalled).ToBeTrue()
}

// Test container helpers
func (s *TestingSuite) TestTestContainer_Basic() {
	container := test.NewTestContainer()
	s.T.Expect(container).Not().ToBeNil()
}

type TestDependency struct {
	Value string
}

type TestTarget struct {
	Dep *TestDependency
}

func (s *TestingSuite) TestTestContainer_RegisterAndResolve() {
	container := test.NewTestContainer()
	dep := &TestDependency{Value: "test"}
	container.Register(dep)

	target := &TestTarget{}
	container.Resolve(target)

	s.T.Expect(target.Dep).Not().ToBeNil()
	s.T.Expect(target.Dep.Value).ToEqual("test")
}

// Test mock request/response
func (s *TestingSuite) TestMockRequest_Basic() {
	req := test.NewMockRequest()
	s.T.Expect(req).Not().ToBeNil()
}

func (s *TestingSuite) TestMockResponse_Basic() {
	res := test.NewMockResponse()
	s.T.Expect(res).Not().ToBeNil()
}
