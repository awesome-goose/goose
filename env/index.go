package env

import (
	"strconv"

	"github.com/awesome-goose/goose/env/sources"
	"github.com/awesome-goose/goose/types"
)

type Env struct {
	store map[string]string
}

func NewEnv() *Env {
	e := &Env{store: make(map[string]string)}
	e.FromSources(sources.NewOsEnvSource(), sources.NewFileEnvSource())
	return e
}

// FromSources applies multiple EnvSources to populate the env
func (e *Env) FromSources(sources ...types.EnvSource) {
	for _, src := range sources {
		src.Load(e)
	}
}

// Get returns a value from the store or an empty string if not found
func (e *Env) Get(key string) string {
	if val, ok := e.store[key]; ok {
		return val
	}

	return ""
}

// Set assigns a value to the store
func (e *Env) Set(key, value string) {
	e.store[key] = value
}

// GetWithDefault returns a value from the store or the default if not found
func (e *Env) GetWithDefault(key, defaultValue string) string {
	if val, ok := e.store[key]; ok {
		return val
	}

	return defaultValue
}

// GetInt returns an integer value from the store
func (e *Env) GetInt(key string) int {
	val := e.Get(key)
	if val == "" {
		return 0
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return i
}

// GetBool returns a boolean value from the store
func (e *Env) GetBool(key string) bool {
	val := e.Get(key)
	if val == "" {
		return false
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return false
	}
	return b
}

// GetFloat returns a float64 value from the store
func (e *Env) GetFloat(key string) float64 {
	val := e.Get(key)
	if val == "" {
		return 0.0
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0.0
	}
	return f
}
