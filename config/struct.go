package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/utils/props"
)

// Export loads a specific YAML file from disk and unmarshals it into the given struct.
func (c *Config) Export(namespace string, config any) error {
	// Try both .yaml and .yml extensions
	extensions := []string{".yaml", ".yml"}

	var filePath string
	found := false
	for _, ext := range extensions {
		path := filepath.Join(c.Dir(), namespace+ext)
		if _, err := os.Stat(path); err == nil {
			filePath = path
			found = true
			break
		}
	}

	if !found {
		return errors.ErrConfigFileNotFound.WithMeta(namespace)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return errors.ErrFailedToReadConfigFile.WithError(err)
	}

	if err := yaml.Unmarshal(content, config); err != nil {
		return errors.ErrFailedToUnmarshalConfigToStruct.WithError(err)
	}

	return nil
}

func (c *Config) Import(namespace string, in any) error {
	if namespace == "" {
		return errors.ErrNamespaceRequired
	}

	// Marshal the struct to JSON first
	bytes, err := json.Marshal(in)
	if err != nil {
		return errors.ErrFailedToMarshalStruct.WithError(err)
	}

	// Unmarshal back to Props
	var asMap props.Props
	if err := json.Unmarshal(bytes, &asMap); err != nil {
		return errors.ErrFailedToConvertStructToMap.WithError(err)
	}

	// Store in the internal tree
	c.tree[namespace] = asMap
	return nil
}
