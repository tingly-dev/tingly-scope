package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"example/tingly-code/tools"

	"github.com/tingly-dev/tingly-scope/pkg/message"
)

// TaskInjector injects task progress into messages
// Implements message.Injector for use with ReActAgent.InjectorChain
type TaskInjector struct {
	store   *tools.TaskStore
	mu      sync.RWMutex
	enabled bool
}

// NewTaskInjector creates a new task injector
func NewTaskInjector(store *tools.TaskStore) *TaskInjector {
	return &TaskInjector{
		store:   store,
		enabled: true,
	}
}

// Name returns the injector name (implements message.Injector)
func (ti *TaskInjector) Name() string {
	return "task_list"
}

// Inject adds task progress information to the message (implements message.Injector)
// This modifies the message content by prepending the task summary
func (ti *TaskInjector) Inject(ctx context.Context, msg *message.Msg) *message.Msg {
	if !ti.enabled {
		return msg
	}

	ti.mu.RLock()
	tasks := ti.store.List()
	ti.mu.RUnlock()

	if len(tasks) == 0 {
		return msg
	}

	summary := ti.formatTaskSummary(tasks)

	// Get original content blocks
	blocks := msg.GetContentBlocks()

	// Create new blocks with summary prepended
	newBlocks := make([]message.ContentBlock, 0, len(blocks)+1)
	newBlocks = append(newBlocks, message.Text(summary))
	newBlocks = append(newBlocks, blocks...)

	// Create a new message with the injected content
	// preserving all original properties
	injectedMsg := message.NewMsgWithTimestamp(
		msg.Name,
		newBlocks,
		msg.Role,
		msg.Timestamp,
	)
	injectedMsg.ID = msg.ID
	injectedMsg.Metadata = msg.Metadata
	injectedMsg.InvocationID = msg.InvocationID

	return injectedMsg
}

// Enable enables the injector
func (ti *TaskInjector) Enable() {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.enabled = true
}

// Disable disables the injector
func (ti *TaskInjector) Disable() {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.enabled = false
}

// HasTasks returns true if there are any tasks in the store
func (ti *TaskInjector) HasTasks() bool {
	ti.mu.RLock()
	defer ti.mu.RUnlock()
	tasks := ti.store.List()
	return len(tasks) > 0
}

// formatTaskSummary formats the task list as a system reminder
func (ti *TaskInjector) formatTaskSummary(tasks []*tools.Task) string {
	var parts []string
	parts = append(parts, "<system-reminder>")
	parts = append(parts, "# Task Progress")

	// Calculate statistics
	total := len(tasks)
	completed := 0
	inProgress := 0
	for _, task := range tasks {
		switch task.Status {
		case "completed":
			completed++
		case "in_progress":
			inProgress++
		}
	}

	// Summary line
	percent := 0
	if total > 0 {
		percent = (completed * 100) / total
	}
	parts = append(parts, fmt.Sprintf("**Progress:** %d/%d completed (%d%%)", completed, total, percent))

	// Current in-progress task
	var currentTask *tools.Task
	for _, task := range tasks {
		if task.Status == "in_progress" {
			currentTask = task
			break
		}
	}

	if currentTask != nil {
		parts = append(parts, "\n## Current Task")
		activeForm := currentTask.ActiveForm
		if activeForm == "" {
			activeForm = currentTask.Subject
		}
		parts = append(parts, fmt.Sprintf("🔄 **%s:** %s", currentTask.ID, activeForm))
		if currentTask.Description != "" {
			parts = append(parts, fmt.Sprintf("   *%s*", currentTask.Description))
		}
	}

	// Pending tasks
	var pendingTasks []*tools.Task
	for _, task := range tasks {
		if task.Status == "pending" {
			pendingTasks = append(pendingTasks, task)
		}
	}

	if len(pendingTasks) > 0 {
		parts = append(parts, "\n## Pending Tasks")
		for _, task := range pendingTasks {
			blockedInfo := ""
			if len(task.BlockedBy) > 0 {
				blockedInfo = fmt.Sprintf(" (blocked by: %s)", strings.Join(task.BlockedBy, ", "))
			}
			parts = append(parts, fmt.Sprintf("⏳ **%s:** %s%s", task.ID, task.Subject, blockedInfo))
		}
	}

	// Completed tasks (show last 3)
	if completed > 0 {
		parts = append(parts, "\n## Recently Completed")
		count := 0
		for i := len(tasks) - 1; i >= 0 && count < 3; i-- {
			if tasks[i].Status == "completed" {
				parts = append(parts, fmt.Sprintf("✅ **%s:** %s", tasks[i].ID, tasks[i].Subject))
				count++
			}
		}
		if completed > 3 {
			parts = append(parts, fmt.Sprintf("   ... and %d more", completed-3))
		}
	}

	parts = append(parts, "</system-reminder>")

	return strings.Join(parts, "\n")
}
