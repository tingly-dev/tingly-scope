package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Task represents a task in the task list
type Task struct {
	ID          string         `json:"id"`
	Subject     string         `json:"subject"`
	Description string         `json:"description"`
	ActiveForm  string         `json:"active_form,omitempty"`
	Status      string         `json:"status"` // pending, in_progress, completed
	Owner       string         `json:"owner,omitempty"`
	Blocks      []string       `json:"blocks,omitempty"`
	BlockedBy   []string       `json:"blocked_by,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// TaskStore manages task storage
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
	file  string
}

var (
	globalTaskStore   *TaskStore
	taskStoreOnce     sync.Once
	taskStoreFileLock sync.Mutex
)

// GetGlobalTaskStore returns the global task store (singleton)
func GetGlobalTaskStore() *TaskStore {
	taskStoreOnce.Do(func() {
		// Get working directory for task store file
		workDir := ""
		if dir, err := os.Getwd(); err == nil {
			workDir = dir
		}
		globalTaskStore = &TaskStore{
			tasks: make(map[string]*Task),
			file:  filepath.Join(workDir, ".tingly-tasks.json"),
		}
		globalTaskStore.load()
	})
	return globalTaskStore
}

// load loads tasks from file
func (ts *TaskStore) load() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	data, err := os.ReadFile(ts.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file yet, that's ok
		}
		return err
	}

	var tasks []*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return err
	}

	for _, task := range tasks {
		ts.tasks[task.ID] = task
	}

	return nil
}

// save saves tasks to file
func (ts *TaskStore) save() error {
	data, err := json.MarshalIndent(ts.tasks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ts.file, data, 0644)
}

// Add adds a new task to the store
func (ts *TaskStore) Add(task *Task) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.tasks[task.ID] = task
	return ts.save()
}

// Get gets a task by ID
func (ts *TaskStore) Get(id string) (*Task, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	task, ok := ts.tasks[id]
	return task, ok
}

// Update updates a task in the store
func (ts *TaskStore) Update(task *Task) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	task.UpdatedAt = time.Now()
	ts.tasks[task.ID] = task
	return ts.save()
}

// List returns all tasks
func (ts *TaskStore) List() []*Task {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make([]*Task, 0, len(ts.tasks))
	for _, task := range ts.tasks {
		result = append(result, task)
	}
	return result
}

// TaskManagementTools holds tools for task management
type TaskManagementTools struct {
	store *TaskStore
}

// NewTaskManagementTools creates a new TaskManagementTools instance
func NewTaskManagementTools() *TaskManagementTools {
	return &TaskManagementTools{
		store: GetGlobalTaskStore(),
	}
}

// Tool descriptions for task management tools
const (
	ToolDescTaskCreate = "Create a new task in the task list"
	ToolDescTaskGet    = "Get a task by ID from the task list"
	ToolDescTaskUpdate = "Update a task in the task list"
	ToolDescTaskList   = "List all tasks in the task list"
)

// TaskCreateParams holds parameters for TaskCreate
type TaskCreateParams struct {
	Subject     string         `json:"subject" required:"true" description:"Brief title for the task"`
	Description string         `json:"description" required:"true" description:"Detailed description of what needs to be done"`
	ActiveForm  string         `json:"active_form,omitempty" description:"Present continuous form for display"`
	Metadata    map[string]any `json:"metadata,omitempty" description:"Additional metadata"`
}

// TaskCreate creates a new task
func (tmt *TaskManagementTools) TaskCreate(ctx context.Context, params TaskCreateParams) (string, error) {
	// Generate unique ID
	id := fmt.Sprintf("task-%d", time.Now().UnixNano())

	task := &Task{
		ID:          id,
		Subject:     params.Subject,
		Description: params.Description,
		ActiveForm:  params.ActiveForm,
		Status:      "pending",
		Metadata:    params.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := tmt.store.Add(task); err != nil {
		return fmt.Sprintf("Error: failed to create task: %v", err), nil
	}

	return fmt.Sprintf("Task created with ID: %s\nSubject: %s", id, params.Subject), nil
}

// TaskGetParams holds parameters for TaskGet
type TaskGetParams struct {
	TaskID string `json:"task_id" required:"true" description:"ID of the task to retrieve"`
}

// TaskGet gets a task by ID
func (tmt *TaskManagementTools) TaskGet(ctx context.Context, params TaskGetParams) (string, error) {
	task, ok := tmt.store.Get(params.TaskID)
	if !ok {
		return fmt.Sprintf("Error: task '%s' not found", params.TaskID), nil
	}

	// Format task as JSON
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error: failed to format task: %v", err), nil
	}

	return string(data), nil
}

// TaskUpdateParams holds parameters for TaskUpdate
type TaskUpdateParams struct {
	TaskID       string         `json:"task_id" required:"true" description:"ID of the task to update"`
	Status       string         `json:"status,omitempty" description:"New status: pending, in_progress, completed"`
	Subject      string         `json:"subject,omitempty" description:"New subject"`
	Description  string         `json:"description,omitempty" description:"New description"`
	ActiveForm   string         `json:"active_form,omitempty" description:"New active form"`
	Owner        string         `json:"owner,omitempty" description:"New owner"`
	Metadata     map[string]any `json:"metadata,omitempty" description:"Metadata to merge"`
	AddBlocks    []string       `json:"add_blocks,omitempty" description:"Task IDs this task blocks"`
	AddBlockedBy []string       `json:"add_blocked_by,omitempty" description:"Task IDs that block this task"`
}

// TaskUpdate updates a task
func (tmt *TaskManagementTools) TaskUpdate(ctx context.Context, params TaskUpdateParams) (string, error) {
	task, ok := tmt.store.Get(params.TaskID)
	if !ok {
		return fmt.Sprintf("Error: task '%s' not found", params.TaskID), nil
	}

	// Update fields if provided
	if params.Status != "" {
		task.Status = params.Status
	}
	if params.Subject != "" {
		task.Subject = params.Subject
	}
	if params.Description != "" {
		task.Description = params.Description
	}
	if params.ActiveForm != "" {
		task.ActiveForm = params.ActiveForm
	}
	if params.Owner != "" {
		task.Owner = params.Owner
	}
	if params.Metadata != nil {
		if task.Metadata == nil {
			task.Metadata = make(map[string]any)
		}
		for k, v := range params.Metadata {
			if v == nil {
				delete(task.Metadata, k)
			} else {
				task.Metadata[k] = v
			}
		}
	}
	if len(params.AddBlocks) > 0 {
		task.Blocks = append(task.Blocks, params.AddBlocks...)
	}
	if len(params.AddBlockedBy) > 0 {
		task.BlockedBy = append(task.BlockedBy, params.AddBlockedBy...)
	}

	if err := tmt.store.Update(task); err != nil {
		return fmt.Sprintf("Error: failed to update task: %v", err), nil
	}

	return fmt.Sprintf("Task '%s' updated successfully", params.TaskID), nil
}

// TaskListParams holds parameters for TaskList
type TaskListParams struct{}

// TaskList lists all tasks
func (tmt *TaskManagementTools) TaskList(ctx context.Context, params TaskListParams) (string, error) {
	tasks := tmt.store.List()

	if len(tasks) == 0 {
		return "No tasks found.", nil
	}

	// Build summary
	var result []string
	result = append(result, "=== Task List ===\n")

	for _, task := range tasks {
		status := task.Status
		owner := task.Owner
		if owner == "" {
			owner = "unassigned"
		}
		blockedBy := ""
		if len(task.BlockedBy) > 0 {
			blockedBy = fmt.Sprintf(" (blocked by: %v)", task.BlockedBy)
		}
		result = append(result, fmt.Sprintf("[%s] %s: %s (owner: %s)%s",
			status, task.ID, task.Subject, owner, blockedBy))
	}

	return fmt.Sprintf("%s\nTotal: %d tasks", fmt.Sprintf("%s", result), len(tasks)), nil
}

func init() {
	// Register task management tools in the global registry
	RegisterTool("task_create", ToolDescTaskCreate, "Task Management", true)
	RegisterTool("task_get", ToolDescTaskGet, "Task Management", true)
	RegisterTool("task_update", ToolDescTaskUpdate, "Task Management", true)
	RegisterTool("task_list", ToolDescTaskList, "Task Management", true)
}
