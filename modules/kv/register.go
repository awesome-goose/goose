package kv

import "github.com/awesome-goose/goose/types"

// Root creates a root KV module that initializes the kv_store table and registers the KV service.
// Use this in the main application module.
func Root(config *Config) types.Module {
	return NewModule(config, true)
}

// Child creates a child KV module that resolves the already-registered KV service.
// Use this in sub-modules that need access to the KV service.
func Child() types.Module {
	return NewModule(nil, false)
}
