package sql

import "github.com/awesome-goose/goose/types"

func Root(config *Config) types.Module {
	m := NewModule(config, true)
	return m
}

func Child(config *Config) types.Module {
	m := NewModule(config, false)
	return m
}
