package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/awesome-goose/goose/utils/path"
)

// A deployed binary has no go.mod above it or the working directory. AppRoot
// must fall back to the working directory rather than fail, since the app
// root only locates optional resources and failing here aborts boot.
func TestAppRootFallsBackToWorkingDirectory(t *testing.T) {
	dir := t.TempDir() // no go.mod anywhere above a fresh temp dir
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	t.Setenv("APP_ROOT", "")

	got, err := path.AppRoot()
	if err != nil {
		t.Fatalf("AppRoot() error = %v, want the working directory", err)
	}
	if want, _ := filepath.EvalSymlinks(dir); got != dir && got != want {
		t.Fatalf("AppRoot() = %q, want %q", got, dir)
	}
}

func TestAppRootHonoursEnv(t *testing.T) {
	t.Setenv("APP_ROOT", "/srv/app")
	if got, _ := path.AppRoot(); got != "/srv/app" {
		t.Fatalf("AppRoot() = %q, want /srv/app", got)
	}
}
