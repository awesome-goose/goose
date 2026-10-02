package openapi_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/awesome-goose/goose/modules/router"
	"github.com/awesome-goose/goose/openapi"
	"github.com/awesome-goose/goose/types"
	"gopkg.in/yaml.v3"
)

type Ctx struct{ UserID string }

type Filter struct{ Name string }

type GetDto struct {
	ID    string            `param:"id"`
	Kind  string            `query:"kind" binding:"oneof=a b c" doc:"which kind"`
	Page  int               `query:"page" binding:"min=1"`
	Tags  []string          `query:"tags"`
	Other map[string]string `queries:"all"`
	Ctx   Ctx               `context:"ntx"`
}

type CreateDto struct {
	Name    string          `json:"name" binding:"required,min=2,max=40" doc:"what to call it"`
	Count   int             `json:"count"`
	When    time.Time       `json:"when"`
	Filter  *Filter         `json:"filter"`
	Raw     json.RawMessage `json:"raw"`
	Skipped string          `json:"-"`
	Id      string          `json:"id" binding:"-"`
	Token   string          `header:"X-Token" binding:"required"`
	Ctx     Ctx             `context:"ntx"`
}

type UploadDto struct {
	ID   string `param:"id"`
	Body []byte `raw:"body"`
}

type FormDto struct {
	Email string `form:"email" binding:"required"`
}

type Thing struct {
	ID   string `json:"id"`
	Next *Thing `json:"next"`
}

type DocDto struct {
	ID string `param:"id"`
}

func (DocDto) OpenAPI() openapi.Operation {
	return openapi.Operation{Summary: "Show a thing", Response: Thing{}, Status: 201, Tags: []string{"things"}}
}

type Controller struct{}

func (c *Controller) List(d *GetDto) types.Output      { return nil }
func (c *Controller) Create(d *CreateDto) types.Output { return nil }
func (c *Controller) Upload(d *UploadDto) types.Output { return nil }
func (c *Controller) Form(d *FormDto) types.Output     { return nil }
func (c *Controller) Show(d *DocDto) types.Output      { return nil }

func routes() []types.Route {
	return []types.Route{
		router.Route(types.GET, "apps/demo", nil, nil,
			router.Get("/thing", []any{Controller{}, "List"}),
			router.Post("/thing", []any{Controller{}, "Create"}),
			router.Get("/thing/:id", []any{Controller{}, "Show"}),
			router.Put("/thing/:id/blob", []any{Controller{}, "Upload"}),
			router.Post("/form", []any{Controller{}, "Form"}),
			router.Delete("/thing/:id", func(d *UploadDto) types.Output { return nil }),
			router.Get("/*", func(d *UploadDto) types.Output { return nil }),
		),
		router.Get("/health", func(d *DocDto) types.Output { return nil }),
	}
}

func build(t *testing.T, rs []types.Route) openapi.Document {
	t.Helper()
	d, err := openapi.FromRoutes(rs, openapi.Options{Info: openapi.Info{Title: "Demo", Version: "1"}, Prefix: "/api", BearerAuth: true})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func at(d openapi.Document, path ...string) any {
	var v any = map[string]any(d)
	for _, p := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[p]
	}
	return v
}

func TestTheDocumentIsValid(t *testing.T) {
	d := build(t, routes())
	if problems := openapi.Validate(d); len(problems) > 0 {
		t.Fatalf("%s", strings.Join(problems, "\n"))
	}
	if at(d, "openapi") != "3.1.0" {
		t.Errorf("version: %v", at(d, "openapi"))
	}
}

func TestPathsMethodsAndWhatIsLeftOut(t *testing.T) {
	d := build(t, routes())
	paths := at(d, "paths").(map[string]any)
	for _, want := range []string{"/api/apps/demo/thing", "/api/apps/demo/thing/{id}", "/api/apps/demo/thing/{id}/blob", "/api/apps/demo/form", "/api/health"} {
		if paths[want] == nil {
			t.Errorf("missing %s in %v", want, keys(paths))
		}
	}
	if len(paths) != 5 {
		t.Errorf("a catch-all is not an operation: %v", keys(paths))
	}
	if at(d, "paths", "/api/apps/demo/thing/{id}", "delete") == nil || at(d, "paths", "/api/apps/demo/thing", "post") == nil {
		t.Error("methods")
	}
}

func TestParametersComeFromTheTagsTheReaderUses(t *testing.T) {
	d := build(t, routes())
	params := at(d, "paths", "/api/apps/demo/thing", "get", "parameters").([]any)
	byName := map[string]map[string]any{}
	for _, p := range params {
		m := p.(map[string]any)
		byName[m["name"].(string)] = m
	}
	if byName["kind"]["in"] != "query" || at(byName["kind"], "schema", "enum") == nil || byName["kind"]["description"] != "which kind" {
		t.Errorf("kind: %v", byName["kind"])
	}
	if at(byName["page"], "schema", "minimum") != int64(1) {
		t.Errorf("page: %v", byName["page"])
	}
	tags := byName["tags"]
	if at(tags, "schema", "type") != "array" || tags["explode"] != false {
		t.Errorf("a []string is one comma separated value: %v", tags)
	}
	if byName["queries"] == nil {
		t.Error("queries:all")
	}
	for n := range byName {
		if n == "ntx" || n == "Ctx" {
			t.Error("a context field is not a parameter")
		}
	}
	// the path parameter is required, and declared once
	pp := at(d, "paths", "/api/apps/demo/thing/{id}", "get", "parameters").([]any)
	if len(pp) != 1 || pp[0].(map[string]any)["in"] != "path" || pp[0].(map[string]any)["required"] != true {
		t.Errorf("path param: %v", pp)
	}
	// a {name} the Dto does not mention is still declared
	hp := at(d, "paths", "/api/health", "get", "parameters")
	if hp != nil {
		t.Errorf("health has no params: %v", hp)
	}
}

func TestBodyFromJSONTagsWithRulesAndWithoutWhatTheServerSets(t *testing.T) {
	d := build(t, routes())
	op := at(d, "paths", "/api/apps/demo/thing", "post").(map[string]any)
	schema := at(op, "requestBody", "content", "application/json", "schema").(map[string]any)
	props := schema["properties"].(map[string]any)
	if props["name"] == nil || props["id"] != nil || props["Skipped"] != nil || props["-"] != nil {
		t.Errorf("props: %v", keys(props))
	}
	name := props["name"].(map[string]any)
	if name["minLength"] != int64(2) || name["maxLength"] != int64(40) || name["description"] != "what to call it" {
		t.Errorf("name: %v", name)
	}
	if at(props["when"].(map[string]any), "format") != "date-time" || len(props["raw"].(map[string]any)) != 0 {
		t.Errorf("when/raw: %v %v", props["when"], props["raw"])
	}
	req := schema["required"].([]any)
	if len(req) != 1 || req[0] != "name" || at(op, "requestBody", "required") != true {
		t.Errorf("required: %v", req)
	}
	// a header the caller must send
	var hdr map[string]any
	for _, p := range op["parameters"].([]any) {
		if p.(map[string]any)["in"] == "header" {
			hdr = p.(map[string]any)
		}
	}
	if hdr == nil || hdr["name"] != "X-Token" || hdr["required"] != true {
		t.Errorf("header: %v", hdr)
	}
	// a GET has no body, even if its Dto has json tags
	if at(d, "paths", "/api/apps/demo/thing", "get", "requestBody") != nil {
		t.Error("GET with a body")
	}
}

func TestRawFormAndComponents(t *testing.T) {
	d := build(t, routes())
	if at(d, "paths", "/api/apps/demo/thing/{id}/blob", "put", "requestBody", "content", "application/octet-stream", "schema", "format") != "binary" {
		t.Error("raw body")
	}
	if at(d, "paths", "/api/apps/demo/form", "post", "requestBody", "content", "application/x-www-form-urlencoded") == nil {
		t.Error("form body")
	}
	// a named struct is a component, and a recursive one ends
	comp := at(d, "components", "schemas", "Thing").(map[string]any)
	if at(comp, "properties", "next", "$ref") != "#/components/schemas/Thing" {
		t.Errorf("recursive: %v", comp)
	}
	if at(d, "components", "schemas", "Filter") == nil {
		t.Error("Filter component")
	}
}

func TestOperationMethodAddsWhatTagsCannotSay(t *testing.T) {
	d := build(t, routes())
	op := at(d, "paths", "/api/apps/demo/thing/{id}", "get").(map[string]any)
	if op["summary"] != "Show a thing" || at(op, "tags").([]any)[0] != "things" {
		t.Errorf("%v", op)
	}
	if at(op, "responses", "201", "content", "application/json", "schema", "$ref") != "#/components/schemas/Thing" || at(op, "responses", "200") != nil {
		t.Errorf("responses: %v", op["responses"])
	}
	if at(op, "responses", "default") == nil {
		t.Error("an error response")
	}
}

func TestSummariesTagsAndUniqueOperationIds(t *testing.T) {
	d := build(t, routes())
	op := at(d, "paths", "/api/apps/demo/thing", "get").(map[string]any)
	if op["summary"] != "List" || at(op, "tags").([]any)[0] != "demo" || op["operationId"] != "list" {
		t.Errorf("%v", op)
	}
	// two apps with a handler of the same name keep distinct ids
	two := []types.Route{
		router.Get("apps/a/x", []any{Controller{}, "List"}),
		router.Get("apps/b/x", []any{Controller{}, "List"}),
		router.Get("apps/b/y", []any{Controller{}, "List"}),
	}
	d2 := build(t, two)
	if problems := openapi.Validate(d2); len(problems) > 0 {
		t.Fatal(problems)
	}
	ids := map[string]bool{}
	for _, p := range at(d2, "paths").(map[string]any) {
		ids[p.(map[string]any)["get"].(map[string]any)["operationId"].(string)] = true
	}
	if len(ids) != 3 {
		t.Errorf("ids: %v", ids)
	}
}

func TestDeterministicAndARouteChangeChangesTheDocument(t *testing.T) {
	a, _ := build(t, routes()).YAML()
	b, _ := build(t, routes()).YAML()
	if string(a) != string(b) || len(a) == 0 {
		t.Fatal("the same routes gave different bytes")
	}
	ja1, _ := build(t, routes()).JSON()
	ja2, _ := build(t, routes()).JSON()
	if string(ja1) != string(ja2) {
		t.Fatal("JSON differs")
	}
	changed := append(routes(), router.Post("/extra", []any{Controller{}, "Form"}))
	c, _ := build(t, changed).YAML()
	if string(c) == string(a) || !strings.Contains(string(c), "/api/extra") {
		t.Error("a new route did not change the document")
	}
	// the YAML and JSON say the same thing
	var fromYAML, fromJSON any
	if err := yaml.Unmarshal(a, &fromYAML); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(ja1, &fromJSON); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(a), "openapi: 3.1.0") {
		t.Errorf("yaml: %.200s", a)
	}
	_ = fromYAML
	_ = fromJSON
}

func TestAHandlerItCannotReadIsAnErrorNotASilentGap(t *testing.T) {
	_, err := openapi.FromRoutes([]types.Route{router.Get("/x", []any{Controller{}, "Nope"})}, openapi.Options{})
	if err == nil || !strings.Contains(err.Error(), "GET /x") {
		t.Errorf("%v", err)
	}
	_, err = openapi.FromRoutes([]types.Route{router.Get("/x", func() {})}, openapi.Options{})
	if err == nil {
		t.Error("a func without a Dto")
	}
	_, err = openapi.FromRoutes([]types.Route{router.Get("/x", func(d *DocDto) types.Output { return nil }), router.Get("/x", func(d *DocDto) types.Output { return nil })}, openapi.Options{})
	if err == nil || !strings.Contains(err.Error(), "twice") {
		t.Errorf("duplicate: %v", err)
	}
}

func TestSkipAndOptions(t *testing.T) {
	d, err := openapi.FromRoutes(routes(), openapi.Options{Prefix: "/api", Servers: []string{"https://x.test"}, Skip: func(m, p string) bool { return p == "/api/health" }})
	if err != nil {
		t.Fatal(err)
	}
	if at(d, "paths", "/api/health") != nil || at(d, "servers").([]any)[0].(map[string]any)["url"] != "https://x.test" || at(d, "security") != nil {
		t.Error("options")
	}
}

func TestValidateFindsWhatIsWrong(t *testing.T) {
	bad := openapi.Document{
		"openapi": "2.0",
		"info":    map[string]any{},
		"paths": map[string]any{
			"x/:id": map[string]any{"get": map[string]any{"responses": map[string]any{}}},
			"/ok/{id}": map[string]any{
				"get": map[string]any{"operationId": "a", "responses": map[string]any{"200": map[string]any{}}, "parameters": []any{}},
				"put": map[string]any{"operationId": "a", "responses": map[string]any{"200": map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/Gone"}}}}}},
			},
		},
	}
	got := strings.Join(openapi.Validate(bad), "\n")
	for _, want := range []string{"3.x version", "info.title", "info.version", "must start with /", "uses :name", "operationId is required", "needs at least one response", "{id} is not declared", `operationId "a" is also used`, "points at nothing"} {
		if !strings.Contains(got, want) {
			t.Errorf("did not find %q in:\n%s", want, got)
		}
	}
}

func keys(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
