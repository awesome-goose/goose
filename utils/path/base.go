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

// AppRoot returns the application root.
//
// Resolution order: the APP_ROOT environment variable; the nearest directory
// containing go.mod walking up from the executable, then from the working
// directory (the `go run` / development case); finally the working directory
// itself. A deployed binary has no go.mod beside it, and the root is only
// used to locate optional resources (config/, lang/, ...), so failing to boot
// there would be wrong.
func AppRoot() (string, error) {
	if root := os.Getenv("APP_ROOT"); root != "" {
		return root, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Try from executable location
	dir := exe
	if fi, err := os.Stat(exe); err == nil && fi.IsDir() {
		// If exe is a directory, use as is
	} else {
		dir = strings.TrimSuffix(exe, "/"+filepath.Base(exe))
	}
	if root := searchGoMod(dir); root != "" {
		return root, nil
	}

	// Try from current working directory
	cwd, err := os.Getwd()
	if err == nil {
		if root := searchGoMod(cwd); root != "" {
			return root, nil
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd, nil
	}
	return "", os.ErrNotExist
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
