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

// ConstrainedTool is an optional interface for tools with output constraints
// Tools that implement this interface will have their outputs automatically limited
type ConstrainedTool interface {
	Tool

	// Constraint returns the output constraint for this tool
	Constraint() tool.OutputConstraint
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

// RegisterAll automatically registers all tool methods from a struct
// Methods must have the signature: func (T) Method(ctx context.Context, params Params) (string, error)
// Tool names are derived from method names (e.g., ViewFile -> view_file)
func (tt *TypedToolkit) RegisterAll(provider any, descriptions ...map[string]string) error {
	val := reflect.ValueOf(provider)
	typ := val.Type()

	// Get description map if provided
	descMap := make(map[string]string)
	if len(descriptions) > 0 && descriptions[0] != nil {
		descMap = descriptions[0]
	}

	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)

		// Check method signature: func (T) Method(ctx, params) (string, error)
		if method.Type.NumIn() != 3 || method.Type.NumOut() != 2 {
			continue
		}

		// Check first parameter is context.Context
		ctxType := method.Type.In(1)
		if ctxType != reflect.TypeOf((*context.Context)(nil)).Elem() {
			continue
		}

		// Check return types: string and error
		if method.Type.Out(0) != reflect.TypeOf("") || method.Type.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}

		// Get the parameter type (should be a struct)
		paramType := method.Type.In(2)
		if paramType.Kind() != reflect.Struct {
			continue
		}

		// Create tool name from method name (e.g., ViewFile -> view_file)
		name := toSnakeCase(method.Name)

		// Get description from map
		description := descMap[method.Name]
		if description == "" {
			description = "Tool: " + name
		}

		// Create the tool
		tool := &ReflectTool{
			name:        name,
			description: description,
			method:      method,
			receiver:    provider,
			paramType:   paramType,
			paramSchema: StructToSchema(reflect.New(paramType).Elem().Interface()),
		}

		tt.Register(tool)
	}

	return nil
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

	// Apply constraint if the tool implements ConstrainedTool
	var constraint tool.OutputConstraint

	// Priority 1: Tool implements ConstrainedTool
	if constrained, ok := t.(ConstrainedTool); ok {
		constraint = constrained.Constraint()
	}

	// Priority 2: Global constraint from config
	if constraint == nil {
		if c, ok := tool.GetGlobalConstraint(toolBlock.Name); ok {
			constraint = c
		}
	}

	// Priority 3: Global default constraint
	if constraint == nil {
		if c := tool.GetGlobalDefaultConstraint(); c != nil {
			constraint = c
		}
	}

	// Apply constraint if exists
	if constraint != nil {
		result = constraint.Apply(result)
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

// normalizeBoolStrings converts string representations of booleans and numbers to actual types
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
				// Try to convert to integer
				if intVal, err := parseInt(str); err == nil {
					result[k] = intVal
				} else if floatVal, err := parseFloat(str); err == nil {
					result[k] = floatVal
				} else {
					result[k] = v
				}
			}
		} else if nestedMap, ok := v.(map[string]any); ok {
			result[k] = normalizeBoolStrings(nestedMap)
		} else if slice, ok := v.([]any); ok {
			result[k] = normalizeSlice(slice)
		} else {
			result[k] = v
		}
	}
	return result
}

// normalizeSlice recursively normalizes elements in a slice
func normalizeSlice(slice []any) []any {
	result := make([]any, len(slice))
	for i, v := range slice {
		if str, ok := v.(string); ok {
			switch strings.ToLower(str) {
			case "true":
				result[i] = true
			case "false":
				result[i] = false
			default:
				if intVal, err := parseInt(str); err == nil {
					result[i] = intVal
				} else if floatVal, err := parseFloat(str); err == nil {
					result[i] = floatVal
				} else {
					result[i] = v
				}
			}
		} else if nestedMap, ok := v.(map[string]any); ok {
			result[i] = normalizeBoolStrings(nestedMap)
		} else if nestedSlice, ok := v.([]any); ok {
			result[i] = normalizeSlice(nestedSlice)
		} else {
			result[i] = v
		}
	}
	return result
}

// parseInt attempts to parse a string to an integer
// Returns the parsed int and nil error if successful
func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	// Check for decimal point or exponent - if present, it's a float
	if strings.Contains(s, ".") || strings.Contains(s, "e") || strings.Contains(s, "E") {
		return 0, fmt.Errorf("float format")
	}

	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return 0, err
	}

	// Verify the entire string was consumed (no trailing chars)
	var verify int
	if n, _ := fmt.Sscanf(s, "%d%c", &verify, new(rune)); n != 1 {
		return 0, fmt.Errorf("trailing characters")
	}

	return result, nil
}

// parseFloat attempts to parse a string to a float64
// Returns the parsed float64 and nil error if successful
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	if err != nil {
		return 0, err
	}

	// Verify the entire string was consumed
	var verify float64
	if n, _ := fmt.Sscanf(s, "%f%c", &verify, new(rune)); n != 1 {
		return 0, fmt.Errorf("trailing characters")
	}

	return result, nil
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

// ReflectTool wraps a method as a Tool using reflection
type ReflectTool struct {
	name        string
	description string
	method      reflect.Method
	receiver    any
	paramType   reflect.Type
	paramSchema map[string]any
}

func (rt *ReflectTool) Name() string {
	return rt.name
}

func (rt *ReflectTool) Description() string {
	return rt.description
}

func (rt *ReflectTool) ParameterSchema() map[string]any {
	return rt.paramSchema
}

func (rt *ReflectTool) Call(ctx context.Context, params any) (string, error) {
	// Convert params map to struct
	paramValue := reflect.New(rt.paramType) // Creates *T

	if paramsMap, ok := params.(map[string]any); ok {
		if err := MapToStruct(paramsMap, paramValue.Interface()); err != nil {
			return "", fmt.Errorf("failed to parse parameters: %w", err)
		}
	} else {
		return "", fmt.Errorf("expected map[string]any, got %T", params)
	}

	// Call the method via reflection
	// paramValue is *T, but method expects T, so use .Elem()
	results := rt.method.Func.Call([]reflect.Value{
		reflect.ValueOf(rt.receiver),
		reflect.ValueOf(ctx),
		paramValue.Elem(), // Get the value T from *T
	})

	// Parse results
	if len(results) != 2 {
		return "", fmt.Errorf("expected 2 results, got %d", len(results))
	}

	// Check error
	if !results[1].IsNil() {
		return "", results[1].Interface().(error)
	}

	return results[0].String(), nil
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
