package toolschema

import (
	"testing"
)

func TestStructToSchema(t *testing.T) {
	type TestStruct struct {
		Name        string `json:"name"`
		Age         int    `json:"age"`
		Active      bool   `json:"active"`
		Optional    string `json:"optional,omitempty"`
		Description string `json:"description" description:"The description field"`
		Ignored     string `json:"-"`
		unexported  string
	}

	schema := StructToSchema(TestStruct{})

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties is not a map")
	}

	// Check required fields
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("required is not a slice")
	}

	// Verify required fields include Name, Age, Active, Description
	requiredMap := make(map[string]bool)
	for _, r := range required {
		requiredMap[r] = true
	}

	if !requiredMap["name"] {
		t.Error("name should be required")
	}
	if !requiredMap["age"] {
		t.Error("age should be required")
	}
	if !requiredMap["active"] {
		t.Error("active should be required")
	}
	if requiredMap["optional"] {
		t.Error("optional should not be required (has omitempty)")
	}

	// Verify types
	if props["name"].(map[string]any)["type"] != "string" {
		t.Error("name should be string type")
	}
	if props["age"].(map[string]any)["type"] != "integer" {
		t.Error("age should be integer type")
	}
	if props["active"].(map[string]any)["type"] != "boolean" {
		t.Error("active should be boolean type")
	}

	// Verify description tag
	if props["description"].(map[string]any)["description"] != "The description field" {
		t.Error("description tag not parsed correctly")
	}

	// Verify ignored and unexported fields are not present
	if _, exists := props["Ignored"]; exists {
		t.Error("Ignored field should not be in schema")
	}
	if _, exists := props["unexported"]; exists {
		t.Error("unexported field should not be in schema")
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ViewFile", "view_file"},
		{"EditFile", "edit_file"},
		{"ExecuteBash", "execute_bash"},
		{"ListDirectory", "list_directory"},
		{"Simple", "simple"},
		{"HTTPServer", "h_t_t_p_server"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStructToSchemaNonStruct(t *testing.T) {
	// Test with non-struct input
	schema := StructToSchema("not a struct")

	if schema["type"] != "object" {
		t.Error("non-struct should return object type")
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties is not a map")
	}
	if len(props) != 0 {
		t.Error("non-struct should have empty properties")
	}
}

func TestStructToSchemaPointer(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	// Test with pointer input
	schema := StructToSchema(&TestStruct{})

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties is not a map")
	}

	if _, exists := props["name"]; !exists {
		t.Error("name field should be in schema")
	}
}

func TestStructToSchemaSliceAndMap(t *testing.T) {
	type TestStruct struct {
		Items []string          `json:"items"`
		Meta  map[string]string `json:"meta"`
	}

	schema := StructToSchema(TestStruct{})

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties is not a map")
	}

	if props["items"].(map[string]any)["type"] != "array" {
		t.Error("items should be array type")
	}
	if props["meta"].(map[string]any)["type"] != "object" {
		t.Error("meta should be object type")
	}
}
