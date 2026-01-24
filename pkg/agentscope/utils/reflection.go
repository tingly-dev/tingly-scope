package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// ParseFunctionSchema extracts JSON schema from a function's reflection
func ParseFunctionSchema(fn any) (map[string]any, error) {
	fnType := reflect.TypeOf(fn)
	fnValue := reflect.ValueOf(fn)

	// Handle function pointers
	if fnType.Kind() == reflect.Ptr {
		fnValue = fnValue.Elem()
		fnType = fnValue.Type()
	}

	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected a function, got %s", fnType.Kind())
	}

	name := getFunctionName(fnType)
	description := getFunctionDescription(fn)

	// Extract parameters
	parameters, required, err := extractParameters(fnType)
	if err != nil {
		return nil, fmt.Errorf("failed to extract parameters: %w", err)
	}

	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        name,
			"description": description,
			"parameters": map[string]any{
				"type":       "object",
				"properties": parameters,
				"required":   required,
			},
		},
	}, nil
}

// getFunctionName extracts the function name
func getFunctionName(fnType reflect.Type) string {
	// For simple functions, use the type name
	name := fnType.String()
	// Clean up generic function names
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		name = parts[len(parts)-1]
	}
	// Remove any function signature
	if idx := strings.Index(name, "("); idx >= 0 {
		name = name[:idx]
	}
	return name
}

// getFunctionDescription extracts the function description from doc comment
func getFunctionDescription(fn any) string {
	// In a full implementation, this would parse the function's doc comments
	// For now, return a generic description
	return "A tool function"
}

// extractParameters extracts parameter information from function signature
func extractParameters(fnType reflect.Type) (map[string]any, []string, error) {
	properties := make(map[string]any)
	required := []string{}

	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)

		// Skip context.Context
		if paramType.String() == "context.Context" {
			continue
		}

		// For map[string]any (common tool function signature)
		if paramType.Kind() == reflect.Map && paramType.Key().Kind() == reflect.String {
			properties["kwargs"] = map[string]any{
				"type":        "object",
				"description": "Function arguments",
			}
			return properties, required, nil
		}
	}

	return properties, required, nil
}

// RepairJSON attempts to repair malformed JSON
func RepairJSON(jsonStr string) (string, error) {
	// Remove common issues
	jsonStr = strings.TrimSpace(jsonStr)

	// Try to complete truncated JSON
	if !strings.HasSuffix(jsonStr, "}") && !strings.HasSuffix(jsonStr, "]") {
		// Count braces to balance
		openBraces := strings.Count(jsonStr, "{") - strings.Count(jsonStr, "}")
		_ = strings.Count(jsonStr, "}") - strings.Count(jsonStr, "{") // closeBraces - used when negative
		openBrackets := strings.Count(jsonStr, "[") - strings.Count(jsonStr, "]")
		_ = strings.Count(jsonStr, "]") - strings.Count(jsonStr, "[") // closeBrackets - used when negative

		for i := 0; i < openBraces; i++ {
			jsonStr += "}"
		}
		// Note: handling of negative closeBraces/closeBrackets would go here
		for i := 0; i < openBrackets; i++ {
			jsonStr += "]"
		}
	}

	// Validate JSON
	var result any
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return "", err
	}

	return jsonStr, nil
}

// RemoveTitleField removes title fields from JSON schema
func RemoveTitleField(schema map[string]any) {
	delete(schema, "title")

	if props, ok := schema["properties"].(map[string]any); ok {
		for _, prop := range props {
			if propMap, ok := prop.(map[string]any); ok {
				RemoveTitleField(propMap)
			}
		}
	}

	if items, ok := schema["items"].(map[string]any); ok {
		RemoveTitleField(items)
	}

	if addProps, ok := schema["additionalProperties"].(map[string]any); ok {
		RemoveTitleField(addProps)
	}
}

// CreateToolFromSchema creates a tool definition from a JSON schema
func CreateToolFromSchema(name, description string, parameters map[string]any) map[string]any {
	// Remove title fields
	RemoveTitleField(parameters)

	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        name,
			"description": description,
			"parameters":  parameters,
		},
	}
}
