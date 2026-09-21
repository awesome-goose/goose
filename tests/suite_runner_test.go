package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	test "github.com/awesome-goose/goose/testing"
)

const suiteFixtureEnv = "GOOSE_SUITE_RUNNER_FIXTURE"

// suiteFixture fails on purpose. It only runs when TestSuiteRunnerAttributesFailures
// re-executes this test binary with suiteFixtureEnv set.
type suiteFixture struct {
	test.Suite
}

func (s *suiteFixture) TestAssertionFails() { s.T.Expect(1).ToEqual(2) }
func (s *suiteFixture) TestRequireFails()   { s.T.Require(1).ToEqual(2) }
func (s *suiteFixture) TestPasses()         { s.T.Expect(1).ToEqual(1) }

func TestSuiteRunnerFixture(t *testing.T) {
	if os.Getenv(suiteFixtureEnv) == "" {
		t.Skip("runs only as a child process of TestSuiteRunnerAttributesFailures")
	}
	test.NewSuiteRunner(t, &suiteFixture{}).Run()
}

// A failing assertion must fail the suite method that made it, and only that
// method. The runner used to bind assertions to the parent *testing.T, so the
// failure landed on the parent, the failing method showed as PASS, and a failed
// Require called FailNow on the parent from the subtest's goroutine.
func TestSuiteRunnerAttributesFailures(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestSuiteRunnerFixture$", "-test.v", "-test.timeout=60s")
	cmd.Env = append(os.Environ(), suiteFixtureEnv+"=1")
	raw, _ := cmd.CombinedOutput() // exits non-zero on purpose
	out := string(raw)

	for _, want := range []string{
		"--- FAIL: TestSuiteRunnerFixture/TestAssertionFails",
		"--- FAIL: TestSuiteRunnerFixture/TestRequireFails",
		"--- PASS: TestSuiteRunnerFixture/TestPasses",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("child output does not contain %q\n--- child output ---\n%s", want, out)
		}
	}
	if strings.Contains(out, "panic:") {
		t.Errorf("child run panicked\n--- child output ---\n%s", out)
	}
}
