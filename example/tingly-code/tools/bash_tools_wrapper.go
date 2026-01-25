package tools

import (
	"context"
	"fmt"
)

// ExecuteBashParams holds the parameters for ExecuteBash
type ExecuteBashParams struct {
	Command  string `json:"command" required:"true"`
	Timeout int    `json:"timeout,omitempty"` // in seconds
}

// ExecuteBashTool is a type-safe wrapper for ExecuteBash
type ExecuteBashTool struct {
	bt *BashTools
}

func NewExecuteBashTool(bt *BashTools) *ExecuteBashTool {
	return &ExecuteBashTool{bt: bt}
}

func (t *ExecuteBashTool) Name() string {
	return "execute_bash"
}

func (t *ExecuteBashTool) Description() string {
	return "Run a shell command"
}

func (t *ExecuteBashTool) ParameterSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to execute",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds (default: 120)",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ExecuteBashTool) Call(ctx context.Context, params any) (string, error) {
	var p ExecuteBashParams
	if err := MapToStruct(params.(map[string]any), &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}

	// Convert to old-style kwargs for compatibility
	kwargs := make(map[string]any)
	kwargs["command"] = p.Command
	if p.Timeout > 0 {
		kwargs["timeout"] = float64(p.Timeout)
	}

	return t.bt.ExecuteBash(ctx, kwargs)
}

// JobDoneParams holds the parameters for JobDone
type JobDoneParams struct{}

// JobDoneTool is a type-safe wrapper for JobDone
type JobDoneTool struct {
	bt *BashTools
}

func NewJobDoneTool(bt *BashTools) *JobDoneTool {
	return &JobDoneTool{bt: bt}
}

func (t *JobDoneTool) Name() string {
	return "job_done"
}

func (t *JobDoneTool) Description() string {
	return "Mark the task as complete when you have successfully finished the user's request"
}

func (t *JobDoneTool) ParameterSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"description": "Call this when you have successfully finished the user's request",
	}
}

func (t *JobDoneTool) Call(ctx context.Context, params any) (string, error) {
	return t.bt.JobDone(ctx, map[string]any{})
}
