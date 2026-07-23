package tests

import (
	"os"
	"os/exec"
	"testing"

	"github.com/awesome-goose/goose"
	"github.com/awesome-goose/goose/core"
	"github.com/awesome-goose/goose/io/output"
	"github.com/awesome-goose/goose/platforms/cli"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

type exitCodeParams struct{}

type exitCodeModule struct{}

func (m *exitCodeModule) Imports() []types.Module { return nil }
func (m *exitCodeModule) Exports() []any           { return nil }
func (m *exitCodeModule) Declarations() []any      { return nil }

func (m *exitCodeModule) Boot(k types.Kernel) error {
	_, err := k.AppendRoutes(types.Route{
		Method: types.GET,
		Path:   "fail",
		Handler: func(p *exitCodeParams) types.Output {
			return output.ConsoleError("boom").WithExitCode(1)
		},
	})
	return err
}

// TestHelperProcess_CLIExitCode is not a real test on its own; it's a
// subprocess helper invoked by TestCLIOutputCodeReachesProcessExitCode
// below. os.Exit can't be exercised in-process without killing the whole
// test binary, so this is the standard Go pattern for asserting on it.
func TestHelperProcess_CLIExitCode(t *testing.T) {
	if os.Getenv("GOOSE_CLI_EXIT_CODE_HELPER") != "1" {
		t.Skip("only runs as a subprocess helper")
	}

	os.Args = []string{"myapp", "fail"}

	platform := cli.NewPlatform(cli.WithName("test-cli"))
	k := core.NewKernel()
	_, _ = k.Start(goose.CLI(platform, &exitCodeModule{}, nil))

	// Only reached if Output.Code() failed to reach a real process exit.
	os.Exit(0)
}

func TestCLIOutputCodeReachesProcessExitCode(t *testing.T) {
	test.NewSuiteRunner(t, &CLIExitCodeSuite{}).Run()
}

type CLIExitCodeSuite struct {
	test.Suite
}

// Regression test for bug #4 in BUGS.md: platforms/cli's Response.Write
// accepted the handler's Output.Code() but never used it for anything, so a
// controller returning output.ConsoleError(...).WithExitCode(1) printed the
// red ERROR message correctly but the process's actual exit code stayed 0.
func (s *CLIExitCodeSuite) TestNonZeroOutputCodeExitsProcessNonZero() {
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess_CLIExitCode$")
	cmd.Env = append(os.Environ(), "GOOSE_CLI_EXIT_CODE_HELPER=1")

	err := cmd.Run()

	exitErr, ok := err.(*exec.ExitError)
	s.T.Expect(ok).ToBeTrue()
	if ok {
		s.T.Expect(exitErr.ExitCode()).ToEqual(1)
	}
}
