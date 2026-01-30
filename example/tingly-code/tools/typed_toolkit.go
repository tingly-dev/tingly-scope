package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/tingly-dev/tingly-scope/pkg/message"
	"github.com/tingly-dev/tingly-scope/pkg/model"
	"github.com/tingly-dev/tingly-scope/pkg/tool"
)

// Tool is a type-safe tool with structured parameters
type Tool interface {
	// Name returns the tool name
	Name() string

	// Description returns what the tool does
	Description() string

	// ParameterSchema returns the JSON Schema for parameters
	ParameterSchema() map[string]any

	// Call executes the tool with the given parameters
	Call(ctx context.Context, params any) (string, error)
}

// ToolInfo holds metadata about a tool
type ToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// TypedToolkit is a type-safe toolkit that doesn't use reflection
type TypedToolkit struct {
	tools map[string]Tool
}

// NewTypedToolkit creates a new type-safe toolkit
func NewTypedToolkit() *TypedToolkit {
	return &TypedToolkit{
		tools: make(map[string]Tool),
	}
}

// Register registers a tool
func (tt *TypedToolkit) Register(tool Tool) {
	tt.tools[tool.Name()] = tool
}

// GetSchemas returns all tool schemas for the model
func (tt *TypedToolkit) GetSchemas() []ToolInfo {
	result := make([]ToolInfo, 0, len(tt.tools))
	for _, tool := range tt.tools {
		result = append(result, ToolInfo{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.ParameterSchema(),
		})
	}
	return result
}

// Call executes a tool by name with parameters
func (tt *TypedToolkit) Call(ctx context.Context, toolName string, params map[string]any) (string, error) {
	tool, ok := tt.tools[toolName]
	if !ok {
		return fmt.Sprintf("Error: unknown tool '%s'", toolName), nil
	}

	return tool.Call(ctx, params)
}

// GetModelSchemas returns all tool schemas in model.ToolDefinition format
// This provides compatibility with ReActAgent
func (tt *TypedToolkit) GetModelSchemas() []model.ToolDefinition {
	result := make([]model.ToolDefinition, 0, len(tt.tools))
	for _, tool := range tt.tools {
		result = append(result, model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.ParameterSchema(),
			},
		})
	}
	return result
}

// CallToolBlock executes a tool using a ToolUseBlock
// This provides compatibility with ReActAgent
func (tt *TypedToolkit) CallToolBlock(ctx context.Context, toolBlock *message.ToolUseBlock) (*tool.ToolResponse, error) {
	t, ok := tt.tools[toolBlock.Name]
	if !ok {
		return tool.TextResponse(fmt.Sprintf("Error: unknown tool '%s'", toolBlock.Name)), nil
	}

	// Convert toolBlock.Input to map[string]any
	params := make(map[string]any)
	for k, v := range toolBlock.Input {
		params[k] = v
	}

	result, err := t.Call(ctx, params)
	if err != nil {
		return tool.TextResponse(fmt.Sprintf("Error: %v", err)), nil
	}

	return tool.TextResponse(result), nil
}

// Filter removes disabled tools from the toolkit
// enabled map: tool name -> true if enabled, false if disabled
// If a tool is not in the map, it is kept (default on/opt-out model)
func (tt *TypedToolkit) Filter(enabled map[string]bool) {
	for name, isDisabled := range enabled {
		if !isDisabled {
			delete(tt.tools, name)
		}
	}
}

// HasTool checks if a tool is registered
func (tt *TypedToolkit) HasTool(name string) bool {
	_, ok := tt.tools[name]
	return ok
}

// ToolCount returns the number of registered tools
func (tt *TypedToolkit) ToolCount() int {
	return len(tt.tools)
}

// StructToSchema converts a struct to JSON Schema
func StructToSchema(v any) map[string]any {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	properties := make(map[string]any)
	required := []string{}

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fieldValue := val.Field(i)
		jsonTag := field.Tag.Get("json")

		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse json tag
		name := field.Name
		if jsonTag != "" {
			// Simple parsing: "name,omitempty" or "name"
			for j, c := range jsonTag {
				if c == ',' {
					name = jsonTag[:j]
					break
				}
				if c == 0 {
					name = jsonTag
					break
				}
			}
		}

		// Check if required
		isRequired := !strings.Contains(jsonTag, "omitempty")

		prop := make(map[string]any)

		// Set type based on field kind
		switch fieldValue.Kind() {
		case reflect.String:
			prop["type"] = "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			prop["type"] = "integer"
		case reflect.Float32, reflect.Float64:
			prop["type"] = "number"
		case reflect.Bool:
			prop["type"] = "boolean"
		default:
			prop["type"] = "string"
		}

		// Add description if available
		if desc := getFieldTag(field, "description"); desc != "" {
			prop["description"] = desc
		}

		properties[name] = prop

		if isRequired {
			required = append(required, name)
		}
	}

	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// getFieldTag gets a custom tag from a struct field
func getFieldTag(field reflect.StructField, tagKey string) string {
	tag := field.Tag.Get(tagKey)
	if tag == "" {
		return ""
	}
	// Parse "key:\"value\"" format
	for i := 0; i < len(tag); i++ {
		if tag[i] == '"' && i+1 < len(tag) {
			for j := i + 1; j < len(tag); j++ {
				if tag[j] == '"' {
					return tag[i+1 : j]
				}
			}
		}
	}
	return ""
}

// normalizeBoolStrings converts string representations of booleans to actual bool values
func normalizeBoolStrings(m map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		// Convert string "true"/"false" to bool
		if str, ok := v.(string); ok {
			switch strings.ToLower(str) {
			case "true":
				result[k] = true
			case "false":
				result[k] = false
			default:
				result[k] = v
			}
		} else if nestedMap, ok := v.(map[string]any); ok {
			result[k] = normalizeBoolStrings(nestedMap)
		} else {
			result[k] = v
		}
	}
	return result
}

// MapToStruct converts a map to a struct using JSON unmarshaling
func MapToStruct(m map[string]any, target interface{}) error {
	// Normalize string booleans to actual booleans
	normalized := normalizeBoolStrings(m)

	data, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
