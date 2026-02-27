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

// AppRoot returns the directory where the compiled executable is located.
func AppRoot() (string, error) {
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
