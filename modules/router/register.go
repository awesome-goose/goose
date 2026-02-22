package router

import "github.com/awesome-goose/goose/types"

var (
	router = &routerModule{}
)

func ForRouters(routers ...types.Router) types.Module {
	router.Append(routers...)
	return router
}

func ForRoutes(routes ...types.Route) types.Module {
	router.Append(&staticRouter{routes: routes})
	return router
}
