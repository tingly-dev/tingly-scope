package tools

import (
	"context"
)

// Tool descriptions for bash tools
const (
	ToolDescExecuteBash = "Run shell commands with timeout (avoid for file operations)"
	ToolDescJobDone     = "Mark the task as complete when finished"
)

// BashTools wraps bash-related tools
type BashTools struct {
	session *BashSession
}

// NewBashTools creates a new BashTools instance
func NewBashTools(session *BashSession) *BashTools {
	if session == nil {
		session = GetGlobalBashSession()
	}
	return &BashTools{
		session: session,
	}
}

// ExecuteBashParams holds parameters for ExecuteBash
type ExecuteBashParams struct {
	Command string `json:"command" required:"true" description="Shell command to execute"`
	Timeout int    `json:"timeout,omitempty" description:"Timeout in seconds (default: 120)"`
}

// ExecuteBash runs a shell command with timeout
func (bt *BashTools) ExecuteBash(ctx context.Context, params ExecuteBashParams) (string, error) {
	return bt.session.executeBashInternal(ctx, params.Command, params.Timeout)
}

// JobDoneParams holds parameters for JobDone
type JobDoneParams struct{}

// JobDone marks the task as complete
func (bt *BashTools) JobDone(ctx context.Context, params JobDoneParams) (string, error) {
	return "Task completed successfully", nil
}

// GetSession returns the bash session
func (bt *BashTools) GetSession() *BashSession {
	return bt.session
}
