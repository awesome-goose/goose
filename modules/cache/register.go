package cache

import "github.com/awesome-goose/goose/types"

// Root creates a root Cache module that initializes the CacheStore table and registers the Cache service.
// Use this in the main application module.
func Root(config *Config) types.Module {
	return NewModule(config, true)
}

// Child creates a child Cache module that resolves the already-registered Cache service.
// Use this in sub-modules that need access to the Cache service.
func Child() types.Module {
	return NewModule(nil, false)
}
