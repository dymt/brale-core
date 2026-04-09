package llm

import (
	"reflect"
	"strings"
	"sync"
)

type JSONSchema struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
}

var (
	enumRegistry sync.Map
	schemaCache  sync.Map
)

func RegisterEnum[T ~string](values ...string) {
	enum := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		enum = append(enum, value)
	}
	enumRegistry.Store(reflect.TypeFor[T](), enum)
}

func SchemaFromType[T any]() *JSONSchema {
	typ := reflect.TypeFor[T]()
	if cached, ok := schemaCache.Load(typ); ok {
		return cached.(*JSONSchema)
	}
	schema := &JSONSchema{
		Name:   schemaName(typ),
		Schema: schemaForType(typ),
	}
	actual, _ := schemaCache.LoadOrStore(typ, schema)
	return actual.(*JSONSchema)
}

func schemaName(typ reflect.Type) string {
	if typ == nil {
		return "anonymous"
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Name() == "" {
		return "anonymous"
	}
	return strings.ToLower(typ.Name())
}

func schemaForType(typ reflect.Type) map[string]any {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return scalarSchema(typ)
	}

	properties := make(map[string]any)
	required := make([]string, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		name, optional, ok := jsonFieldName(field)
		if !ok {
			continue
		}
		properties[name] = fieldSchema(field.Type)
		if !optional {
			required = append(required, name)
		}
	}

	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func fieldSchema(typ reflect.Type) map[string]any {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if enum, ok := loadEnum(typ); ok {
		return map[string]any{
			"type": "string",
			"enum": enum,
		}
	}
	switch typ.Kind() {
	case reflect.Struct:
		return schemaForType(typ)
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": fieldSchema(typ.Elem()),
		}
	default:
		return scalarSchema(typ)
	}
}

func scalarSchema(typ reflect.Type) map[string]any {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	default:
		return map[string]any{"type": "string"}
	}
}

func loadEnum(typ reflect.Type) ([]any, bool) {
	if raw, ok := enumRegistry.Load(typ); ok {
		values := raw.([]string)
		out := make([]any, 0, len(values))
		for _, value := range values {
			out = append(out, value)
		}
		return out, true
	}
	return nil, false
}

func jsonFieldName(field reflect.StructField) (string, bool, bool) {
	tag := strings.TrimSpace(field.Tag.Get("json"))
	if tag == "-" {
		return "", false, false
	}
	if tag == "" {
		return field.Name, false, true
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		name = field.Name
	}
	optional := false
	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "omitempty" {
			optional = true
		}
	}
	return name, optional, true
}
