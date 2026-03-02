package core

import (
	"strings"
	"sync"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/types"
)

// maxRouteCacheSize is the maximum number of routes to cache
const maxRouteCacheSize = 1000

// routeCacheEntry stores a route with its extracted params for caching
type routeCacheEntry struct {
	route  *types.Route
	params map[string]string
}

type router struct {
	mu    sync.RWMutex
	cache map[string]*routeCacheEntry
}

func NewRouter() *router {
	return &router{
		cache: make(map[string]*routeCacheEntry),
	}
}

// ClearCache clears the route cache. Useful when routes are dynamically modified.
func (r *router) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = make(map[string]*routeCacheEntry)
}

func (r *router) Find(routes types.Routes, method string, paths []string) (*types.Route, map[string]string, error) {
	if len(paths) == 0 {
		paths = []string{"/"}
	}

	cacheKey := method + ":" + strings.Join(paths, "/")

	// Check cache with read lock
	r.mu.RLock()
	if entry, found := r.cache[cacheKey]; found {
		r.mu.RUnlock()
		// Return a copy of params to prevent mutation
		paramsCopy := make(map[string]string, len(entry.params))
		for k, v := range entry.params {
			paramsCopy[k] = v
		}
		return entry.route, paramsCopy, nil
	}
	r.mu.RUnlock()

	foundRoute, params, err := r.findRecursive(routes, method, paths, nil, make(map[string]string))
	if err != nil {
		return nil, nil, err
	}

	if foundRoute != nil {
		// Cache the result with write lock
		r.mu.Lock()
		// Evict cache if it exceeds max size (simple eviction: clear all)
		if len(r.cache) >= maxRouteCacheSize {
			r.cache = make(map[string]*routeCacheEntry)
		}
		r.cache[cacheKey] = &routeCacheEntry{
			route:  foundRoute,
			params: params,
		}
		r.mu.Unlock()
		return foundRoute, params, nil
	}

	return nil, nil, errors.ErrRouteNotFound.WithMeta(map[string]any{"method": method, "path": strings.Join(paths, "/")})
}

func (r *router) findRecursive(currentRoutes types.Routes, method string, paths []string, collectedMiddlewares types.Middlewares, params map[string]string) (*types.Route, map[string]string, error) {
	if len(paths) == 0 {
		return nil, params, nil
	}

	var wildcardRoute *types.Route

	currentPathSegment := paths[0]
	remainingPaths := paths[1:]

	for _, route := range currentRoutes {
		// Aggregate middlewares from the current level.
		newMiddlewares := append(collectedMiddlewares, route.Middlewares...)

		// Check for a wildcard route at this level and save it as a fallback.
		if route.IsPath("*") {
			wildcardRoute = &route
			continue
		}

		// Normalize route path for parameter and segment matching
		routePath := route.Path
		if len(routePath) > 0 && routePath[0] == '/' {
			routePath = routePath[1:]
		}

		// Handle multi-segment route paths (e.g., "user/:id")
		routeSegments := strings.Split(routePath, "/")
		if len(routeSegments) > 1 {
			// Multi-segment route path - match all segments at once
			if len(paths) < len(routeSegments) {
				continue // Not enough path segments to match
			}

			branchParams := make(map[string]string)
			for k, v := range params {
				branchParams[k] = v
			}

			matched := true
			for i, seg := range routeSegments {
				if len(seg) > 0 && seg[0] == ':' {
					// Parameter segment
					branchParams[seg[1:]] = paths[i]
				} else if seg != paths[i] {
					matched = false
					break
				}
			}

			if !matched {
				continue
			}

			// Consume all matched segments
			remainingAfterMulti := paths[len(routeSegments):]

			if len(remainingAfterMulti) == 0 {
				if route.IsMethod(method) {
					finalRoute := route.Clone()
					finalRoute.Middlewares = newMiddlewares
					return &finalRoute, branchParams, nil
				}
			}

			// Has more paths - check children
			if route.HasChildren() {
				foundRoute, foundParams, err := r.findRecursive(route.Children, method, remainingAfterMulti, newMiddlewares, branchParams)
				if err == nil && foundRoute != nil {
					return foundRoute, foundParams, nil
				}
			}
			continue
		}

		// Single-segment route path (original logic)
		// Parameterized path support
		isParam := false
		paramName := ""
		if len(routePath) > 0 && routePath[0] == ':' {
			isParam = true
			paramName = routePath[1:]
		}

		if !route.IsPath(currentPathSegment) && routePath != currentPathSegment && !isParam {
			continue
		}

		// Copy params for this branch
		branchParams := make(map[string]string)
		for k, v := range params {
			branchParams[k] = v
		}
		if isParam {
			branchParams[paramName] = currentPathSegment
		}

		// If this is the last path segment, we have a potential match.
		if len(remainingPaths) == 0 {
			if route.IsMethod(method) {
				finalRoute := route.Clone()
				finalRoute.Middlewares = newMiddlewares
				return &finalRoute, branchParams, nil
			}

			// Special case: if this route has children, try to match a child with Path: "/" or parameter and the correct method
			if route.HasChildren() {
				for _, child := range route.Children {
					childPath := child.Path
					if len(childPath) > 0 && childPath[0] == '/' {
						childPath = childPath[1:]
					}
					childIsParam := false
					childParamName := ""
					if len(childPath) > 0 && childPath[0] == ':' {
						childIsParam = true
						childParamName = childPath[1:]
					}
					if (child.IsPath("/") || childIsParam) && child.IsMethod(method) {
						finalRoute := child.Clone()
						finalRoute.Middlewares = append(newMiddlewares, child.Middlewares...)
						childParams := make(map[string]string)
						for k, v := range branchParams {
							childParams[k] = v
						}
						if childIsParam {
							childParams[childParamName] = "/"
						}
						return &finalRoute, childParams, nil
					}
				}
			}
		}

		// If there are more path segments, traverse into children.
		if route.HasChildren() {
			foundRoute, foundParams, err := r.findRecursive(route.Children, method, remainingPaths, newMiddlewares, branchParams)
			if err != nil {
				if foundRoute != nil {
					return foundRoute, foundParams, nil
				}
			} else if foundRoute != nil {
				return foundRoute, foundParams, nil
			}
		}
	}

	if wildcardRoute != nil {
		return wildcardRoute, params, nil
	}

	return nil, params, nil // No route found at this level.
}
