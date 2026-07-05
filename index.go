package goose

import (
	"github.com/awesome-goose/goose/core"
	"github.com/awesome-goose/goose/types"
)

var defaultKernel = core.NewKernel()

// Start starts one or more platform instances
// - Single instance: runs directly
// - Multiple instances: API/Web run concurrently, CLI runs when `cli` arg is passed
//
// Example (single platform):
//
//	stop, err := goose.Start(goose.API(apiPlatform, apiModule, initializers))
//
// Example (multi-platform):
//
//	stop, err := goose.Start(
//		goose.API(apiPlatform, apiModule, apiInitializers),
//		goose.Web(webPlatform, webModule, webInitializers),
//		goose.SPA(spaPlatform, spaModule, spaInitializers),
//		goose.CLI(cliPlatform, cliModule, cliInitializers),
//	)
func Start(instances ...*types.Instance) (func() error, error) {
	return defaultKernel.Start(instances...)
}

// API creates an API platform instance
func API(platform types.Platform, module types.Module, initializers []func(container types.Container) error) *types.Instance {
	return &types.Instance{
		Name:         platform.Name(),
		Type:         types.PlatformTypeAPI,
		Platform:     platform,
		Module:       module,
		Initializers: initializers,
	}
}

// Web creates a Web platform instance
func Web(platform types.Platform, module types.Module, initializers []func(container types.Container) error) *types.Instance {
	return &types.Instance{
		Name:         platform.Name(),
		Type:         types.PlatformTypeWeb,
		Platform:     platform,
		Module:       module,
		Initializers: initializers,
	}
}

// SPA creates a SPA platform instance: a single HTTP service that serves
// JSON API routes under an API prefix and a single-page app's static assets
// (with index.html fallback for client-side routing) for everything else.
func SPA(platform types.Platform, module types.Module, initializers []func(container types.Container) error) *types.Instance {
	return &types.Instance{
		Name:         platform.Name(),
		Type:         types.PlatformTypeSPA,
		Platform:     platform,
		Module:       module,
		Initializers: initializers,
	}
}

// CLI creates a CLI platform instance
func CLI(platform types.Platform, module types.Module, initializers []func(container types.Container) error) *types.Instance {
	return &types.Instance{
		Name:         platform.Name(),
		Type:         types.PlatformTypeCLI,
		Platform:     platform,
		Module:       module,
		Initializers: initializers,
	}
}
