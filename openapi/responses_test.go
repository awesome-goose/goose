package openapi_test

import (
	"strings"
	"testing"

	"github.com/awesome-goose/goose/modules/router"
	"github.com/awesome-goose/goose/openapi"
	"github.com/awesome-goose/goose/openapi/internal/clash"
	"github.com/awesome-goose/goose/openapi/internal/clash/more"
	otherclash "github.com/awesome-goose/goose/openapi/internal/other/clash"
	"github.com/awesome-goose/goose/types"
)

type RegisterDto struct{}

func (RegisterDto) OpenAPI() openapi.Operation {
	return openapi.Operation{Summary: "Register", Response: Thing{}, Status: 201, Responses: map[int]any{200: nil, 202: Filter{}}}
}

type FileDto struct {
	ID string `param:"id"`
}

func (FileDto) OpenAPI() openapi.Operation {
	return openapi.Operation{Summary: "Download", ContentType: "application/octet-stream", Response: Thing{}}
}

type BadStatusDto struct{}

func (BadStatusDto) OpenAPI() openapi.Operation {
	return openapi.Operation{Responses: map[int]any{404: nil}}
}

type ClashDto struct{}

func (ClashDto) OpenAPI() openapi.Operation { return openapi.Operation{Response: clash.Thing{}} }

type NamedDto struct{}

func (NamedDto) OpenAPI() openapi.Operation { return openapi.Operation{Response: clash.Named{}} }

func TestMoreSuccessStatusesAndBinaryBodies(t *testing.T) {
	d := build(t, []types.Route{router.Route(types.GET, "apps/demo", nil, nil,
		router.Post("/owners", func(d *RegisterDto) types.Output { return nil }),
		router.Get("/files/:id", func(d *FileDto) types.Output { return nil }),
	)})
	op := at(d, "paths", "/api/apps/demo/owners", "post").(map[string]any)
	for code, ref := range map[string]string{"201": "Thing", "200": "Thing", "202": "Filter"} {
		if got := at(op, "responses", code, "content", "application/json", "schema", "$ref"); got != "#/components/schemas/"+ref {
			t.Errorf("%s: %v", code, got)
		}
	}
	file := at(d, "paths", "/api/apps/demo/files/{id}", "get", "responses", "200", "content").(map[string]any)
	if len(file) != 1 || at(file, "application/octet-stream", "schema", "format") != "binary" {
		t.Errorf("binary body: %v", file)
	}
}

func TestANonSuccessExtraStatusIsAnError(t *testing.T) {
	_, err := openapi.FromRoutes([]types.Route{router.Get("/x", func(d *BadStatusDto) types.Output { return nil })}, openapi.Options{})
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("want an error naming 404, got %v", err)
	}
}

type MoreDto struct{}

func (MoreDto) OpenAPI() openapi.Operation { return openapi.Operation{Response: more.Thing{}} }

type AgainDto struct{}

func (AgainDto) OpenAPI() openapi.Operation { return openapi.Operation{Response: otherclash.Thing{}} }

func clashRoutes(last types.Route) []types.Route {
	return []types.Route{
		router.Get("/a/:id", func(d *DocDto) types.Output { return nil }),
		router.Get("/b", func(d *ClashDto) types.Output { return nil }),
		router.Get("/c", func(d *ClashDto) types.Output { return nil }),
		last,
	}
}

func TestEveryTypeWithATakenNameGetsItsOwnSchema(t *testing.T) {
	d := build(t, clashRoutes(router.Get("/d", func(d *MoreDto) types.Output { return nil })))
	d2 := build(t, append(clashRoutes(router.Get("/d", func(d *MoreDto) types.Output { return nil })), router.Get("/e", func(d *AgainDto) types.Output { return nil })))
	schemas := at(d2, "components", "schemas").(map[string]any)
	refs := map[string]string{}
	for path, want := range map[string]string{"/api/a/{id}": "Thing", "/api/b": "ClashThing", "/api/d": "MoreThing", "/api/e": "OtherClashThing"} {
		ref, _ := at(d2, "paths", path, "get", "responses", "200", "content", "application/json", "schema", "$ref").(string)
		if path == "/api/a/{id}" {
			ref, _ = at(d2, "paths", path, "get", "responses", "201", "content", "application/json", "schema", "$ref").(string)
		}
		name := strings.TrimPrefix(ref, "#/components/schemas/")
		if schemas[name] == nil || refs[name] != "" {
			t.Fatalf("%s: %q (taken by %s)", path, ref, refs[name])
		}
		refs[name] = path
		if name != want {
			t.Errorf("%s: %s, want %s", path, name, want)
		}
	}
	if at(d, "components", "schemas", "MoreThing") == nil {
		t.Errorf("three types: %v", at(d, "components", "schemas"))
	}
}

func TestStrictNamesMakesAClashAnErrorUnlessOneIsNamed(t *testing.T) {
	_, err := openapi.FromRoutes(clashRoutes(router.Get("/d", func(d *ClashDto) types.Output { return nil })), openapi.Options{StrictNames: true})
	if err == nil || !strings.Contains(err.Error(), `schema "Thing" is two types`) || !strings.Contains(err.Error(), "openapi/internal/clash.Thing") {
		t.Fatalf("want a clash error naming both types, got %v", err)
	}
	if n := strings.Count(err.Error(), "is two types"); n != 1 {
		t.Errorf("a clash is reported once, got %d: %v", n, err)
	}
	d, err := openapi.FromRoutes([]types.Route{
		router.Get("/a/:id", func(d *DocDto) types.Output { return nil }),
		router.Get("/b", func(d *NamedDto) types.Output { return nil }),
	}, openapi.Options{StrictNames: true})
	if err != nil {
		t.Fatal(err)
	}
	if at(d, "components", "schemas", "ClashThing") == nil || at(d, "components", "schemas", "Thing") == nil {
		t.Errorf("both components: %v", at(d, "components", "schemas"))
	}
}
