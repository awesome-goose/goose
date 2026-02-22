package config

import (
	"strings"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/utils/props"
)

// Get retrieves a value using dotted path (e.g., "db.host") and renders {{ENV_VAR}}.
func (c *Config) Get(path string) (any, error) {
	parts := strings.Split(path, ".")
	current := any(c.tree)
	for _, part := range parts {
		m, ok := current.(props.Props)
		if !ok {
			return nil, errors.ErrPathNotFound.WithMeta(path)
		}
		current, ok = m[part]
		if !ok {
			return nil, errors.ErrKeyNotFound.WithMeta(path)
		}
	}
	if str, ok := current.(string); ok {
		return renderEnv(str), nil
	}
	return current, nil
}

// Set sets a value using dotted path (e.g., "db.host").
func (c *Config) Set(path string, value any) error {
	parts := strings.Split(path, ".")
	lastKey := parts[len(parts)-1]
	current := c.tree
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part]
		if !ok {
			next = props.Props{}
			current[part] = next
		}
		asMap, ok := next.(props.Props)
		if !ok {
			return errors.ErrInvalidSet.WithMeta(path)
		}
		current = asMap
	}
	current[lastKey] = value
	return nil
}
