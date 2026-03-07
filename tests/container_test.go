package tests

import (
	"testing"

	"github.com/awesome-goose/goose/core"
	gooseErr "github.com/awesome-goose/goose/errors"
	test "github.com/awesome-goose/goose/testing"
)

func TestContainer(t *testing.T) {
	test.NewSuiteRunner(t, &ContainerSuite{}).Run()
}

type ContainerSuite struct {
	test.Suite
	container *core.Container
}

func (s *ContainerSuite) SetupTest() {
	s.container = core.NewContainer()
}

func (s *ContainerSuite) TeardownTest() {
	if s.container != nil {
		s.container.Reset()
	}
}

type TestService struct {
	Name string
}

type TestServiceDep struct {
	Service *TestService `inject:""`
}

type TestInterface interface {
	Do() string
}

type TestImpl struct{}

func (t *TestImpl) Do() string { return "done" }

func (s *ContainerSuite) TestRegister_Singleton() {
	callCount := 0
	err := s.container.Register(func() *TestService {
		callCount++
		return &TestService{Name: "singleton"}
	}, "", true)

	s.T.Expect(err).ToBeNil()

	var svc1, svc2 *TestService
	s.container.Resolve(&svc1, "")
	s.container.Resolve(&svc2, "")

	s.T.Expect(svc1).ToEqual(svc2)
	s.T.Expect(callCount).ToEqual(1)
}

func (s *ContainerSuite) TestRegister_Transient() {
	callCount := 0
	err := s.container.Register(func() *TestService {
		callCount++
		return &TestService{Name: "transient"}
	}, "", false)

	s.T.Expect(err).ToBeNil()

	var svc1, svc2 *TestService
	s.container.Resolve(&svc1, "")
	s.container.Resolve(&svc2, "")

	s.T.Expect(callCount).ToEqual(2)
}

func (s *ContainerSuite) TestRegister_NilResolver() {
	err := s.container.Register(nil, "", true)
	s.T.Expect(err).Not().ToBeNil()
}

func (s *ContainerSuite) TestRegister_InvalidResolver() {
	err := s.container.Register("not a function", "", true)
	s.T.Expect(err).Not().ToBeNil()
	s.T.Expect(gooseErr.ErrInvalidResolver.Is(err)).ToBeTrue()
}

func (s *ContainerSuite) TestResolve_Success() {
	s.container.Register(func() *TestService {
		return &TestService{Name: "test"}
	}, "", true)

	var service *TestService
	err := s.container.Resolve(&service, "")

	s.T.Expect(err).ToBeNil()
	s.T.Expect(service).Not().ToBeNil()
	s.T.Expect(service.Name).ToEqual("test")
}

func (s *ContainerSuite) TestResolve_NotFound() {
	var service *TestService
	err := s.container.Resolve(&service, "")

	s.T.Expect(err).Not().ToBeNil()
}

func (s *ContainerSuite) TestResolve_NilAbstraction() {
	err := s.container.Resolve(nil, "")
	s.T.Expect(err).Not().ToBeNil()
}

func (s *ContainerSuite) TestNamedBinding() {
	s.container.Register(func() *TestService {
		return &TestService{Name: "service-a"}
	}, "a", true)

	s.container.Register(func() *TestService {
		return &TestService{Name: "service-b"}
	}, "b", true)

	var svcA, svcB *TestService
	s.container.Resolve(&svcA, "a")
	s.container.Resolve(&svcB, "b")

	s.T.Expect(svcA.Name).ToEqual("service-a")
	s.T.Expect(svcB.Name).ToEqual("service-b")
}

func (s *ContainerSuite) TestCreate_Struct() {
	instance, err := s.container.Create(TestService{Name: "created"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(instance).Not().ToBeNil()

	svc, ok := instance.(*TestService)
	s.T.Expect(ok).ToBeTrue()
	s.T.Expect(svc).Not().ToBeNil()
}

func (s *ContainerSuite) TestCreate_PointerToStruct() {
	input := &TestService{Name: "ptr-created"}
	instance, err := s.container.Create(input)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(instance).Not().ToBeNil()
}

func (s *ContainerSuite) TestCreate_Nil() {
	instance, err := s.container.Create(nil)

	s.T.Expect(err).Not().ToBeNil()
	s.T.Expect(instance).ToBeNil()
}

func (s *ContainerSuite) TestCreate_InvalidType() {
	instance, err := s.container.Create("not a struct")

	s.T.Expect(err).Not().ToBeNil()
	s.T.Expect(instance).ToBeNil()
}

func (s *ContainerSuite) TestCreate_CachesInstance() {
	instance1, _ := s.container.Create(TestService{})
	instance2, _ := s.container.Create(TestService{})

	s.T.Expect(instance1).ToEqual(instance2)
}

func (s *ContainerSuite) TestCreate_WithDependencies() {
	s.container.Register(func() *TestService {
		return &TestService{Name: "injected"}
	}, "", true)

	instance, err := s.container.Create(TestServiceDep{})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(instance).Not().ToBeNil()

	dep, ok := instance.(*TestServiceDep)
	s.T.Expect(ok).ToBeTrue()
	s.T.Expect(dep.Service).Not().ToBeNil()
	s.T.Expect(dep.Service.Name).ToEqual("injected")
}

func (s *ContainerSuite) TestFill_Success() {
	s.container.Register(func() *TestService {
		return &TestService{Name: "filled"}
	}, "", true)

	target := &TestServiceDep{}
	err := s.container.Fill(target)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(target.Service).Not().ToBeNil()
	s.T.Expect(target.Service.Name).ToEqual("filled")
}

func (s *ContainerSuite) TestFill_NilTarget() {
	err := s.container.Fill(nil)
	s.T.Expect(err).Not().ToBeNil()
}

func (s *ContainerSuite) TestFill_NotPointer() {
	err := s.container.Fill(TestServiceDep{})
	s.T.Expect(err).Not().ToBeNil()
}

func (s *ContainerSuite) TestCall_Success() {
	s.container.Register(func() *TestService {
		return &TestService{Name: "called"}
	}, "", true)

	var receivedService *TestService
	err := s.container.Call(func(svc *TestService) {
		receivedService = svc
	})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(receivedService).Not().ToBeNil()
	s.T.Expect(receivedService.Name).ToEqual("called")
}

func (s *ContainerSuite) TestCall_InvalidFunction() {
	err := s.container.Call("not a function")
	s.T.Expect(err).Not().ToBeNil()
}

func (s *ContainerSuite) TestCall_NilFunction() {
	err := s.container.Call(nil)
	s.T.Expect(err).Not().ToBeNil()
}

func (s *ContainerSuite) TestCall_MissingDependency() {
	err := s.container.Call(func(svc *TestService) {})
	s.T.Expect(err).Not().ToBeNil()
}

func (s *ContainerSuite) TestReset() {
	s.container.Register(func() *TestService {
		return &TestService{Name: "test"}
	}, "", true)

	s.container.Reset()

	var service *TestService
	err := s.container.Resolve(&service, "")
	s.T.Expect(err).Not().ToBeNil()
}

func (s *ContainerSuite) TestInterfaceBinding() {
	s.container.Register(func() TestInterface {
		return &TestImpl{}
	}, "", true)

	var impl TestInterface
	err := s.container.Resolve(&impl, "")

	s.T.Expect(err).ToBeNil()
	s.T.Expect(impl).Not().ToBeNil()
	s.T.Expect(impl.Do()).ToEqual("done")
}

func (s *ContainerSuite) TestRegister_ResolverWithError_Success() {
	err := s.container.Register(func() (*TestService, error) {
		return &TestService{Name: "with-error"}, nil
	}, "", true)

	s.T.Expect(err).ToBeNil()

	var svc *TestService
	err = s.container.Resolve(&svc, "")
	s.T.Expect(err).ToBeNil()
	s.T.Expect(svc.Name).ToEqual("with-error")
}

func (s *ContainerSuite) TestRegister_ResolverWithError_Fails() {
	err := s.container.Register(func() (*TestService, error) {
		return nil, gooseErr.New("TEST_ERROR", "test error", "", "")
	}, "", true)

	s.T.Expect(err).Not().ToBeNil()
}

type Level1 struct {
	Name string
}

type Level2 struct {
	L1 *Level1 `inject:""`
}

type Level3 struct {
	L2 *Level2 `inject:""`
}

func (s *ContainerSuite) TestCreate_MultiLevelDependencies() {
	// Register Level1 - standalone
	s.container.Register(func() *Level1 {
		return &Level1{Name: "level1"}
	}, "", true)

	// Register Level2 with its Level1 already filled by factory
	s.container.Register(func() *Level2 {
		return &Level2{L1: &Level1{Name: "level1-nested"}}
	}, "", true)

	// Create Level3 - container should inject L2
	instance, err := s.container.Create(Level3{})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(instance).Not().ToBeNil()

	l3, ok := instance.(*Level3)
	s.T.Expect(ok).ToBeTrue()
	s.T.Expect(l3.L2).Not().ToBeNil()
	s.T.Expect(l3.L2.L1).Not().ToBeNil()
	s.T.Expect(l3.L2.L1.Name).ToEqual("level1-nested")
}

type CloseAwareService struct {
	closed bool
}

func (c *CloseAwareService) OnClose() error {
	c.closed = true
	return nil
}

func (s *ContainerSuite) TestClose() {
	svc := &CloseAwareService{}
	s.container.Register(func() *CloseAwareService {
		return svc
	}, "", true)

	var resolved *CloseAwareService
	s.container.Resolve(&resolved, "")

	err := s.container.Close()

	s.T.Expect(err).ToBeNil()
	s.T.Expect(svc.closed).ToBeTrue()

	var afterClose *CloseAwareService
	err = s.container.Resolve(&afterClose, "")
	s.T.Expect(err).Not().ToBeNil()
}
