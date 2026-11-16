package openapi_test

import (
	"testing"

	"github.com/awesome-goose/goose/modules/router"
	"github.com/awesome-goose/goose/openapi"
	"github.com/awesome-goose/goose/types"
)

// The served document names operations as the committed copy does (Yan BUGS #6 / U25):
// by method and path, whatever order the routes registered in.
func TestPathOperationID(t *testing.T) {
	id := openapi.PathOperationID("api", "apps")
	for in, want := range map[[2]string]string{
		{"get", "/api/apps/social/posts/{id}"}:                  "getSocialPostsById",
		{"post", "/api/apps/capital/{type}/charge/{providerId}"}: "postCapitalByTypeChargeByProviderid",
		{"get", "/api/apps/polylog/event-log/dashboard"}:         "getPolylogEventLogDashboard",
		{"post", "/api/apps/identity/2fa/verify"}:                "postIdentity2FaVerify",
		{"get", "/api/client/config"}:                            "getClientConfig",
	} {
		if got := id(in[0], in[1], "Whatever"); got != want {
			t.Errorf("%s %s = %q, want %q", in[0], in[1], got, want)
		}
	}
}

func TestOperationIDOptionNamesEveryOperationAndRefusesAClash(t *testing.T) {
	d, err := openapi.FromRoutes(routes(), openapi.Options{Prefix: "/api", OperationID: openapi.PathOperationID("api", "apps")})
	if err != nil {
		t.Fatal(err)
	}
	if got := at(d, "paths", "/api/apps/demo/thing/{id}", "get", "operationId"); got != "getDemoThingById" {
		t.Errorf("operationId = %v", got)
	}
	same := func(string, string, string) string { return "same" }
	if _, err := openapi.FromRoutes(routes(), openapi.Options{OperationID: same}); err == nil {
		t.Error("two operations with one id must be an error")
	}
}

type Entity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type sharedDto struct {
	Body map[string]any `json:",merge"`
}

// A generic controller documents the handlers its shared Dtos cannot.
type Generic[E any] struct{}

func (g *Generic[E]) List(*sharedDto) types.Output { return nil }
func (g *Generic[E]) Get(*sharedDto) types.Output  { return nil }
func (g *Generic[E]) OpenAPIFor(handler string) openapi.Operation {
	switch handler {
	case "List":
		return openapi.Operation{Response: []E{}}
	case "Get":
		return openapi.Operation{Response: *new(E), Summary: "from the controller"}
	}
	return openapi.Operation{}
}

type EntityController struct{ Generic[Entity] }

type attrsDto struct {
	Type string `param:"type"`
}

type Attributes struct {
	Values map[string]string `json:"values" binding:"required"`
}

func (attrsDto) OpenAPI() openapi.Operation {
	return openapi.Operation{Request: Attributes{}, Response: openapi.Page[Entity]{}}
}

func TestControllerDocsRequestBodyAndPage(t *testing.T) {
	rs := []types.Route{
		router.Get("/entities", []any{EntityController{}, "List"}),
		router.Get("/entities/:id", []any{EntityController{}, "Get"}),
		router.Patch("/attributes/:type", func(*attrsDto) types.Output { return nil }),
	}
	d, err := openapi.FromRoutes(rs, openapi.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if at(d, "paths", "/entities", "get", "responses", "200", "content", "application/json", "schema", "items", "$ref") != "#/components/schemas/Entity" {
		t.Errorf("List response: %v", at(d, "paths", "/entities", "get", "responses", "200"))
	}
	if at(d, "paths", "/entities/{id}", "get", "summary") != "from the controller" {
		t.Errorf("controller summary not used")
	}
	if at(d, "paths", "/attributes/{type}", "patch", "requestBody", "content", "application/json", "schema", "$ref") != "#/components/schemas/Attributes" {
		t.Errorf("request body: %v", at(d, "paths", "/attributes/{type}", "patch", "requestBody"))
	}
	if at(d, "paths", "/attributes/{type}", "patch", "responses", "200", "content", "application/json", "schema", "$ref") != "#/components/schemas/PageEntity" {
		t.Errorf("page response")
	}
	page := at(d, "components", "schemas", "PageEntity", "properties").(map[string]any)
	if _, ok := page["items"]; !ok {
		t.Errorf("Page has no items: %v", page)
	}
	if _, ok := page["next"]; !ok {
		t.Errorf("Page has no next: %v", page)
	}
	if problems := openapi.Validate(d); len(problems) > 0 {
		t.Errorf("valid: %v", problems)
	}
}
