package agent

import (
	"context"
	"testing"

	"example/tingly-code/tools"
)

// TestTaskInjector tests the task injector functionality
func TestTaskInjector(t *testing.T) {
	ctx := context.Background()

	// Create a temporary task store for testing
	tempDir := t.TempDir()
	taskFile := tempDir + "/.tingly-tasks.json"
	store := tools.NewTaskStore(taskFile)

	// Create some test tasks
	task1 := &tools.Task{
		ID:          "task-1",
		Subject:     "Test task 1",
		Description: "First test task",
		Status:      "pending",
		ActiveForm:  "Testing task 1",
	}
	task2 := &tools.Task{
		ID:          "task-2",
		Subject:     "Test task 2",
		Description: "Second test task",
		Status:      "in_progress",
		ActiveForm:  "Testing task 2",
	}
	task3 := &tools.Task{
		ID:          "task-3",
		Subject:     "Test task 3",
		Description: "Third test task",
		Status:      "completed",
		ActiveForm:  "Testing task 3",
	}

	store.Add(task1)
	store.Add(task2)
	store.Add(task3)

	// Create injector
	injector := NewTaskInjector(store)

	// Test injection
	input := "Hello, what should I do next?"
	output := injector.Inject(ctx, input)

	// Verify injection happened
	if output == input {
		t.Fatal("Expected output to be different from input (injection should have happened)")
	}

	// Verify content contains task progress indicators
	if !contains(output, "# Task Progress") {
		t.Error("Expected output to contain '# Task Progress'")
	}
	if !contains(output, "Progress:") {
		t.Error("Expected output to contain 'Progress:'")
	}
	if !contains(output, "🔄") {
		t.Error("Expected output to contain in-progress task indicator")
	}
	if !contains(output, "⏳") {
		t.Error("Expected output to contain pending task indicator")
	}
	if !contains(output, "✅") {
		t.Error("Expected output to contain completed task indicator")
	}
	if !contains(output, "Testing task 2") {
		t.Error("Expected output to contain task 2 ActiveForm")
	}

	// Verify original message is preserved
	if !contains(output, input) {
		t.Error("Expected output to contain original message")
	}
}

// TestTaskInjectorEmpty tests injector with no tasks
func TestTaskInjectorEmpty(t *testing.T) {
	ctx := context.Background()

	// Create a temporary task store for testing
	tempDir := t.TempDir()
	taskFile := tempDir + "/.tingly-tasks.json"
	store := tools.NewTaskStore(taskFile)

	// Create injector
	injector := NewTaskInjector(store)

	// Test injection with no tasks
	input := "Hello, what should I do next?"
	output := injector.Inject(ctx, input)

	// With no tasks, output should equal input
	if output != input {
		t.Errorf("With no tasks, expected output to equal input, got:\n%s", output)
	}
}

// TestTaskInjectorDisabled tests disabled injector
func TestTaskInjectorDisabled(t *testing.T) {
	ctx := context.Background()

	// Create a temporary task store for testing
	tempDir := t.TempDir()
	taskFile := tempDir + "/.tingly-tasks.json"
	store := tools.NewTaskStore(taskFile)

	// Add a task
	task := &tools.Task{
		ID:      "task-1",
		Subject: "Test task",
		Status:  "in_progress",
	}
	store.Add(task)

	// Create injector and disable it
	injector := NewTaskInjector(store)
	injector.Disable()

	// Test injection
	input := "Hello"
	output := injector.Inject(ctx, input)

	// When disabled, output should equal input
	if output != input {
		t.Errorf("When disabled, expected output to equal input, got: %s", output)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
