package router

import "github.com/awesome-goose/goose/types"

func Route(method types.Method, path string, handler any, middlewares types.Middlewares, children ...types.Route) types.Route {
	return types.Route{
		Method:      method,
		Path:        path,
		Handler:     handler,
		Middlewares: middlewares,
		Children:    children,
	}
}

func Get(path string, handler any, middlewares ...types.Middleware) types.Route {
	return Route(types.GET, path, handler, middlewares)
}

func Post(path string, handler any, middlewares ...types.Middleware) types.Route {
	return Route(types.POST, path, handler, middlewares)
}

func Put(path string, handler any, middlewares ...types.Middleware) types.Route {
	return Route(types.PUT, path, handler, middlewares)
}

func Delete(path string, handler any, middlewares ...types.Middleware) types.Route {
	return Route(types.DELETE, path, handler, middlewares)
}

func Patch(path string, handler any, middlewares ...types.Middleware) types.Route {
	return Route(types.PATCH, path, handler, middlewares)
}

func Cli(path string, handler any, middlewares ...types.Middleware) types.Route {
	return Route(types.GET, path, handler, middlewares)
}
