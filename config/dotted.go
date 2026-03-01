package config

import (
	"strconv"
	"strings"

	"github.com/awesome-goose/goose/utils/props"
)

// Get retrieves a value using dotted path (e.g., "db.host") and renders {{ENV_VAR}}.
func (c *Config) Get(path string) string {
	parts := strings.Split(path, ".")
	current := any(c.tree)
	for _, part := range parts {
		m, ok := current.(props.Props)
		if !ok {
			return ""
		}
		current, ok = m[part]
		if !ok {
			return ""
		}
	}

	if str, ok := current.(string); ok {
		return renderEnv(str)
	}

	return ""
}

// Set sets a value using dotted path (e.g., "db.host").
func (c *Config) Set(path string, value any) {
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
			return
		}
		current = asMap
	}
	current[lastKey] = value
}

func (c *Config) GetWithDefault(key, defaultValue string) string {
	value := c.Get(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func (c *Config) GetInt(key string) int {
	value := c.Get(key)
	if value == "" {
		return 0
	}
	i, _ := strconv.Atoi(value)
	return i
}

func (c *Config) GetBool(key string) bool {
	value := c.Get(key)
	if value == "" {
		return false
	}
	b, _ := strconv.ParseBool(value)
	return b
}

func (c *Config) GetFloat(key string) float64 {
	value := c.Get(key)
	if value == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(value, 64)
	return f
}
