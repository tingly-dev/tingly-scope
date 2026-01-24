package tools

import (
	"context"
	"fmt"
	"sync"
)

// BatchTool holds the tool registry for batch operations
type BatchTool struct {
	registry map[string]func(ctx context.Context, kwargs map[string]any) (string, error)
	mu       sync.RWMutex
}

// NewBatchTool creates a new BatchTool instance
func NewBatchTool() *BatchTool {
	return &BatchTool{
		registry: make(map[string]func(ctx context.Context, kwargs map[string]any) (string, error)),
	}
}

// Register registers a tool for batch execution
func (bt *BatchTool) Register(name string, fn func(ctx context.Context, kwargs map[string]any) (string, error)) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.registry[name] = fn
}

// GetRegistered returns all registered tools
func (bt *BatchTool) GetRegistered() map[string]func(ctx context.Context, kwargs map[string]any) (string, error) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	result := make(map[string]func(ctx context.Context, kwargs map[string]any) (string, error))
	for k, v := range bt.registry {
		result[k] = v
	}
	return result
}

// Invocation represents a single tool invocation in a batch
type Invocation struct {
	ToolName string                 `json:"tool_name"`
	Input    map[string]any         `json:"input"`
}

// InvocationResult represents the result of a single invocation
type InvocationResult struct {
	Status     string      `json:"status"` // "success" or "error"
	ToolName   string      `json:"tool_name"`
	Result     any         `json:"result"`
	Error      string      `json:"error,omitempty"`
}

// Batch executes multiple tool invocations in parallel
func (bt *BatchTool) Batch(ctx context.Context, description string, invocations []Invocation) (string, error) {
	if len(invocations) == 0 {
		return "No invocations provided", nil
	}

	var wg sync.WaitGroup
	results := make(chan InvocationResult, len(invocations))

	// Execute invocations in parallel
	for _, inv := range invocations {
		wg.Add(1)
		go func(invocation Invocation) {
			defer wg.Done()

			if invocation.ToolName == "" {
				results <- InvocationResult{
					Status:   "error",
					ToolName: "unknown",
					Error:    "Missing tool_name",
				}
				return
			}

			bt.mu.RLock()
			toolFunc, exists := bt.registry[invocation.ToolName]
			bt.mu.RUnlock()

			if !exists {
				results <- InvocationResult{
					Status:   "error",
					ToolName: invocation.ToolName,
					Error:    fmt.Sprintf("Tool not found: %s", invocation.ToolName),
				}
				return
			}

			result, err := toolFunc(ctx, invocation.Input)
			if err != nil {
				results <- InvocationResult{
					Status:   "error",
					ToolName: invocation.ToolName,
					Error:    err.Error(),
				}
				return
			}

			results <- InvocationResult{
				Status:   "success",
				ToolName: invocation.ToolName,
				Result:   result,
			}
		}(inv)
	}

	// Wait for all to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var successResults []InvocationResult
	var errors []string

	for result := range results {
		if result.Status == "error" {
			errors = append(errors, fmt.Sprintf("%s: %s", result.ToolName, result.Error))
		} else {
			successResults = append(successResults, result)
		}
	}

	// Format output
	var output []string
	output = append(output, fmt.Sprintf("Batch operation: %s", description))
	output = append(output, fmt.Sprintf("Completed: %d/%d", len(successResults), len(invocations)))

	if len(successResults) > 0 {
		output = append(output, "\n=== Results ===")
		for _, r := range successResults {
			output = append(output, fmt.Sprintf("\n%s:", r.ToolName))
			output = append(output, fmt.Sprintf("%v", r.Result))
		}
	}

	if len(errors) > 0 {
		output = append(output, "\n=== Errors ===")
		for _, e := range errors {
			output = append(output, fmt.Sprintf("- %s", e))
		}
	}

	return fmt.Sprintf("%s", output), nil
}

// Global batch tool instance
var (
	globalBatchTool *BatchTool
	batchToolOnce   sync.Once
)

// GetGlobalBatchTool returns the global batch tool (singleton)
func GetGlobalBatchTool() *BatchTool {
	batchToolOnce.Do(func() {
		globalBatchTool = NewBatchTool()
	})
	return globalBatchTool
}
