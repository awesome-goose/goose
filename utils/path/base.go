package path

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

// UserHome returns the current user's home directory.
func UserHome() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

// AppRoot returns the directory of the calling file (e.g., where main.go is).
func AppRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to determine caller location")
	}
	return filepath.Dir(filename), nil
}

// WorkingDir returns the current working directory where the app is run from.
func WorkingDir() (string, error) {
	return os.Getwd()
}
