package path

import (
	"fmt"
)

const (
	CONFIG_PATH   = "config"
	APP_PATH      = "app"
	DATABASE_PATH = "database"
	LANG_PATH     = "lang"
	PUBLIC_PATH   = "public"
	ASSETS_PATH   = "assets"
	STORAGE_PATH  = "storage"
)

func Config() (string, error) {
	appRoot, err := AppRoot()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", appRoot, CONFIG_PATH), nil
}

func App() (string, error) {
	appRoot, err := AppRoot()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", appRoot, APP_PATH), nil
}

func Database() (string, error) {
	appRoot, err := AppRoot()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", appRoot, DATABASE_PATH), nil
}

func Lang() (string, error) {
	appRoot, err := AppRoot()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", appRoot, LANG_PATH), nil
}

func Public() (string, error) {
	appRoot, err := AppRoot()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", appRoot, PUBLIC_PATH), nil
}

func Assets() (string, error) {
	appRoot, err := AppRoot()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", appRoot, ASSETS_PATH), nil
}

func Storage() (string, error) {
	appRoot, err := AppRoot()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", appRoot, STORAGE_PATH), nil
}
