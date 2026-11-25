package openapi_test

import (
	"strings"
	"testing"

	"github.com/awesome-goose/goose/modules/router"
	"github.com/awesome-goose/goose/openapi"
	"github.com/awesome-goose/goose/openapi/internal/clash"
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

func TestTwoTypesWithOneNameAreAnErrorUnlessOneIsNamed(t *testing.T) {
	_, err := openapi.FromRoutes([]types.Route{
		router.Get("/a/:id", func(d *DocDto) types.Output { return nil }),
		router.Get("/b", func(d *ClashDto) types.Output { return nil }),
	}, openapi.Options{})
	if err == nil || !strings.Contains(err.Error(), `schema "Thing" is two types`) || !strings.Contains(err.Error(), "openapi/internal/clash.Thing") {
		t.Fatalf("want a clash error naming both types, got %v", err)
	}
	d := build(t, []types.Route{
		router.Get("/a/:id", func(d *DocDto) types.Output { return nil }),
		router.Get("/b", func(d *NamedDto) types.Output { return nil }),
	})
	if at(d, "components", "schemas", "ClashThing") == nil || at(d, "components", "schemas", "Thing") == nil {
		t.Errorf("both components: %v", at(d, "components", "schemas"))
	}
}
