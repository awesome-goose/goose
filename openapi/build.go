package openapi

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/awesome-goose/goose/types"
)

// Info is the document's title block.
type Info struct {
	Title       string `json:"title" yaml:"title"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Options shapes what Build writes.
type Options struct {
	Info Info
	// Servers are base URLs, for example "https://api.example.com/api".
	Servers []string
	// Prefix is put in front of every path (the platform's API prefix, "/api").
	Prefix string
	// BearerAuth adds a bearer security scheme and applies it to every operation.
	BearerAuth bool
	// Skip leaves a route out (health checks, internal routes).
	Skip func(method, path string) bool
}

// Operation lets a Dto say more about its route than the tags can.
type Operation struct {
	Summary     string
	Description string
	// Response is a value whose type is the success body; its json tags are the schema.
	Response any
	// Status is the success status; 200 when 0.
	Status int
	// Tags replace the tag taken from the path.
	Tags []string
	// Deprecated marks the operation.
	Deprecated bool
}

// Documented is implemented by a Dto (by value) that wants to add to its operation.
type Documented interface{ OpenAPI() Operation }

// Document is an OpenAPI 3.1 document. It is plain maps and slices, so it encodes
// to JSON or YAML with sorted keys and is easy to compare and to patch.
type Document map[string]any

type builder struct {
	opts    Options
	paths   map[string]map[string]any
	schemas map[string]any
	// seen names the Go type behind each schema name, to give a clash a distinct name
	seen map[string]reflect.Type
	ids  map[string]int
	tags map[string]bool
	err  []string
}

// Build documents the routes the kernel booted with. Call it after Start has returned.
func Build(k types.Kernel, o Options) (Document, error) { return FromRoutes(k.Routes(), o) }

// FromRoutes documents a route table. It needs no running server: handlers are read by type.
func FromRoutes(routes []types.Route, o Options) (Document, error) {
	b := &builder{opts: o, paths: map[string]map[string]any{}, schemas: map[string]any{}, seen: map[string]reflect.Type{}, ids: map[string]int{}, tags: map[string]bool{}}
	b.walk(routes, "")
	if len(b.err) > 0 {
		sort.Strings(b.err)
		return nil, fmt.Errorf("openapi: %s", strings.Join(b.err, "; "))
	}
	return b.document(), nil
}

func (b *builder) document() Document {
	info := b.opts.Info
	if info.Title == "" {
		info.Title = "API"
	}
	if info.Version == "" {
		info.Version = "0.0.0"
	}
	i := map[string]any{"title": info.Title, "version": info.Version}
	if info.Description != "" {
		i["description"] = info.Description
	}
	doc := Document{"openapi": "3.1.0", "info": i}
	if len(b.opts.Servers) > 0 {
		var s []any
		for _, u := range b.opts.Servers {
			s = append(s, map[string]any{"url": u})
		}
		doc["servers"] = s
	}
	paths := map[string]any{}
	for p, ops := range b.paths {
		paths[p] = ops
	}
	doc["paths"] = paths
	comps := map[string]any{}
	if len(b.schemas) > 0 {
		comps["schemas"] = b.schemas
	}
	if b.opts.BearerAuth {
		comps["securitySchemes"] = map[string]any{"bearerAuth": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "JWT"}}
		doc["security"] = []any{map[string]any{"bearerAuth": []any{}}}
	}
	if len(comps) > 0 {
		doc["components"] = comps
	}
	if len(b.tags) > 0 {
		names := make([]string, 0, len(b.tags))
		for t := range b.tags {
			names = append(names, t)
		}
		sort.Strings(names)
		var tl []any
		for _, n := range names {
			tl = append(tl, map[string]any{"name": n})
		}
		doc["tags"] = tl
	}
	return doc
}

var multiSlash = regexp.MustCompile(`/{2,}`)

func join(parent, child string) string {
	c := strings.Trim(child, "/")
	p := strings.TrimRight(parent, "/")
	if c == "" {
		if p == "" {
			return "/"
		}
		return p
	}
	return multiSlash.ReplaceAllString(p+"/"+c, "/")
}

func (b *builder) walk(routes []types.Route, parent string) {
	for _, r := range routes {
		path := join(parent, r.Path)
		if strings.Contains(r.Path, "*") {
			continue // a catch-all is not an operation
		}
		if r.Handler != nil {
			b.operation(r, path)
		}
		if len(r.Children) > 0 {
			b.walk(r.Children, path)
		}
	}
}

// methodOf names the HTTP method of a route.
func methodOf(m types.Method) string {
	for _, name := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		if m.Is(name) {
			return strings.ToLower(name)
		}
	}
	return ""
}

var camel = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func words(name string) string {
	s := strings.ToLower(camel.ReplaceAllString(name, "$1 $2"))
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// handlerInput finds the input struct type of a handler and a name for it.
func handlerInput(h any) (in reflect.Type, name string, ok bool) {
	switch x := h.(type) {
	case []any:
		if len(x) != 2 {
			return nil, "", false
		}
		method, isStr := x[1].(string)
		if !isStr || x[0] == nil {
			return nil, "", false
		}
		t := reflect.TypeOf(x[0])
		if t.Kind() != reflect.Ptr {
			t = reflect.PointerTo(t)
		}
		m, found := t.MethodByName(method)
		if !found || m.Type.NumIn() != 2 {
			return nil, "", false
		}
		a := m.Type.In(1)
		if a.Kind() != reflect.Ptr || a.Elem().Kind() != reflect.Struct {
			return nil, "", false
		}
		return a.Elem(), method, true
	default:
		t := reflect.TypeOf(h)
		if t == nil || t.Kind() != reflect.Func || t.NumIn() != 1 || t.In(0).Kind() != reflect.Ptr || t.In(0).Elem().Kind() != reflect.Struct {
			return nil, "", false
		}
		return t.In(0).Elem(), "", true
	}
}

func (b *builder) operation(r types.Route, path string) {
	method := methodOf(r.Method)
	if method == "" {
		return // a CLI route
	}
	full := join(b.opts.Prefix, path)
	if b.opts.Skip != nil && b.opts.Skip(strings.ToUpper(method), full) {
		return
	}
	in, handlerName, ok := handlerInput(r.Handler)
	if !ok {
		b.err = append(b.err, fmt.Sprintf("%s %s: the handler is not func(*Dto) or a controller method", strings.ToUpper(method), full))
		return
	}

	op := map[string]any{}
	var extra Operation
	if d, isDoc := reflect.New(in).Elem().Interface().(Documented); isDoc {
		extra = d.OpenAPI()
	}

	params, body, required := b.fields(in, method)
	// every {name} of the path is a path parameter, declared or not
	templ := templated(full)
	inPath := map[string]bool{}
	for _, name := range pathParams(full) {
		inPath[name] = true
	}
	declared := map[string]bool{}
	kept := params[:0]
	for _, p := range params {
		if p["in"] == "path" {
			// the reader fills a param field only when the path has that segment
			if !inPath[p["name"].(string)] {
				continue
			}
			declared[p["name"].(string)] = true
		}
		kept = append(kept, p)
	}
	params = kept
	for _, name := range pathParams(full) {
		if !declared[name] {
			params = append(params, map[string]any{"name": name, "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
		}
	}
	sort.SliceStable(params, func(i, j int) bool {
		a, c := params[i], params[j]
		if a["in"] != c["in"] {
			return order(a["in"].(string)) < order(c["in"].(string))
		}
		return a["name"].(string) < c["name"].(string)
	})
	if len(params) > 0 {
		list := make([]any, len(params))
		for i, p := range params {
			list[i] = p
		}
		op["parameters"] = list
	}
	if body != nil {
		op["requestBody"] = body
		if required {
			body["required"] = true
		}
	}

	summary := extra.Summary
	if summary == "" && handlerName != "" {
		summary = words(handlerName)
	}
	if summary != "" {
		op["summary"] = summary
	}
	if extra.Description != "" {
		op["description"] = extra.Description
	}
	if extra.Deprecated {
		op["deprecated"] = true
	}
	tags := extra.Tags
	if len(tags) == 0 {
		tags = []string{tagOf(full, b.opts.Prefix)}
	}
	for _, t := range tags {
		b.tags[t] = true
	}
	op["tags"] = toAny(tags)
	op["operationId"] = b.operationID(handlerName, method, templ, tags[0])

	status := extra.Status
	if status == 0 {
		status = 200
	}
	resp := map[string]any{"description": "Success"}
	if extra.Response != nil {
		resp["content"] = map[string]any{"application/json": map[string]any{"schema": b.schemaOf(reflect.TypeOf(extra.Response))}}
	} else {
		resp["content"] = map[string]any{"application/json": map[string]any{"schema": map[string]any{}}}
	}
	op["responses"] = map[string]any{
		strconv.Itoa(status): resp,
		"default":            map[string]any{"description": "An error: a status code and a message", "content": map[string]any{"application/json": map[string]any{"schema": errorSchema}}},
	}

	if b.paths[templ] == nil {
		b.paths[templ] = map[string]any{}
	}
	if _, dup := b.paths[templ][method]; dup {
		b.err = append(b.err, fmt.Sprintf("%s %s is registered twice", strings.ToUpper(method), full))
		return
	}
	b.paths[templ][method] = op
}

var errorSchema = map[string]any{"type": "object", "properties": map[string]any{
	"status":  map[string]any{"type": "string"},
	"title":   map[string]any{"type": "string"},
	"message": map[string]any{"type": "string"},
	"data":    map[string]any{},
}}

func order(in string) int {
	switch in {
	case "path":
		return 0
	case "query":
		return 1
	}
	return 2
}

func toAny(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

var param = regexp.MustCompile(`:([A-Za-z0-9_]+)`)

// templated turns /spec/:id into /spec/{id}.
func templated(path string) string { return param.ReplaceAllString(path, "{$1}") }

func pathParams(path string) []string {
	var out []string
	for _, m := range param.FindAllStringSubmatch(path, -1) {
		out = append(out, m[1])
	}
	return out
}

// tagOf groups operations: apps/<name>/... by the app, anything else by its first segment.
func tagOf(path, prefix string) string {
	p := strings.Trim(strings.TrimPrefix(path, strings.TrimRight(prefix, "/")), "/")
	segs := strings.Split(p, "/")
	if len(segs) >= 2 && segs[0] == "apps" {
		return segs[1]
	}
	if segs[0] == "" {
		return "default"
	}
	return segs[0]
}

// operationID is the handler's name, made unique by the tag when two apps share one, and by
// the method and path when that is not enough; a func handler has no name, so it is method plus path.
func (b *builder) operationID(handler, method, path, tag string) string {
	base := lowerFirst(handler)
	if base == "" {
		base = method + "_" + sanitize(path)
	}
	id := base
	if b.ids[id] > 0 {
		id = tag + upperFirst(base)
	}
	if b.ids[id] > 0 {
		id = id + "_" + method + "_" + sanitize(path)
	}
	b.ids[id]++
	return id
}

func sanitize(path string) string {
	return strings.Trim(regexp.MustCompile(`[^A-Za-z0-9]+`).ReplaceAllString(path, "_"), "_")
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
