package tests

import (
	"testing"

	"github.com/awesome-goose/goose/core"
	test "github.com/awesome-goose/goose/testing"
	"github.com/awesome-goose/goose/types"
)

func TestRouter(t *testing.T) {
	test.NewSuiteRunner(t, &RouterSuite{}).Run()
}

type RouterSuite struct {
	test.Suite
	router types.RouterFinder
}

func (s *RouterSuite) SetupTest() {
	s.router = core.NewRouter()
}

func (s *RouterSuite) TestFind_SimpleRouteMatch() {
	routes := types.Routes{
		{Method: "GET", Path: "users"},
	}

	route, params, err := s.router.Find(routes, "GET", []string{"users"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(route.Path).ToEqual("users")
	s.T.Expect(params).ToHaveLength(0)
}

func (s *RouterSuite) TestFind_RouteWithLeadingSlash() {
	routes := types.Routes{
		{Method: "GET", Path: "/users"},
	}

	route, params, err := s.router.Find(routes, "GET", []string{"users"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(params).ToHaveLength(0)
}

func (s *RouterSuite) TestFind_NestedRouteMatch() {
	routes := types.Routes{
		{
			Path: "api",
			Children: types.Routes{
				{Method: "GET", Path: "users"},
			},
		},
	}

	route, params, err := s.router.Find(routes, "GET", []string{"api", "users"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(route.Path).ToEqual("users")
	s.T.Expect(params).ToHaveLength(0)
}

func (s *RouterSuite) TestFind_ParameterExtraction() {
	routes := types.Routes{
		{Method: "GET", Path: ":id"},
	}

	route, params, err := s.router.Find(routes, "GET", []string{"123"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(params["id"]).ToEqual("123")
}

func (s *RouterSuite) TestFind_NestedParameterExtraction() {
	routes := types.Routes{
		{
			Path: "users",
			Children: types.Routes{
				{Method: "GET", Path: ":id"},
			},
		},
	}

	route, params, err := s.router.Find(routes, "GET", []string{"users", "456"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(params["id"]).ToEqual("456")
}

func (s *RouterSuite) TestFind_MultipleParameters() {
	routes := types.Routes{
		{
			Path: "users",
			Children: types.Routes{
				{
					Path: ":userId",
					Children: types.Routes{
						{Method: "GET", Path: ":action"},
					},
				},
			},
		},
	}

	route, params, err := s.router.Find(routes, "GET", []string{"users", "123", "edit"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(params["userId"]).ToEqual("123")
	s.T.Expect(params["action"]).ToEqual("edit")
}

func (s *RouterSuite) TestFind_WildcardRoute() {
	routes := types.Routes{
		{Method: "GET", Path: "*"},
	}

	route, params, err := s.router.Find(routes, "GET", []string{"anything"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(route.Path).ToEqual("*")
	s.T.Expect(params).ToHaveLength(0)
}

func (s *RouterSuite) TestFind_MethodMismatch() {
	routes := types.Routes{
		{Method: "GET", Path: "users"},
	}

	route, params, err := s.router.Find(routes, "POST", []string{"users"})

	s.T.Expect(err).Not().ToBeNil()
	s.T.Expect(route).ToBeNil()
	s.T.Expect(params).ToBeNil()
}

func (s *RouterSuite) TestFind_RouteNotFound() {
	routes := types.Routes{
		{Method: "GET", Path: "users"},
	}

	route, params, err := s.router.Find(routes, "GET", []string{"nonexistent"})

	s.T.Expect(err).Not().ToBeNil()
	s.T.Expect(route).ToBeNil()
	s.T.Expect(params).ToBeNil()
}

func (s *RouterSuite) TestFind_EmptyPaths() {
	routes := types.Routes{
		{Method: "GET", Path: "/"},
	}

	route, params, err := s.router.Find(routes, "GET", []string{})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(route.Path).ToEqual("/")
	s.T.Expect(params).ToHaveLength(0)
}

func (s *RouterSuite) TestFind_PostMethod() {
	routes := types.Routes{
		{Method: "POST", Path: "users"},
	}

	route, _, err := s.router.Find(routes, "POST", []string{"users"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(route.Method).ToEqual(types.Method("POST"))
}

func (s *RouterSuite) TestFind_PutMethod() {
	routes := types.Routes{
		{Method: "PUT", Path: "users"},
	}

	route, _, err := s.router.Find(routes, "PUT", []string{"users"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(route.Method).ToEqual(types.Method("PUT"))
}

func (s *RouterSuite) TestFind_DeleteMethod() {
	routes := types.Routes{
		{Method: "DELETE", Path: "users"},
	}

	route, _, err := s.router.Find(routes, "DELETE", []string{"users"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(route.Method).ToEqual(types.Method("DELETE"))
}

// testMiddleware is a simple middleware for testing
type testMiddleware struct {
	name string
}

func (m *testMiddleware) Handle(ctx types.Context) error {
	return nil
}

func (s *RouterSuite) TestFind_MiddlewareAggregation() {
	middleware1 := &testMiddleware{name: "m1"}
	middleware2 := &testMiddleware{name: "m2"}

	routes := types.Routes{
		{
			Path:        "api",
			Middlewares: types.Middlewares{middleware1},
			Children: types.Routes{
				{
					Method:      "GET",
					Path:        "users",
					Middlewares: types.Middlewares{middleware2},
				},
			},
		},
	}

	route, _, err := s.router.Find(routes, "GET", []string{"api", "users"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(route.Middlewares).ToHaveLength(2)
}

func (s *RouterSuite) TestFind_DeepNesting() {
	routes := types.Routes{
		{
			Path: "api",
			Children: types.Routes{
				{
					Path: "v1",
					Children: types.Routes{
						{
							Path: "users",
							Children: types.Routes{
								{Method: "GET", Path: ":id"},
							},
						},
					},
				},
			},
		},
	}

	route, params, err := s.router.Find(routes, "GET", []string{"api", "v1", "users", "999"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(params["id"]).ToEqual("999")
}

func (s *RouterSuite) TestFind_MultiSegmentPath() {
	routes := types.Routes{
		{Method: "GET", Path: "user/:id"},
	}

	route, params, err := s.router.Find(routes, "GET", []string{"user", "123"})

	s.T.Expect(err).ToBeNil()
	s.T.Expect(route).Not().ToBeNil()
	s.T.Expect(params["id"]).ToEqual("123")
}

func (s *RouterSuite) TestFind_CaseSensitiveMethod() {
	routes := types.Routes{
		{Method: "GET", Path: "users"},
	}

	// Router is case-sensitive - lowercase "get" won't match "GET"
	route, _, err := s.router.Find(routes, "get", []string{"users"})

	s.T.Expect(err).Not().ToBeNil()
	s.T.Expect(route).ToBeNil()
}

func (s *RouterSuite) TestFind_MultipleRoutesSameLevel() {
	routes := types.Routes{
		{Method: "GET", Path: "users"},
		{Method: "GET", Path: "posts"},
		{Method: "GET", Path: "comments"},
	}

	route1, _, err1 := s.router.Find(routes, "GET", []string{"users"})
	route2, _, err2 := s.router.Find(routes, "GET", []string{"posts"})
	route3, _, err3 := s.router.Find(routes, "GET", []string{"comments"})

	s.T.Expect(err1).ToBeNil()
	s.T.Expect(route1.Path).ToEqual("users")
	s.T.Expect(err2).ToBeNil()
	s.T.Expect(route2.Path).ToEqual("posts")
	s.T.Expect(err3).ToBeNil()
	s.T.Expect(route3.Path).ToEqual("comments")
}

func (s *RouterSuite) TestFind_SamePathDifferentMethods() {
	routes := types.Routes{
		{Method: "GET", Path: "users"},
		{Method: "POST", Path: "users"},
		{Method: "DELETE", Path: "users"},
	}

	getRoute, _, getErr := s.router.Find(routes, "GET", []string{"users"})
	postRoute, _, postErr := s.router.Find(routes, "POST", []string{"users"})
	deleteRoute, _, deleteErr := s.router.Find(routes, "DELETE", []string{"users"})

	s.T.Expect(getErr).ToBeNil()
	s.T.Expect(getRoute.Method).ToEqual(types.Method("GET"))
	s.T.Expect(postErr).ToBeNil()
	s.T.Expect(postRoute.Method).ToEqual(types.Method("POST"))
	s.T.Expect(deleteErr).ToBeNil()
	s.T.Expect(deleteRoute.Method).ToEqual(types.Method("DELETE"))
}

func (s *RouterSuite) TestFind_RouteCloneDoesNotAffectOriginal() {
	originalMiddleware := &testMiddleware{name: "original"}
	routes := types.Routes{
		{
			Method:      "GET",
			Path:        "users",
			Middlewares: types.Middlewares{originalMiddleware},
		},
	}

	route, _, _ := s.router.Find(routes, "GET", []string{"users"})
	route.Middlewares = append(route.Middlewares, &testMiddleware{name: "added"})

	s.T.Expect(routes[0].Middlewares).ToHaveLength(1)
}
