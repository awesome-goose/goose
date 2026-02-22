package router

import (
	"fmt"
	"slices"

	"github.com/awesome-goose/goose/types"
)

type resourceRoutes struct {
	route types.Route
}

func (r *resourceRoutes) All() types.Route {
	return r.route
}

func (r *resourceRoutes) Only(handlers ...string) types.Route {
	filteredChildren := types.Routes{}
	for _, child := range r.route.Children {
		handlerTuple, ok := child.Handler.([]any)
		if !ok || len(handlerTuple) < 2 {
			panic(fmt.Sprintf("handler for %s must be a tuple of [handler, methodName], got %T", r.route.Path, child.Handler))
		}

		handlerMethod, ok := handlerTuple[1].(string)
		if !ok {
			panic(fmt.Sprintf("handler method for %s must be a string, got %T", r.route.Path, handlerTuple[1]))
		}

		if slices.Contains(handlers, handlerMethod) {
			filteredChildren = append(filteredChildren, child)
		}
	}

	r.route.Children = filteredChildren
	return r.route
}

func (r *resourceRoutes) Except(handlers ...string) types.Route {
	filteredChildren := types.Routes{}
	for _, child := range r.route.Children {
		handlerTuple, ok := child.Handler.([]any)
		if !ok || len(handlerTuple) < 2 {
			panic(fmt.Sprintf("handler for %s must be a tuple of [handler, methodName], got %T", r.route.Path, child.Handler))
		}

		handlerMethod, ok := handlerTuple[1].(string)
		if !ok {
			panic(fmt.Sprintf("handler method for %s must be a string, got %T", r.route.Path, handlerTuple[1]))
		}

		if !slices.Contains(handlers, handlerMethod) {
			filteredChildren = append(filteredChildren, child)
		}
	}

	r.route.Children = filteredChildren
	return r.route
}

func Resource(path string, handler any, middlewares ...types.Middleware) *resourceRoutes {
	return &resourceRoutes{
		route: types.Route{
			Path:        path,
			Middlewares: middlewares,
			Children: types.Routes{
				Post("/", []any{handler, "Create"}),
				Get("/", []any{handler, "List"}),
				Get("/:id", []any{handler, "Get"}),
				Patch("/:id", []any{handler, "Update"}),
				Delete("/:id", []any{handler, "Delete"}),
			},
		},
	}
}
