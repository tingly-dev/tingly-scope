package tools

import (
	"sort"
	"strings"
	"sync"
)

// ToolDescriptor describes a tool
type ToolDescriptor struct {
	Name        string
	Description string
	DefaultOn   bool   // default enabled state
	Category    string // tool category for grouping
}

// ToolRegistry maintains a global registry of all available tools
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*ToolDescriptor
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolDescriptor),
	}
}

// RegisterTool registers a tool descriptor
func (tr *ToolRegistry) RegisterTool(name, description, category string, defaultOn bool) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.tools[name] = &ToolDescriptor{
		Name:        name,
		Description: description,
		DefaultOn:   defaultOn,
		Category:    category,
	}
}

// ListTools returns all registered tools, sorted by name
func (tr *ToolRegistry) ListTools() []*ToolDescriptor {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make([]*ToolDescriptor, 0, len(tr.tools))
	for _, td := range tr.tools {
		result = append(result, td)
	}

	// Sort by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// GetTool returns a tool descriptor by name
func (tr *ToolRegistry) GetTool(name string) (*ToolDescriptor, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	td, ok := tr.tools[name]
	return td, ok
}

// ListToolsByCategory returns tools grouped by category
func (tr *ToolRegistry) ListToolsByCategory() map[string][]*ToolDescriptor {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make(map[string][]*ToolDescriptor)
	for _, td := range tr.tools {
		result[td.Category] = append(result[td.Category], td)
	}

	// Sort tools within each category
	for category := range result {
		sort.Slice(result[category], func(i, j int) bool {
			return result[category][i].Name < result[category][j].Name
		})
	}

	return result
}

// global registry instance
var globalRegistry = NewToolRegistry()

// RegisterTool registers a tool in the global registry
// This is typically called from init() functions in tool files
func RegisterTool(name, description, category string, defaultOn bool) {
	globalRegistry.RegisterTool(name, description, category, defaultOn)
}

// ListTools returns all registered tools from the global registry
func ListTools() []*ToolDescriptor {
	return globalRegistry.ListTools()
}

// ListToolsByCategory returns tools grouped by category from the global registry
func ListToolsByCategory() map[string][]*ToolDescriptor {
	return globalRegistry.ListToolsByCategory()
}

// GetTool returns a tool descriptor by name from the global registry
func GetTool(name string) (*ToolDescriptor, bool) {
	return globalRegistry.GetTool(name)
}

// FormatToolStatus returns a formatted status string for a tool
// status = true for enabled, false for disabled
func FormatToolStatus(name, description string, enabled bool) string {
	if enabled {
		return "  ✓ " + padRight(name, 18) + description
	}
	return "  ✗ " + padRight(name, 18) + description + " [DISABLED]"
}

// padRight pads a string to the right with spaces
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// GetToolCategories returns all unique tool categories
func GetToolCategories() []string {
	cats := globalRegistry.ListToolsByCategory()
	result := make([]string, 0, len(cats))
	for cat := range cats {
		result = append(result, cat)
	}
	sort.Strings(result)
	return result
}
