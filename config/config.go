package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/awesome-goose/goose/utils/path"
	"github.com/awesome-goose/goose/utils/props"
	"gopkg.in/yaml.v3"
)

type AppConfigPath string

func (p AppConfigPath) String() string {
	return string(p)
}

type Config struct {
	dir  string
	tree props.Props
}

func NewConfig(appPath AppConfigPath) (*Config, error) {
	dir := appPath.String()
	if dir == "" {
		defaultDir, err := path.Config()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve config path: %w", err)
		}

		dir = defaultDir
	}

	tree := props.Props{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		var data props.Props
		if unmarshalErr := yaml.Unmarshal(content, &data); unmarshalErr != nil {
			return unmarshalErr
		}

		key := strings.TrimSuffix(filepath.Base(path), ext)
		tree[key] = data

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Config{
		tree: tree,
		dir:  dir,
	}, nil
}

func (c *Config) Dir() string {
	return c.dir
}

func (c *Config) Tree() props.Props {
	return c.tree
}
