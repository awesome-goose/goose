package goose

import (
	"github.com/awesome-goose/goose/core"
	"github.com/awesome-goose/goose/types"
)

var (
	defaultKernel = core.NewKernel()
)

func Start(platform types.Platform, module types.Module, initializers []func(container types.Container) error) (func() error, error) {
	return defaultKernel.Start(platform, module, initializers)
}
