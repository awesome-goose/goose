package router

import "github.com/awesome-goose/goose/types"

// ForRoutes wraps a static list of routes in a Module that registers them
// with the kernel during Boot.
//
// Each call returns a fresh module — there is no cross-call singleton — so
// importing the result into two different parent modules registers the same
// routes once per parent (an explicit, opt-in behavior) rather than leaking
// every ForRoutes call across every importer.
func ForRoutes(routes ...types.Route) types.Module {
	return &staticRouter{routes: routes}
}

// ForRouters wraps one or more types.Router implementations (typically
// controllers or generators resolved from the DI container) in a Module.
// Each call returns a fresh module.
func ForRouters(routers ...types.Router) types.Module {
	return &dynamicRouters{routers: routers}
}
