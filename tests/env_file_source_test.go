package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/awesome-goose/goose/env"
	"github.com/awesome-goose/goose/env/sources"
	test "github.com/awesome-goose/goose/testing"
)

func TestEnvFileSource(t *testing.T) {
	test.NewSuiteRunner(t, &EnvFileSourceSuite{}).Run()
}

type EnvFileSourceSuite struct {
	test.Suite
	origWD string
}

func (s *EnvFileSourceSuite) SetupTest() {
	wd, err := os.Getwd()
	s.T.Require(err).ToBeNil()
	s.origWD = wd
}

func (s *EnvFileSourceSuite) TeardownTest() {
	s.T.Require(os.Chdir(s.origWD)).ToBeNil()
}

// chdirToAppRootWithEnv creates <tmp>/go.mod (so utils/path.AppRoot resolves
// here, not further up the real tree) and <tmp>/.env with body, then cd's
// into it.
func (s *EnvFileSourceSuite) chdirToAppRootWithEnv(body string) {
	dir := s.T.T().TempDir()
	s.T.Require(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module scratch\n\ngo 1.25\n"), 0o644)).ToBeNil()
	s.T.Require(os.WriteFile(filepath.Join(dir, ".env"), []byte(body), 0o644)).ToBeNil()
	s.T.Require(os.Chdir(dir)).ToBeNil()
}

// A real OS environment variable — set by a deployment platform, CI secret, or
// a developer's shell — must win over a value the checked-in (or accidentally
// deployed) .env file happens to also define; a stale or example .env must
// never silently override it. This was backwards: Load unconditionally called
// os.Setenv for every key in the file.
func (s *EnvFileSourceSuite) TestRealEnvVarWinsOverTheFile() {
	s.chdirToAppRootWithEnv("DB_NAME=from_file\n")
	os.Setenv("DB_NAME", "from_real_env")
	defer os.Unsetenv("DB_NAME")

	e := env.NewEnv()

	s.T.Expect(e.Get("DB_NAME")).ToEqual("from_real_env")
	s.T.Expect(os.Getenv("DB_NAME")).ToEqual("from_real_env")
}

// The whole point of a .env file: fill in what the real environment doesn't
// already set.
func (s *EnvFileSourceSuite) TestFileFillsInWhatTheRealEnvDoesNotSet() {
	s.chdirToAppRootWithEnv("DB_NAME=from_file\n")
	os.Unsetenv("DB_NAME")

	e := env.NewEnv()

	s.T.Expect(e.Get("DB_NAME")).ToEqual("from_file")
	s.T.Expect(os.Getenv("DB_NAME")).ToEqual("from_file")
}

// A real env var explicitly set to "" is a deliberate choice (e.g. "disable
// this"), not an absence — the file must not fill it in either.
func (s *EnvFileSourceSuite) TestExplicitEmptyRealEnvVarIsNotOverridden() {
	s.chdirToAppRootWithEnv("DB_NAME=from_file\n")
	os.Setenv("DB_NAME", "")
	defer os.Unsetenv("DB_NAME")

	sources.NewFileEnvSource().Load(env.NewEnv())

	s.T.Expect(os.Getenv("DB_NAME")).ToEqual("")
}

func (s *EnvFileSourceSuite) TestMissingEnvFileIsSilentlyIgnored() {
	dir := s.T.T().TempDir()
	s.T.Require(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module scratch\n\ngo 1.25\n"), 0o644)).ToBeNil()
	s.T.Require(os.Chdir(dir)).ToBeNil()

	sources.NewFileEnvSource().Load(env.NewEnv()) // must not panic
}
