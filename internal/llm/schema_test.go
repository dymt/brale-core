package llm_test

import (
	"testing"

	"brale-core/internal/decision/agent"
	"brale-core/internal/decision/provider"
	"brale-core/internal/llm"
)

func TestSchemaFromTypeBuildsNestedObjectsAndEnums(t *testing.T) {
	schema := llm.SchemaFromType[provider.MechanicsProviderOut]()
	if schema == nil {
		t.Fatalf("schema is nil")
	}

	properties, _ := schema.Schema["properties"].(map[string]any)
	liq, _ := properties["liquidation_stress"].(map[string]any)
	if liq["type"] != "object" {
		t.Fatalf("liquidation_stress.type=%v want object", liq["type"])
	}

	liqProps, _ := liq["properties"].(map[string]any)
	conf, _ := liqProps["confidence"].(map[string]any)
	enumVals, _ := conf["enum"].([]any)
	if len(enumVals) == 0 {
		t.Fatalf("confidence enum should not be empty")
	}
}

func TestSchemaFromTypeUsesRegisteredEnums(t *testing.T) {
	schema := llm.SchemaFromType[agent.IndicatorSummary]()
	properties, _ := schema.Schema["properties"].(map[string]any)
	expansion, _ := properties["expansion"].(map[string]any)
	enumVals, _ := expansion["enum"].([]any)
	if len(enumVals) == 0 {
		t.Fatalf("expansion enum should not be empty")
	}
}

func TestSchemaFromTypeOmitsIgnoredFieldsAndOptionalRequired(t *testing.T) {
	type sample struct {
		Name   string `json:"name,omitempty"`
		Hidden string `json:"-"`
	}

	schema := llm.SchemaFromType[sample]()
	properties, _ := schema.Schema["properties"].(map[string]any)
	if _, ok := properties["hidden"]; ok {
		t.Fatalf("hidden field should be omitted")
	}

	required, _ := schema.Schema["required"].([]string)
	for _, item := range required {
		if item == "name" {
			t.Fatalf("omitempty field should not be required")
		}
	}
}

func TestSchemaFromTypeHandlesRecursiveTypes(t *testing.T) {
	type node struct {
		Name  string `json:"name"`
		Child *node  `json:"child,omitempty"`
	}

	schema := llm.SchemaFromType[node]()
	properties, _ := schema.Schema["properties"].(map[string]any)
	child, _ := properties["child"].(map[string]any)
	if child["type"] != "object" {
		t.Fatalf("child.type=%v want object", child["type"])
	}
}
