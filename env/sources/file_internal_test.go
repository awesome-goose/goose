package sources

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/awesome-goose/goose/types"
)

// memEnv is a types.Env (package env imports this one, so it can't be used).
type memEnv map[string]string

func (m memEnv) FromSources(...types.EnvSource) {}
func (m memEnv) Get(k string) string            { return m[k] }
func (m memEnv) Set(k, v string)                { m[k] = v }
func (m memEnv) GetWithDefault(k, d string) string {
	if v, ok := m[k]; ok {
		return v
	}
	return d
}
func (m memEnv) GetInt(string) int       { return 0 }
func (m memEnv) GetBool(string) bool     { return false }
func (m memEnv) GetFloat(string) float64 { return 0 }

// capture swaps stderr and exit for the test and returns what was written and
// the exit status (-1 when exit was not called).
func capture(t *testing.T) (*bytes.Buffer, *int) {
	t.Helper()
	buf, code := &bytes.Buffer{}, -1
	oldErr, oldExit := stderr, exit
	stderr, exit = buf, func(c int) { code = c }
	t.Cleanup(func() { stderr, exit = oldErr, oldExit })
	return buf, &code
}

func appRoot(t *testing.T, withEnv string) string {
	t.Helper()
	dir := t.TempDir()
	if withEnv != "" {
		if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(withEnv), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("APP_ROOT", dir)
	return dir
}

func unset(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			t.Cleanup(func() { os.Setenv(k, v) })
		} else {
			t.Cleanup(func() { os.Unsetenv(k) })
		}
		os.Unsetenv(k)
	}
}

// Lane 3's case: the root resolved somewhere without a .env and the app ran on
// defaults (DB_NAME=yan) without a word. Now it says where it looked and why.
func TestMissingEnvFileWarnsWithTheRootAndWhy(t *testing.T) {
	unset(t, EnvFileVar, EnvFileRequiredVar)
	dir := appRoot(t, "")
	out, code := capture(t)
	NewFileEnvSource().Load(memEnv{})
	msg := out.String()
	if !strings.Contains(msg, filepath.Join(dir, ".env")) || !strings.Contains(msg, "APP_ROOT") || !strings.Contains(msg, "warning") {
		t.Fatalf("warning = %q", msg)
	}
	if *code != -1 {
		t.Fatalf("exited %d without being asked to", *code)
	}
}

func TestRequiredEnvFileStopsTheProcess(t *testing.T) {
	unset(t, EnvFileVar)
	appRoot(t, "")
	t.Setenv(EnvFileRequiredVar, "true")
	out, code := capture(t)
	NewFileEnvSource().Load(memEnv{})
	if *code != 78 || !strings.Contains(out.String(), "cannot read") {
		t.Fatalf("exit %d, %q", *code, out.String())
	}
}

func TestExplicitEnvFileIsReadAndAMissingOneStops(t *testing.T) {
	unset(t, EnvFileRequiredVar, "GOOSE_TEST_FROM_FILE")
	appRoot(t, "GOOSE_TEST_FROM_FILE=root\n")
	f := filepath.Join(t.TempDir(), "custom.env")
	if err := os.WriteFile(f, []byte("GOOSE_TEST_FROM_FILE=custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvFileVar, f)
	out, code := capture(t)
	NewFileEnvSource().Load(memEnv{})
	if os.Getenv("GOOSE_TEST_FROM_FILE") != "custom" || *code != -1 || out.Len() != 0 {
		t.Fatalf("value %q, exit %d, %q", os.Getenv("GOOSE_TEST_FROM_FILE"), *code, out.String())
	}

	t.Setenv(EnvFileVar, f+".missing")
	NewFileEnvSource().Load(memEnv{})
	if *code != 78 {
		t.Fatalf("a missing GOOSE_ENV_FILE did not stop the process: exit %d", *code)
	}
}

func TestEnvFileDashMeansNoFileQuietly(t *testing.T) {
	unset(t, EnvFileRequiredVar)
	appRoot(t, "")
	t.Setenv(EnvFileVar, "-")
	out, code := capture(t)
	NewFileEnvSource().Load(memEnv{})
	if out.Len() != 0 || *code != -1 {
		t.Fatalf("%q, exit %d", out.String(), *code)
	}
}

func TestFoundEnvFileIsQuiet(t *testing.T) {
	unset(t, EnvFileVar, EnvFileRequiredVar, "GOOSE_TEST_FROM_FILE")
	appRoot(t, "GOOSE_TEST_FROM_FILE=root\n")
	out, code := capture(t)
	NewFileEnvSource().Load(memEnv{})
	if os.Getenv("GOOSE_TEST_FROM_FILE") != "root" || out.Len() != 0 || *code != -1 {
		t.Fatalf("value %q, %q, exit %d", os.Getenv("GOOSE_TEST_FROM_FILE"), out.String(), *code)
	}
}
