package input

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"

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
		return fmt.Errorf("payload must be a non-nil pointer to a struct")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("payload must be a pointer to a struct")
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

		// Try JSON first
		if err := json.Unmarshal(body, &jsonData); err == nil {
			return nil
		}

		// Try form data
		formData, _ = url.ParseQuery(string(body))
		return nil
	}

	for idx := 0; idx < t.NumField(); idx++ {
		field := t.Field(idx)
		fieldValue := v.Field(idx)

		if !fieldValue.CanSet() {
			continue
		}

		// Check header tag
		if tag := field.Tag.Get("header"); tag != "" {
			if values, ok := headers[tag]; ok && len(values) > 0 {
				if err := setFieldValue(fieldValue, values[0]); err != nil {
					return fmt.Errorf("failed to set header field %s: %w", field.Name, err)
				}
				continue
			}
		}

		// Check context tag
		if tag := field.Tag.Get("context"); tag != "" {
			if value := i.context.GetValue(tag); value != nil {
				if err := setFieldFromJSON(fieldValue, value); err != nil {
					return fmt.Errorf("failed to set context field %s: %w", field.Name, err)
				}
				continue
			}
		}

		// Check param tag
		if tag := field.Tag.Get("param"); tag != "" {
			if value, ok := params[tag]; ok {
				if err := setFieldValue(fieldValue, value); err != nil {
					return fmt.Errorf("failed to set param field %s: %w", field.Name, err)
				}
				continue
			}
		}

		// Check query tag
		if tag := field.Tag.Get("query"); tag != "" {
			if value, ok := queries[tag]; ok {
				if err := setFieldValue(fieldValue, value); err != nil {
					return fmt.Errorf("failed to set query field %s: %w", field.Name, err)
				}
				continue
			}
		}

		// Check flag tag (for CLI)
		if tag := field.Tag.Get("flag"); tag != "" {
			if value, ok := queries[tag]; ok {
				if err := setFieldValue(fieldValue, value); err != nil {
					return fmt.Errorf("failed to set flag field %s: %w", field.Name, err)
				}
				continue
			}
		}

		// Check form tag
		if tag := field.Tag.Get("form"); tag != "" {
			if err := parseBody(); err != nil {
				return err
			}
			if formData != nil {
				if value := formData.Get(tag); value != "" {
					if err := setFieldValue(fieldValue, value); err != nil {
						return fmt.Errorf("failed to set form field %s: %w", field.Name, err)
					}
					continue
				}
			}
		}

		// Check json tag
		if tag := field.Tag.Get("json"); tag != "" {
			// Handle omitempty and other options
			tagName := strings.Split(tag, ",")[0]
			if tagName == "-" {
				continue
			}

			if err := parseBody(); err != nil {
				return err
			}
			if jsonData != nil {
				if value, ok := jsonData[tagName]; ok {
					if err := setFieldFromJSON(fieldValue, value); err != nil {
						return fmt.Errorf("failed to set json field %s: %w", field.Name, err)
					}
				}
			}
		}
	}

	return nil
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
			return fmt.Errorf("unsupported slice type: %v", field.Type())
		}
	default:
		return fmt.Errorf("unsupported field type: %v", field.Kind())
	}
	return nil
}

func setFieldFromJSON(field reflect.Value, value any) error {
	if value == nil {
		return nil
	}

	fieldType := field.Type()
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
			field.SetString(fmt.Sprintf("%v", v))
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
			return fmt.Errorf("cannot convert %T to int", value)
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
			return fmt.Errorf("cannot convert %T to uint", value)
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
			return fmt.Errorf("cannot convert %T to float", value)
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
			return fmt.Errorf("cannot convert %T to bool", value)
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
		return fmt.Errorf("unsupported field type for JSON: %v", field.Kind())
	}
	return nil
}
