package agent

import (
	"context"
	"strings"
)

// AgentInjectors manages all injectors for the agent
// This allows easy extension with new injector types without changing function signatures
type AgentInjectors struct {
	taskInjector *TaskInjector
	// Future injectors can be added here, e.g.:
	// fileContextInjector  *FileContextInjector
	// sessionInjector      *SessionInjector
	// memoryInjector       *MemoryInjector
}

// NewAgentInjectors creates a new AgentInjectors with a task injector
func NewAgentInjectors(taskInjector *TaskInjector) *AgentInjectors {
	return &AgentInjectors{
		taskInjector: taskInjector,
	}
}

// InjectAll applies all injectors to the content in sequence
// Each injector receives the output of the previous injector
func (ai *AgentInjectors) InjectAll(ctx context.Context, content string) string {
	if ai == nil {
		return content
	}

	result := content

	// Apply task injector
	if ai.taskInjector != nil {
		result = ai.taskInjector.Inject(ctx, result)
	}

	// Future injectors are applied here in sequence
	// if ai.fileContextInjector != nil {
	//     result = ai.fileContextInjector.Inject(ctx, result)
	// }

	return result
}

// EnableTaskInjector enables the task injector
func (ai *AgentInjectors) EnableTaskInjector() {
	if ai.taskInjector != nil {
		ai.taskInjector.Enable()
	}
}

// DisableTaskInjector disables the task injector
func (ai *AgentInjectors) DisableTaskInjector() {
	if ai.taskInjector != nil {
		ai.taskInjector.Disable()
	}
}

// GetTaskInjector returns the task injector (for direct access if needed)
func (ai *AgentInjectors) GetTaskInjector() *TaskInjector {
	return ai.taskInjector
}

// HasTaskInjectors returns true if there are any active tasks
func (ai *AgentInjectors) HasTaskInjectors(ctx context.Context) bool {
	if ai.taskInjector == nil {
		return false
	}

	// Check if injecting would change the content
	testContent := ai.taskInjector.Inject(ctx, "")
	return strings.Contains(testContent, "# Task Progress")
}
