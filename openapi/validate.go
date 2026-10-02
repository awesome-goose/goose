package openapi

import (
	"fmt"
	"sort"
	"strings"
)

// Validate checks a document against the parts of OpenAPI 3.1 this package writes:
// the version and info, that every path starts with "/" and declares each {name} it
// uses (and no other path parameter), that operation ids are unique, that every
// operation has a response, and that every $ref points at a schema that exists. It
// returns every problem, sorted, or nil.
func Validate(d Document) []string {
	var out []string
	add := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }

	if v, _ := d["openapi"].(string); !strings.HasPrefix(v, "3.") {
		add("openapi: want a 3.x version, got %q", v)
	}
	info, _ := d["info"].(map[string]any)
	if s, _ := info["title"].(string); s == "" {
		add("info.title is required")
	}
	if s, _ := info["version"].(string); s == "" {
		add("info.version is required")
	}
	schemas := map[string]bool{}
	if c, ok := d["components"].(map[string]any); ok {
		if s, ok := c["schemas"].(map[string]any); ok {
			for n := range s {
				schemas[n] = true
			}
		}
	}
	paths, _ := d["paths"].(map[string]any)
	ids := map[string]string{}
	for p, v := range paths {
		if !strings.HasPrefix(p, "/") {
			add("path %q must start with /", p)
		}
		if strings.Contains(p, ":") {
			add("path %q uses :name; write {name}", p)
		}
		want := map[string]bool{}
		for _, m := range param.FindAllStringSubmatch(strings.ReplaceAll(strings.ReplaceAll(p, "{", ":"), "}", ""), -1) {
			want[m[1]] = true
		}
		ops, _ := v.(map[string]any)
		for method, o := range ops {
			op, _ := o.(map[string]any)
			where := strings.ToUpper(method) + " " + p
			if id, _ := op["operationId"].(string); id != "" {
				if prev, dup := ids[id]; dup {
					add("%s: operationId %q is also used by %s", where, id, prev)
				}
				ids[id] = where
			} else {
				add("%s: operationId is required", where)
			}
			if r, _ := op["responses"].(map[string]any); len(r) == 0 {
				add("%s: needs at least one response", where)
			}
			have := map[string]bool{}
			for _, pr := range asList(op["parameters"]) {
				pm, _ := pr.(map[string]any)
				if pm["in"] == "path" {
					have[pm["name"].(string)] = true
					if pm["required"] != true {
						add("%s: path parameter %q must be required", where, pm["name"])
					}
				}
				if pm["name"] == nil || pm["in"] == nil || pm["schema"] == nil {
					add("%s: a parameter needs name, in and schema", where)
				}
			}
			for n := range want {
				if !have[n] {
					add("%s: path parameter {%s} is not declared", where, n)
				}
			}
			for n := range have {
				if !want[n] {
					add("%s: path parameter %q is not in the path", where, n)
				}
			}
			if rb, ok := op["requestBody"].(map[string]any); ok && rb["content"] == nil {
				add("%s: requestBody needs content", where)
			}
			checkRefs(op, schemas, where, &out)
		}
	}
	if c, ok := d["components"].(map[string]any); ok {
		checkRefs(c, schemas, "components", &out)
	}
	sort.Strings(out)
	return out
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}

func checkRefs(v any, schemas map[string]bool, where string, out *[]string) {
	switch x := v.(type) {
	case map[string]any:
		if ref, ok := x["$ref"].(string); ok {
			name := strings.TrimPrefix(ref, "#/components/schemas/")
			if name == ref || !schemas[name] {
				*out = append(*out, fmt.Sprintf("%s: $ref %q points at nothing", where, ref))
			}
		}
		for _, c := range x {
			checkRefs(c, schemas, where, out)
		}
	case []any:
		for _, c := range x {
			checkRefs(c, schemas, where, out)
		}
	}
}
