package tools

import (
	"encoding/json"
	"testing"
)

// TestJSONTagUnmarshal tests that JSON unmarshaling works correctly with snake_case tags
func TestJSONTagUnmarshal(t *testing.T) {
	type TestParams struct {
		FilePath string `json:"file_path" required:"true"`
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantVal string
	}{
		{
			name:    "correct snake_case key",
			input:   `{"file_path": "/test/path"}`,
			wantErr: false,
			wantVal: "/test/path",
		},
		{
			name:    "incorrect camelCase key",
			input:   `{"FilePath": "/test/path"}`,
			wantErr: false, // JSON unmarshal doesn't error, just doesn't populate the field
			wantVal: "",    // empty because key doesn't match
		},
		{
			name:    "both keys - snake_case takes precedence",
			input:   `{"file_path": "/test/snake", "FilePath": "/test/camel"}`,
			wantErr: false,
			wantVal: "/test/snake",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params TestParams
			err := json.Unmarshal([]byte(tt.input), &params)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if params.FilePath != tt.wantVal {
				t.Errorf("FilePath = %q, want %q", params.FilePath, tt.wantVal)
			}
		})
	}
}

// TestMapToStructWithJSONTags tests MapToStruct with different key formats
func TestMapToStructWithJSONTags(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		wantVal string
	}{
		{
			name:    "correct snake_case key",
			input:   map[string]any{"file_path": "/test/path"},
			wantVal: "/test/path",
		},
		{
			name:    "incorrect camelCase key",
			input:   map[string]any{"FilePath": "/test/path"},
			wantVal: "", // empty because key doesn't match JSON tag
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := ViewFileParams{}
			err := MapToStruct(tt.input, &params)
			if err != nil {
				t.Errorf("MapToStruct() error = %v", err)
				return
			}
			if params.FilePath != tt.wantVal {
				t.Errorf("FilePath = %q, want %q", params.FilePath, tt.wantVal)
			}
		})
	}
}

// TestStructToSchema verifies that the schema generation uses JSON tags correctly
func TestStructToSchema(t *testing.T) {
	schema := StructToSchema(ViewFileParams{})

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema.properties is not a map")
	}

	// Check that file_path (not FilePath) is in the schema
	if _, ok := props["file_path"]; !ok {
		t.Error("schema should contain 'file_path' property")
	}

	// Check that FilePath is NOT in the schema (it should use the JSON tag)
	if _, ok := props["FilePath"]; ok {
		t.Error("schema should NOT contain 'FilePath' property (should use JSON tag 'file_path')")
	}

	// Print schema for debugging
	t.Logf("Generated schema: %+v", schema)
}

// TestViewFileParamsJSONRoundTrip tests JSON serialization round-trip
func TestViewFileParamsJSONRoundTrip(t *testing.T) {
	original := ViewFileParams{
		FilePath: "/test/django/db/models/aggregates.py",
		Limit:    100,
		Offset:   10,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	t.Logf("JSON output: %s", string(jsonData))

	// Unmarshal back
	var decoded ViewFileParams
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.FilePath != original.FilePath {
		t.Errorf("FilePath = %q, want %q", decoded.FilePath, original.FilePath)
	}
	if decoded.Limit != original.Limit {
		t.Errorf("Limit = %d, want %d", decoded.Limit, original.Limit)
	}
	if decoded.Offset != original.Offset {
		t.Errorf("Offset = %d, want %d", decoded.Offset, original.Offset)
	}

	// Also test that unmarshaling with wrong key name doesn't work
	var wrongDecoded ViewFileParams
	wrongJSON := `{"FilePath": "/test/path", "Limit": 100, "Offset": 10}`
	if err := json.Unmarshal([]byte(wrongJSON), &wrongDecoded); err != nil {
		t.Fatalf("json.Unmarshal() with wrong keys error = %v", err)
	}

	t.Logf("After unmarshal with wrong keys: FilePath=%q, Limit=%d, Offset=%d",
		wrongDecoded.FilePath, wrongDecoded.Limit, wrongDecoded.Offset)

	if wrongDecoded.FilePath != "" {
		t.Errorf("FilePath should be empty when using wrong key, got %q", wrongDecoded.FilePath)
	}
}
