package tests

import (
	"testing"

	"github.com/awesome-goose/goose/core"
	"github.com/awesome-goose/goose/platforms/cli"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

type logInjectedController struct {
	Log types.Log `inject:""`
}

type logInjectedModule struct{}

func (m *logInjectedModule) Imports() []types.Module { return nil }
func (m *logInjectedModule) Exports() []any           { return nil }
func (m *logInjectedModule) Declarations() []any {
	return []any{&logInjectedController{}}
}

func TestDefaultServices(t *testing.T) {
	test.NewSuiteRunner(t, &DefaultServicesSuite{}).Run()
}

type DefaultServicesSuite struct {
	test.Suite
}

// Regression test for bug #1 in BUGS.md: the default `services` list
// registered the logger slice under the named type log.AppLoggers, but the
// types.Log constructor asked for the unnamed []*log.Logger. Since those are
// different reflect.Types, the Log constructor's own container.Register call
// failed silently (its error is discarded in the services registration
// loop), so types.Log never got a binding unless the app supplied its own
// Log Initializer.
func (s *DefaultServicesSuite) TestLogResolvesWithoutCustomInitializer() {
	platform := cli.NewPlatform(cli.WithName("test-cli"))
	instance := &types.Instance{
		Name:     "test-cli",
		Type:     types.PlatformTypeCLI,
		Platform: platform,
		Module:   &logInjectedModule{},
	}

	k := core.NewKernel()

	// Start() itself may still return an error once it reaches routing (this
	// module registers no routes), but that must happen only *after* the
	// default `services` and the module tree hydrate successfully. Before
	// the fix, hydration failed first: creating logInjectedController's
	// Log field hit ErrCannotCreateInterfaceField because types.Log was
	// never bound in the container.
	_, _ = k.Start(instance)

	var resolvedLog types.Log
	err := k.Container().Resolve(&resolvedLog, "")
	s.T.Expect(err).ToBeNil()
	s.T.Expect(resolvedLog).Not().ToBeNil()
}
