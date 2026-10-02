package openapi

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	timeType    = reflect.TypeOf(time.Time{})
	rawJSONType = reflect.TypeOf(json.RawMessage{})
	durationTyp = reflect.TypeOf(time.Duration(0))
)

// fields reads the input struct of a handler: where each value comes from and what
// it must look like. The body is described only for methods that carry one.
func (b *builder) fields(in reflect.Type, method string) (params []map[string]any, body map[string]any, bodyRequired bool) {
	hasBody := method != "get"
	jsonProps := map[string]any{}
	var jsonReq []string
	formProps := map[string]any{}
	var formReq []string

	for i := 0; i < in.NumField(); i++ {
		f := in.Field(i)
		if !f.IsExported() {
			continue
		}
		// binding:"-" says a caller cannot set it (ids, timestamps): it is not part of a body
		if f.Tag.Get("binding") == "-" && (f.Tag.Get("json") != "" || f.Tag.Get("form") != "") {
			continue
		}
		if f.Tag.Get("context") != "" {
			continue // the server fills it
		}
		req, rules := binding(f.Tag.Get("binding"))
		desc := f.Tag.Get("doc")

		schema := func() map[string]any {
			s := b.schemaOf(f.Type)
			s = withRules(s, rules)
			if desc != "" {
				s = describe(s, desc)
			}
			return s
		}

		switch {
		case f.Tag.Get("raw") == "body":
			body = map[string]any{"content": map[string]any{"application/octet-stream": map[string]any{"schema": map[string]any{"type": "string", "format": "binary"}}}}
			bodyRequired = req
		case f.Tag.Get("queries") == "all":
			params = append(params, map[string]any{"name": "queries", "in": "query", "style": "form", "explode": true,
				"description": "Any other query parameters, as name=value pairs",
				"schema":      map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}}})
		case f.Tag.Get("header") != "":
			params = append(params, parameter(f.Tag.Get("header"), "header", req, schema(), desc))
		case f.Tag.Get("param") != "":
			params = append(params, parameter(f.Tag.Get("param"), "path", true, schema(), desc))
		case f.Tag.Get("query") != "":
			s := schema()
			p := parameter(f.Tag.Get("query"), "query", req, s, desc)
			if s["type"] == "array" {
				p["style"], p["explode"] = "form", false // the reader splits one value on commas
			}
			params = append(params, p)
		case f.Tag.Get("form") != "":
			if hasBody {
				name := f.Tag.Get("form")
				formProps[name] = schema()
				if req {
					formReq = append(formReq, name)
				}
			}
		case f.Tag.Get("json") != "":
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "-" || !hasBody {
				continue
			}
			if strings.HasSuffix(f.Tag.Get("json"), ",merge") {
				// the whole body fills one field: its own properties are the body's
				if st := deref(f.Type); st.Kind() == reflect.Struct {
					for k, v := range b.objectOf(st) {
						if k == "properties" {
							for pk, pv := range v.(map[string]any) {
								jsonProps[pk] = pv
							}
						}
					}
				}
				continue
			}
			jsonProps[name] = schema()
			if req {
				jsonReq = append(jsonReq, name)
			}
		}
	}
	if len(jsonProps) > 0 {
		obj := map[string]any{"type": "object", "properties": jsonProps}
		if len(jsonReq) > 0 {
			sort.Strings(jsonReq)
			obj["required"] = toAny(jsonReq)
			bodyRequired = true
		}
		body = map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": obj}}}
	}
	if len(formProps) > 0 {
		obj := map[string]any{"type": "object", "properties": formProps}
		if len(formReq) > 0 {
			sort.Strings(formReq)
			obj["required"] = toAny(formReq)
			bodyRequired = true
		}
		content := map[string]any{"application/x-www-form-urlencoded": map[string]any{"schema": obj}}
		if body != nil {
			for k, v := range body["content"].(map[string]any) {
				content[k] = v
			}
		}
		body = map[string]any{"content": content}
	}
	return params, body, bodyRequired
}

func parameter(name, in string, required bool, schema map[string]any, desc string) map[string]any {
	p := map[string]any{"name": name, "in": in, "schema": schema}
	if required || in == "path" {
		p["required"] = true
	}
	if desc != "" {
		p["description"] = desc
	}
	return p
}

func describe(s map[string]any, desc string) map[string]any {
	if _, isRef := s["$ref"]; isRef {
		return map[string]any{"allOf": []any{s}, "description": desc}
	}
	s["description"] = desc
	return s
}

type rule struct {
	kind string
	arg  string
}

// binding reads a `binding:"required,min=1,max=9,oneof=a b"` tag.
func binding(tag string) (required bool, rules []rule) {
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == "" || part == "-":
		case part == "required":
			required = true
		case strings.Contains(part, "="):
			k, v, _ := strings.Cut(part, "=")
			rules = append(rules, rule{k, v})
		}
	}
	return
}

func withRules(s map[string]any, rules []rule) map[string]any {
	if len(rules) == 0 {
		return s
	}
	if _, isRef := s["$ref"]; isRef {
		return s
	}
	for _, r := range rules {
		switch r.kind {
		case "oneof":
			var vals []any
			for _, v := range strings.Fields(r.arg) {
				if s["type"] == "integer" || s["type"] == "number" {
					if n, err := strconv.ParseFloat(v, 64); err == nil {
						vals = append(vals, n)
						continue
					}
				}
				vals = append(vals, v)
			}
			s["enum"] = vals
		case "min", "max", "len":
			n, err := strconv.ParseFloat(r.arg, 64)
			if err != nil {
				continue
			}
			switch s["type"] {
			case "string":
				setBound(s, map[string]string{"min": "minLength", "max": "maxLength", "len": "minLength"}[r.kind], n)
				if r.kind == "len" {
					s["maxLength"] = n
				}
			case "array":
				setBound(s, map[string]string{"min": "minItems", "max": "maxItems", "len": "minItems"}[r.kind], n)
				if r.kind == "len" {
					s["maxItems"] = n
				}
			case "integer", "number":
				setBound(s, map[string]string{"min": "minimum", "max": "maximum", "len": "minimum"}[r.kind], n)
			}
		}
	}
	return s
}

func setBound(s map[string]any, key string, n float64) {
	if key == "" {
		return
	}
	if n == float64(int64(n)) {
		s[key] = int64(n)
	} else {
		s[key] = n
	}
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// schemaOf is the JSON Schema of a Go type. A named struct becomes a component and a $ref, so
// a type used twice is described once and a recursive type ends.
func (b *builder) schemaOf(t reflect.Type) map[string]any {
	t = deref(t)
	switch {
	case t == timeType:
		return map[string]any{"type": "string", "format": "date-time"}
	case t == rawJSONType:
		return map[string]any{}
	case t == durationTyp:
		return map[string]any{"type": "integer", "format": "int64", "description": "nanoseconds"}
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return map[string]any{"type": "integer"}
	case reflect.Int64, reflect.Uint64:
		return map[string]any{"type": "integer", "format": "int64"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return map[string]any{"type": "string", "format": "byte"}
		}
		return map[string]any{"type": "array", "items": b.schemaOf(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": b.schemaOf(t.Elem())}
	case reflect.Struct:
		return b.ref(t)
	}
	return map[string]any{} // interfaces and anything else: any JSON
}

// ref registers a struct as a component and points at it.
func (b *builder) ref(t reflect.Type) map[string]any {
	name := t.Name()
	if name == "" {
		return b.objectOf(t)
	}
	if prev, ok := b.seen[name]; ok && prev != t {
		// two different types with one name: qualify by the package's last element
		pkg := t.PkgPath()
		name = upperFirst(sanitize(pkg[strings.LastIndex(pkg, "/")+1:])) + name
	}
	if _, ok := b.seen[name]; !ok {
		b.seen[name] = t
		b.schemas[name] = map[string]any{} // placeholder so a recursive type stops here
		b.schemas[name] = b.objectOf(t)
	}
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

// objectOf is the object schema of a struct by its json tags.
func (b *builder) objectOf(t reflect.Type) map[string]any {
	props := map[string]any{}
	var req []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "-" {
			continue
		}
		if f.Anonymous && name == "" {
			if st := deref(f.Type); st.Kind() == reflect.Struct {
				for k, v := range b.objectOf(st) {
					if k == "properties" {
						for pk, pv := range v.(map[string]any) {
							props[pk] = pv
						}
					}
				}
				continue
			}
		}
		if name == "" {
			name = f.Name
		}
		s := b.schemaOf(f.Type)
		if d := f.Tag.Get("doc"); d != "" {
			s = describe(s, d)
		}
		props[name] = s
		if r, _ := binding(f.Tag.Get("binding")); r && !strings.Contains(tag, "omitempty") {
			req = append(req, name)
		}
	}
	out := map[string]any{"type": "object", "properties": props}
	if len(req) > 0 {
		sort.Strings(req)
		out["required"] = toAny(req)
	}
	return out
}
