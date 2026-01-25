package tools

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// RegisterAll automatically registers all tool methods from a struct
// Methods must have the signature: func (T) Method(ctx context.Context, params Params) (string, error)
// Tool names are derived from method names (e.g., ViewFile -> view_file)
//
// Example:
//
//	type ViewFileParams struct {
//	    Path string `json:"path" required:"true" description:"Path to the file"`
//	}
//
//	func (ft *FileTools) ViewFile(ctx context.Context, params ViewFileParams) (string, error) {
//	    // implementation
//	}
//
// Usage:
//
//	tt.RegisterAll(fileTools, map[string]string{
//	    "ViewFile": "Read file contents with line numbers",
//	    "ReplaceFile": "Create or overwrite a file",
//	})
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
	paramValue := reflect.New(rt.paramType)  // Creates *T

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
		paramValue.Elem(),  // Get the value T from *T
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

// parseTag parses a tag string in the format key="value" key2="value2"
func parseTag(tag, key string) string {
	prefix := key + `="`
	idx := strings.Index(tag, prefix)
	if idx == -1 {
		return ""
	}
	start := idx + len(prefix)
	end := strings.Index(tag[start:], `"`)
	if end == -1 {
		return ""
	}
	return tag[start : start+end]
}
