package tests

import (
	"os"
	"testing"

	"github.com/awesome-goose/goose"
	"github.com/awesome-goose/goose/core"
	"github.com/awesome-goose/goose/io/output"
	"github.com/awesome-goose/goose/platforms/api"
	"github.com/awesome-goose/goose/platforms/cli"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

type dispatchInstallParams struct{}

type dispatchModule struct {
	seen *[]string
}

func (m *dispatchModule) Imports() []types.Module { return nil }
func (m *dispatchModule) Exports() []any           { return nil }
func (m *dispatchModule) Declarations() []any      { return nil }

func (m *dispatchModule) Boot(k types.Kernel) error {
	_, err := k.AppendRoutes(types.Route{
		Method: types.GET,
		Path:   "install",
		Handler: func(p *dispatchInstallParams) types.Output {
			*m.seen = append(*m.seen, "install")
			return output.Console("installed")
		},
	})
	return err
}

func TestMultiInstanceCLIDispatch(t *testing.T) {
	test.NewSuiteRunner(t, &MultiInstanceCLIDispatchSuite{}).Run()
}

type MultiInstanceCLIDispatchSuite struct {
	test.Suite
	origArgs []string
}

func (s *MultiInstanceCLIDispatchSuite) SetupTest() {
	s.origArgs = os.Args
}

func (s *MultiInstanceCLIDispatchSuite) TeardownTest() {
	os.Args = s.origArgs
}

// Regression test for bug #3 in BUGS.md: runMulti decided CLI mode by
// checking os.Args[1] == "cli", but never stripped that selector before the
// CLI platform's Request parsed os.Args. A route registered as "install"
// (the same convention single-instance CLI apps use) 404'd on
// `myapp cli install` because Request.Paths() still saw ["cli", "install"]
// instead of ["install"].
func (s *MultiInstanceCLIDispatchSuite) TestCLISelectorIsStrippedBeforeRouting() {
	os.Args = []string{"myapp", "cli", "install"}

	var seen []string
	module := &dispatchModule{seen: &seen}

	// Registered alongside a CLI instance to force runMulti (the bug only
	// exists in the multi-instance path). The API instance never actually
	// boots in CLI mode, so binding a real port isn't a concern.
	apiPlatform := api.NewPlatform(api.WithName("test-api"))
	cliPlatform := cli.NewPlatform(cli.WithName("test-cli"))

	k := core.NewKernel()
	_, err := k.Start(
		goose.API(apiPlatform, module, nil),
		goose.CLI(cliPlatform, module, nil),
	)

	s.T.Expect(err).ToBeNil()
	s.T.Expect(seen).ToEqual([]string{"install"})
}
