package input

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/awesome-goose/goose/errors"
	"github.com/awesome-goose/goose/types"
	"github.com/awesome-goose/goose/utils/props"
)

type Input struct {
	context types.Context
}

func NewInput(context types.Context) *Input {
	return &Input{context}
}

func (i *Input) Populate(payload any) error {
	v := reflect.ValueOf(payload)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.ErrPayloadMustBeNonNilPointer
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return errors.ErrPayloadMustBePointerToStruct
	}

	t := v.Type()

	// Prepare data sources
	headers := i.context.Request().Headers()
	queries := i.context.Request().Queries()
	params := i.context.Request().Params()

	// Parse body for JSON and form data (lazy, cached)
	var jsonData props.Props
	var formData url.Values
	var bodyParsed bool

	parseBody := func() error {
		if bodyParsed {
			return nil
		}
		bodyParsed = true

		body, err := i.context.Request().Body()
		if err != nil {
			return err
		}
		if len(body) == 0 {
			return nil
		}

		// Try JSON first. Numbers are kept as written (json.Number) so an
		// integer above 2^53 (money in minor units, a chain amount) reaches an
		// int64, uint64, string or struct field exactly instead of rounded
		// through float64.
		if decodeJSONNumbers(body, &jsonData) == nil {
			return nil
		}
		jsonData = nil

		// Try form data
		formData, _ = url.ParseQuery(string(body))
		return nil
	}

	var missing []string
	for idx := 0; idx < t.NumField(); idx++ {
		field := t.Field(idx)
		fieldValue := v.Field(idx)

		if !fieldValue.CanSet() {
			continue
		}

		given, err := func() (bool, error) {
			// raw:"body" hands over the request body exactly as received — for
			// binary uploads, which are neither JSON nor form data. The field
			// must be []byte or string.
			if tag := field.Tag.Get("raw"); tag == "body" {
				body, err := i.context.Request().Body()
				if err != nil {
					return false, err
				}
				switch {
				case fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.Uint8:
					fieldValue.SetBytes(body)
				case fieldValue.Kind() == reflect.String:
					fieldValue.SetString(string(body))
				}
				return len(body) > 0, nil
			}

			// raw:"query" hands over the query string exactly as sent (types.RawQuery): a
			// signature over it (the SAML HTTP-Redirect binding's) covers the sender's own
			// escapes, which the decoded queries lose. The field must be a string.
			if tag := field.Tag.Get("raw"); tag == "query" && fieldValue.Kind() == reflect.String {
				q := types.RawQuery(i.context.Request())
				fieldValue.SetString(q)
				return q != "", nil
			}

			// queries:"all" hands over every query parameter as a map[string]string
			// (a copy). A list endpoint filters on whatever the caller names, which the
			// DTO cannot know in advance, so it cannot list the fields one by one.
			if tag := field.Tag.Get("queries"); tag == "all" && fieldValue.Type() == reflect.TypeOf(map[string]string(nil)) {
				all := make(map[string]string, len(queries))
				for k, v := range queries {
					all[k] = v
				}
				fieldValue.Set(reflect.ValueOf(all))
				return true, nil
			}

			// headers:"all" hands over every request header (a copy), as
			// map[string]string (first value) or map[string][]string: a webhook
			// receiver verifies a provider's signature header it can't name in
			// advance.
			if tag := field.Tag.Get("headers"); tag == "all" {
				switch fieldValue.Type() {
				case reflect.TypeOf(map[string]string(nil)):
					all := make(map[string]string, len(headers))
					for k, vs := range headers {
						if len(vs) > 0 {
							all[k] = vs[0]
						}
					}
					fieldValue.Set(reflect.ValueOf(all))
					return true, nil
				case reflect.TypeOf(map[string][]string(nil)):
					all := make(map[string][]string, len(headers))
					for k, vs := range headers {
						all[k] = append([]string(nil), vs...)
					}
					fieldValue.Set(reflect.ValueOf(all))
					return true, nil
				}
			}

			// Check header tag
			if tag := field.Tag.Get("header"); tag != "" {
				if values, ok := headers[tag]; ok && len(values) > 0 {
					if err := setFieldValue(fieldValue, values[0]); err != nil {
						return false, errors.ErrFailedToSetHeaderField.WithError(err).WithMeta(field.Name)
					}
					return values[0] != "", nil
				}
			}

			// Check context tag
			if tag := field.Tag.Get("context"); tag != "" {
				if value := i.context.GetValue(tag); value != nil {
					if err := setFieldFromJSON(fieldValue, value); err != nil {
						return false, errors.ErrFailedToSetContextField.WithError(err).WithMeta(field.Name)
					}
					return true, nil
				}
			}

			// Check param tag
			if tag := field.Tag.Get("param"); tag != "" {
				if value, ok := params[tag]; ok {
					if err := setFieldValue(fieldValue, value); err != nil {
						return false, errors.ErrFailedToSetParamField.WithError(err).WithMeta(field.Name)
					}
					return value != "", nil
				}
			}

			// Check query tag
			if tag := field.Tag.Get("query"); tag != "" {
				if value, ok := queries[tag]; ok {
					if err := setFieldValue(fieldValue, value); err != nil {
						return false, errors.ErrFailedToSetQueryField.WithError(err).WithMeta(field.Name)
					}
					return value != "", nil
				}
			}

			// Check flag tag (for CLI)
			if tag := field.Tag.Get("flag"); tag != "" {
				if value, ok := queries[tag]; ok {
					if err := setFieldValue(fieldValue, value); err != nil {
						return false, errors.ErrFailedToSetFlagField.WithError(err).WithMeta(field.Name)
					}
					return value != "", nil
				}
			}

			// Check form tag
			if tag := field.Tag.Get("form"); tag != "" {
				if err := parseBody(); err != nil {
					return false, err
				}
				if formData != nil {
					if value := formData.Get(tag); value != "" {
						if err := setFieldValue(fieldValue, value); err != nil {
							return false, errors.ErrFailedToSetFormField.WithError(err).WithMeta(field.Name)
						}
						return true, nil
					}
				}
			}

			// Check json tag
			if tag := field.Tag.Get("json"); tag != "" {
				// Handle omitempty and other options
				tagName := strings.Split(tag, ",")[0]
				if tagName == "-" {
					return false, nil
				}

				if err := parseBody(); err != nil {
					return false, err
				}

				if jsonData != nil {
					if strings.HasSuffix(tag, ",merge") {
						if err := setFieldFromJSON(fieldValue, jsonData); err != nil {
							return false, errors.ErrFailedToSetJSONField.WithError(err).WithMeta(field.Name)
						}
						return true, nil
					}
					if value, ok := jsonData[tagName]; ok {
						if err := setFieldFromJSON(fieldValue, value); err != nil {
							return false, errors.ErrFailedToSetJSONField.WithError(err).WithMeta(field.Name)
						}
						// null, like an absent key, gives nothing; so does "" for a string
						if str, isStr := value.(string); value == nil || (isStr && str == "") {
							return false, nil
						}
						return true, nil
					}
				}
			}
			return false, nil
		}()
		if err != nil {
			return err
		}
		if !given && required(field) {
			missing = append(missing, wireName(field))
		}
	}

	if len(missing) > 0 {
		return errors.ErrMissingRequiredFields.WithMeta(missing).WithDetail("missing: " + strings.Join(missing, ", "))
	}
	return nil
}

// required reports whether a field is tagged binding:"required" (the same tag OpenAPI documents).
func required(field reflect.StructField) bool {
	for _, part := range strings.Split(field.Tag.Get("binding"), ",") {
		if strings.TrimSpace(part) == "required" {
			return !callerCannotSend(field)
		}
	}
	return false
}

// callerCannotSend is a field the request does not supply: the server fills it (context), it
// takes everything at once (queries:"all", headers:"all", json ",merge"), or json:"-".
func callerCannotSend(field reflect.StructField) bool {
	if field.Tag.Get("context") != "" || field.Tag.Get("queries") == "all" || field.Tag.Get("headers") == "all" {
		return true
	}
	if tag, ok := field.Tag.Lookup("json"); ok {
		name := strings.Split(tag, ",")[0]
		if strings.HasSuffix(tag, ",merge") {
			return true
		}
		if name == "-" && !hasOtherSource(field) {
			return true
		}
	}
	return false
}

func hasOtherSource(field reflect.StructField) bool {
	for _, k := range []string{"raw", "header", "param", "query", "flag", "form"} {
		if field.Tag.Get(k) != "" {
			return true
		}
	}
	return false
}

// wireName is the name a caller sends the field under: its json, form, query, header, param or
// flag name, "body" for raw:"body", or the Go name.
func wireName(field reflect.StructField) string {
	if n := strings.Split(field.Tag.Get("json"), ",")[0]; n != "" && n != "-" {
		return n
	}
	for _, k := range []string{"form", "query", "header", "param", "flag"} {
		if n := field.Tag.Get(k); n != "" {
			return n
		}
	}
	if field.Tag.Get("raw") == "body" {
		return "body"
	}
	return field.Name
}

func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			field.Set(reflect.ValueOf(strings.Split(value, ",")))
		} else {
			return errors.ErrUnsupportedSliceType.WithMeta(field.Type())
		}
	default:
		return errors.ErrUnsupportedFieldType.WithMeta(field.Kind())
	}
	return nil
}

// decodeJSONNumbers is json.Unmarshal with UseNumber (and, like Unmarshal,
// nothing allowed after the value).
func decodeJSONNumbers(body []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.ErrFailedToSetJSONField
	}
	return nil
}

var jsonNumberType = reflect.TypeOf(json.Number(""))

// asFloats turns the json.Numbers in v back into float64, recursively: what
// a field of type any (or map[string]any, []any, ...) received before numbers
// were kept exact, so handlers that type-assert float64 see no change.
func asFloats(v any) any {
	switch t := v.(type) {
	case json.Number:
		f, _ := t.Float64()
		return f
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, x := range t {
			out[k] = asFloats(x)
		}
		return out
	case props.Props:
		out := make(props.Props, len(t))
		for k, x := range t {
			out[k] = asFloats(x)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = asFloats(x)
		}
		return out
	}
	return v
}

func setFieldFromJSON(field reflect.Value, value any) error {
	if value == nil {
		return nil
	}

	fieldType := field.Type()

	if n, ok := value.(json.Number); ok {
		switch {
		case fieldType == jsonNumberType:
			field.Set(reflect.ValueOf(n))
			return nil
		case field.Kind() == reflect.String:
			field.SetString(n.String())
			return nil
		case field.Kind() >= reflect.Int && field.Kind() <= reflect.Int64:
			if i, err := strconv.ParseInt(n.String(), 10, 64); err == nil {
				field.SetInt(i)
				return nil
			}
			f, err := n.Float64()
			if err != nil {
				return errors.ErrCannotConvertToInt.WithMeta(value)
			}
			field.SetInt(int64(f))
			return nil
		case field.Kind() >= reflect.Uint && field.Kind() <= reflect.Uint64:
			if u, err := strconv.ParseUint(n.String(), 10, 64); err == nil {
				field.SetUint(u)
				return nil
			}
			f, err := n.Float64()
			if err != nil {
				return errors.ErrCannotConvertToUint.WithMeta(value)
			}
			field.SetUint(uint64(f))
			return nil
		}
	}

	// Slices, maps and structs are re-marshalled below, where a json.Number
	// writes out exactly as it came; anything assigned directly (a field of
	// type any, map[string]any, props.Props, ...) gets float64s as before.
	switch field.Kind() {
	case reflect.Slice, reflect.Map, reflect.Struct:
		if reflect.TypeOf(value).AssignableTo(fieldType) && !containsNumbers(value) {
			field.Set(reflect.ValueOf(value))
			return nil
		}
		if fieldType.Kind() == reflect.Map && fieldType.Elem().Kind() == reflect.Interface {
			value = asFloats(value)
		} else if fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Interface {
			value = asFloats(value)
		}
	default:
		value = asFloats(value)
	}
	valueReflect := reflect.ValueOf(value)

	// Handle direct assignment if types match
	if valueReflect.Type().AssignableTo(fieldType) {
		field.Set(valueReflect)
		return nil
	}

	// Handle type conversions
	switch field.Kind() {
	case reflect.String:
		switch v := value.(type) {
		case string:
			field.SetString(v)
		case float64:
			field.SetString(strconv.FormatFloat(v, 'f', -1, 64))
		default:
			field.SetString(strconv.FormatFloat(0, 'f', -1, 64))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case float64:
			field.SetInt(int64(v))
		case string:
			intVal, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(intVal)
		default:
			return errors.ErrCannotConvertToInt.WithMeta(value)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := value.(type) {
		case float64:
			field.SetUint(uint64(v))
		case string:
			uintVal, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return err
			}
			field.SetUint(uintVal)
		default:
			return errors.ErrCannotConvertToUint.WithMeta(value)
		}
	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float64:
			field.SetFloat(v)
		case string:
			floatVal, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return err
			}
			field.SetFloat(floatVal)
		default:
			return errors.ErrCannotConvertToFloat.WithMeta(value)
		}
	case reflect.Bool:
		switch v := value.(type) {
		case bool:
			field.SetBool(v)
		case string:
			boolVal, err := strconv.ParseBool(v)
			if err != nil {
				return err
			}
			field.SetBool(boolVal)
		default:
			return errors.ErrCannotConvertToBool.WithMeta(value)
		}
	case reflect.Slice, reflect.Map, reflect.Struct:
		// Re-marshal and unmarshal to handle complex types
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return err
		}
		newVal := reflect.New(fieldType)
		if err := json.Unmarshal(jsonBytes, newVal.Interface()); err != nil {
			return err
		}
		field.Set(newVal.Elem())
	case reflect.Ptr:
		// Create a new instance and set it
		newVal := reflect.New(fieldType.Elem())
		if err := setFieldFromJSON(newVal.Elem(), value); err != nil {
			return err
		}
		field.Set(newVal)
	default:
		return errors.ErrUnsupportedJSONFieldType.WithMeta(field.Kind())
	}
	return nil
}

// containsNumbers reports whether v holds a json.Number anywhere.
func containsNumbers(v any) bool {
	switch t := v.(type) {
	case json.Number:
		return true
	case map[string]any:
		for _, x := range t {
			if containsNumbers(x) {
				return true
			}
		}
	case props.Props:
		for _, x := range t {
			if containsNumbers(x) {
				return true
			}
		}
	case []any:
		for _, x := range t {
			if containsNumbers(x) {
				return true
			}
		}
	}
	return false
}

// MissingFields reports the wire names of the required fields err says are missing
// (errors.ErrMissingRequiredFields), in field order.
func MissingFields(err error) ([]string, bool) {
	var e *errors.Error
	if !stderrors.As(err, &e) || e.GetCode() != errors.ErrMissingRequiredFields.GetCode() {
		return nil, false
	}
	fields, ok := e.GetMeta().([]string)
	return fields, ok && len(fields) > 0
}
