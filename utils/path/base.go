package path

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

// UserHome returns the current user's home directory.
func UserHome() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

// How AppRoot found the root (ResolveAppRoot's second result).
const (
	RootFromEnv        = "APP_ROOT"
	RootFromExecutable = "go.mod above the executable"
	RootFromWorkingDir = "go.mod above the working directory"
	RootIsWorkingDir   = "working directory"
)

// AppRoot returns the application root.
//
// Resolution order: the APP_ROOT environment variable; the nearest directory
// containing go.mod walking up from the executable, then from the working
// directory (the `go run` / development case); finally the working directory
// itself. A deployed binary has no go.mod beside it, and the root is only
// used to locate optional resources (config/, lang/, .env, ...), so failing to
// boot there would be wrong.
//
// The walk up from the executable takes the first go.mod it meets: a binary
// built into some other module's directory (a bin/ inside another checkout)
// resolves to that module, not to the app. Set APP_ROOT for such a layout;
// ResolveAppRoot says which rule picked the root, and the .env loader warns
// when the root it used has no .env (see env/sources).
func AppRoot() (string, error) {
	root, _, err := ResolveAppRoot()
	return root, err
}

// ResolveAppRoot is AppRoot plus how the root was found (RootFromEnv,
// RootFromExecutable, RootFromWorkingDir or RootIsWorkingDir), for messages
// that must say why a path was used.
func ResolveAppRoot() (root, how string, err error) {
	if root := os.Getenv("APP_ROOT"); root != "" {
		return root, RootFromEnv, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return "", "", err
	}

	// Try from executable location
	dir := exe
	if fi, err := os.Stat(exe); err == nil && fi.IsDir() {
		// If exe is a directory, use as is
	} else {
		dir = strings.TrimSuffix(exe, "/"+filepath.Base(exe))
	}

	if root := searchGoMod(dir); root != "" {
		return root, RootFromExecutable, nil
	}

	// Try from current working directory
	cwd, err := os.Getwd()
	if err == nil {
		if root := searchGoMod(cwd); root != "" {
			return root, RootFromWorkingDir, nil
		}
		return cwd, RootIsWorkingDir, nil
	}

	return "", "", os.ErrNotExist
}

// searchGoMod walks up from startDir looking for go.mod
func searchGoMod(startDir string) string {
	dir := startDir
	for {
		modPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modPath); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// CurrentDir returns the current working directory.
func CurrentDir() (string, error) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return "", os.ErrNotExist
	}
	return filepath.Dir(file), nil
}
