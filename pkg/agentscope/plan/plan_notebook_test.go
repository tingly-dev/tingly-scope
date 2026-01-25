package plan

import (
	"context"
	"testing"
)

func TestNewSubTask(t *testing.T) {
	tests := []struct {
		name             string
		subtaskName      string
		description      string
		expectedOutcome  string
		wantIDNotEmpty   bool
		wantState        SubTaskState
	}{
		{
			name:             "create basic subtask",
			subtaskName:      "Design API",
			description:      "Design REST API endpoints",
			expectedOutcome:  "API specification document",
			wantIDNotEmpty:   true,
			wantState:        SubTaskStateTodo,
		},
		{
			name:             "create subtask with empty fields",
			subtaskName:      "Simple task",
			description:      "",
			expectedOutcome:  "",
			wantIDNotEmpty:   true,
			wantState:        SubTaskStateTodo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := NewSubTask(tt.subtaskName, tt.description, tt.expectedOutcome)

			if st.Name != tt.subtaskName {
				t.Errorf("Name = %v, want %v", st.Name, tt.subtaskName)
			}
			if st.Description != tt.description {
				t.Errorf("Description = %v, want %v", st.Description, tt.description)
			}
			if st.ExpectedOutcome != tt.expectedOutcome {
				t.Errorf("ExpectedOutcome = %v, want %v", st.ExpectedOutcome, tt.expectedOutcome)
			}
			if tt.wantIDNotEmpty && st.ID == "" {
				t.Error("ID should not be empty")
			}
			if st.State != tt.wantState {
				t.Errorf("State = %v, want %v", st.State, tt.wantState)
			}
			if st.CreatedAt == "" {
				t.Error("CreatedAt should not be empty")
			}
		})
	}
}

func TestSubTask_Finish(t *testing.T) {
	st := NewSubTask("Test task", "Description", "Expected outcome")

	st.Finish("Task completed successfully")

	if st.State != SubTaskStateDone {
		t.Errorf("State = %v, want %v", st.State, SubTaskStateDone)
	}
	if st.Outcome != "Task completed successfully" {
		t.Errorf("Outcome = %v, want %v", st.Outcome, "Task completed successfully")
	}
	if st.FinishedAt == "" {
		t.Error("FinishedAt should not be empty after Finish")
	}
}

func TestSubTask_Abandon(t *testing.T) {
	st := NewSubTask("Test task", "Description", "Expected outcome")

	st.Abandon()

	if st.State != SubTaskStateAbandoned {
		t.Errorf("State = %v, want %v", st.State, SubTaskStateAbandoned)
	}
	if st.FinishedAt == "" {
		t.Error("FinishedAt should not be empty after Abandon")
	}
}

func TestSubTask_SetInProgress(t *testing.T) {
	st := NewSubTask("Test task", "Description", "Expected outcome")

	st.SetInProgress()

	if st.State != SubTaskStateInProgress {
		t.Errorf("State = %v, want %v", st.State, SubTaskStateInProgress)
	}
}

func TestSubTask_ToMarkdown(t *testing.T) {
	tests := []struct {
		name      string
		subtask   *SubTask
		detailed   bool
		wantEmpty bool
	}{
		{
			name: "simple markdown",
			subtask: NewSubTask("Task name", "Task description", "Expected result"),
			detailed: false,
			wantEmpty: false,
		},
		{
			name: "detailed markdown for todo",
			subtask: func() *SubTask {
				st := NewSubTask("Task name", "Task description", "Expected result")
				return st
			}(),
			detailed: true,
			wantEmpty: false,
		},
		{
			name: "detailed markdown for done",
			subtask: func() *SubTask {
				st := NewSubTask("Task name", "Task description", "Expected result")
				st.Finish("Completed")
				return st
			}(),
			detailed: true,
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := tt.subtask.ToMarkdown(tt.detailed)
			if tt.wantEmpty && md != "" {
				t.Errorf("ToMarkdown() = %v, want empty", md)
			}
			if !tt.wantEmpty && md == "" {
				t.Error("ToMarkdown() should not be empty")
			}
			// Should contain task name
			if !tt.wantEmpty && !contains(md, tt.subtask.Name) {
				t.Errorf("ToMarkdown() should contain task name %v", tt.subtask.Name)
			}
		})
	}
}

func TestNewPlan(t *testing.T) {
	plan := NewPlan("Build API", "Create REST API", "Working API")

	if plan.Name != "Build API" {
		t.Errorf("Name = %v, want %v", plan.Name, "Build API")
	}
	if plan.State != PlanStateTodo {
		t.Errorf("State = %v, want %v", plan.State, PlanStateTodo)
	}
	if len(plan.SubTasks) != 0 {
		t.Errorf("SubTasks length = %v, want 0", len(plan.SubTasks))
	}
}

func TestPlan_AddSubTask(t *testing.T) {
	plan := NewPlan("Test plan", "Description", "Expected outcome")

	st := NewSubTask("Subtask 1", "Description 1", "Outcome 1")
	plan.AddSubTask(st)

	if len(plan.SubTasks) != 1 {
		t.Errorf("SubTasks length = %v, want 1", len(plan.SubTasks))
	}
	if plan.SubTasks[0] != st {
		t.Error("First subtask should match the added one")
	}
}

func TestPlan_RefreshState(t *testing.T) {
	t.Run("from todo to in_progress", func(t *testing.T) {
		plan := NewPlan("Test plan", "Description", "Expected outcome")
		st := NewSubTask("Task 1", "Description", "Outcome")
		plan.AddSubTask(st)

		st.SetInProgress()
		msg := plan.RefreshState()

		if plan.State != PlanStateInProgress {
			t.Errorf("State = %v, want %v", plan.State, PlanStateInProgress)
		}
		if msg == "" {
			t.Error("RefreshState() should return a message")
		}
	})

	t.Run("from in_progress back to todo", func(t *testing.T) {
		plan := NewPlan("Test plan", "Description", "Expected outcome")
		st := NewSubTask("Task 1", "Description", "Outcome")
		plan.AddSubTask(st)

		st.SetInProgress()
		plan.RefreshState()

		st.Finish("Done")
		msg := plan.RefreshState()

		if plan.State != PlanStateTodo {
			t.Errorf("State = %v, want %v", plan.State, PlanStateTodo)
		}
		if msg == "" {
			t.Error("RefreshState() should return a message")
		}
	})
}

func TestPlan_Finish(t *testing.T) {
	plan := NewPlan("Test plan", "Description", "Expected outcome")

	plan.Finish(PlanStateDone, "Plan completed successfully")

	if plan.State != PlanStateDone {
		t.Errorf("State = %v, want %v", plan.State, PlanStateDone)
	}
	if plan.Outcome != "Plan completed successfully" {
		t.Errorf("Outcome = %v, want %v", plan.Outcome, "Plan completed successfully")
	}
	if plan.FinishedAt == "" {
		t.Error("FinishedAt should not be empty")
	}
}

func TestPlan_ToMarkdown(t *testing.T) {
	plan := NewPlan("Test Plan", "Plan description", "Expected result")
	plan.AddSubTask(NewSubTask("Task 1", "Description 1", "Outcome 1"))
	plan.AddSubTask(NewSubTask("Task 2", "Description 2", "Outcome 2"))

	md := plan.ToMarkdown(false)

	if !contains(md, "# Test Plan") {
		t.Error("ToMarkdown() should contain plan name")
	}
	if !contains(md, "Plan description") {
		t.Error("ToMarkdown() should contain description")
	}
	if !contains(md, "## Subtasks") {
		t.Error("ToMarkdown() should contain subtasks section")
	}
	if !contains(md, "Task 1") {
		t.Error("ToMarkdown() should contain first task")
	}
	if !contains(md, "Task 2") {
		t.Error("ToMarkdown() should contain second task")
	}
}

func TestPlan_GetSubTaskIndex(t *testing.T) {
	plan := NewPlan("Test plan", "Description", "Expected outcome")
	st1 := NewSubTask("Task 1", "Description 1", "Outcome 1")
	st2 := NewSubTask("Task 2", "Description 2", "Outcome 2")
	plan.AddSubTask(st1)
	plan.AddSubTask(st2)

	t.Run("find existing subtask", func(t *testing.T) {
		idx, err := plan.GetSubTaskIndex(st1.ID)
		if err != nil {
			t.Errorf("GetSubTaskIndex() error = %v", err)
		}
		if idx != 0 {
			t.Errorf("GetSubTaskIndex() = %v, want 0", idx)
		}
	})

	t.Run("subtask not found", func(t *testing.T) {
		_, err := plan.GetSubTaskIndex("non-existent-id")
		if err == nil {
			t.Error("GetSubTaskIndex() should return error for non-existent ID")
		}
	})
}

func TestPlan_GetInProgressSubTaskIndex(t *testing.T) {
	plan := NewPlan("Test plan", "Description", "Expected outcome")
	st1 := NewSubTask("Task 1", "Description 1", "Outcome 1")
	st2 := NewSubTask("Task 2", "Description 2", "Outcome 2")
	plan.AddSubTask(st1)
	plan.AddSubTask(st2)

	t.Run("no subtask in progress", func(t *testing.T) {
		_, err := plan.GetInProgressSubTaskIndex()
		if err == nil {
			t.Error("GetInProgressSubTaskIndex() should return error when no subtask in progress")
		}
	})

	t.Run("find in progress subtask", func(t *testing.T) {
		st2.SetInProgress()
		idx, err := plan.GetInProgressSubTaskIndex()
		if err != nil {
			t.Errorf("GetInProgressSubTaskIndex() error = %v", err)
		}
		if idx != 1 {
			t.Errorf("GetInProgressSubTaskIndex() = %v, want 1", idx)
		}
	})
}

func TestInMemoryPlanStorage(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryPlanStorage()

	t.Run("save and load plan", func(t *testing.T) {
		plan := NewPlan("Test plan", "Description", "Expected outcome")

		err := storage.Save(ctx, plan)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		loaded, err := storage.Load(ctx, plan.ID)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if loaded.ID != plan.ID {
			t.Errorf("loaded ID = %v, want %v", loaded.ID, plan.ID)
		}
		if loaded.Name != plan.Name {
			t.Errorf("loaded Name = %v, want %v", loaded.Name, plan.Name)
		}
	})

	t.Run("load non-existent plan", func(t *testing.T) {
		_, err := storage.Load(ctx, "non-existent-id")
		if err == nil {
			t.Error("Load() should return error for non-existent plan")
		}
	})

	t.Run("delete plan", func(t *testing.T) {
		plan := NewPlan("To delete", "Description", "Expected outcome")
		storage.Save(ctx, plan)

		err := storage.Delete(ctx, plan.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		_, err = storage.Load(ctx, plan.ID)
		if err == nil {
			t.Error("Load() should return error after Delete()")
		}
	})

	t.Run("list plans", func(t *testing.T) {
		storage := NewInMemoryPlanStorage()
		plan1 := NewPlan("Plan 1", "Description 1", "Outcome 1")
		plan2 := NewPlan("Plan 2", "Description 2", "Outcome 2")

		storage.Save(ctx, plan1)
		storage.Save(ctx, plan2)

		plans, err := storage.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(plans) != 2 {
			t.Errorf("List() returned %v plans, want 2", len(plans))
		}
	})
}

func TestNewPlanNotebook(t *testing.T) {
	storage := NewInMemoryPlanStorage()
	notebook := NewPlanNotebook(storage)

	if notebook == nil {
		t.Fatal("NewPlanNotebook() should not return nil")
	}
}

func TestPlanNotebook_CreatePlan(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryPlanStorage()
	notebook := NewPlanNotebook(storage)

	subtasks := []*SubTask{
		NewSubTask("Task 1", "Description 1", "Outcome 1"),
		NewSubTask("Task 2", "Description 2", "Outcome 2"),
	}

	plan, err := notebook.CreatePlan(ctx, "Test Plan", "Plan description", "Expected outcome", subtasks)
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}

	if plan.Name != "Test Plan" {
		t.Errorf("Plan Name = %v, want %v", plan.Name, "Test Plan")
	}
	if len(plan.SubTasks) != 2 {
		t.Errorf("Plan SubTasks length = %v, want 2", len(plan.SubTasks))
	}

	// Verify plan is set as current
	current := notebook.GetCurrentPlan()
	if current == nil {
		t.Error("GetCurrentPlan() should not be nil after CreatePlan()")
	}
	if current.ID != plan.ID {
		t.Errorf("GetCurrentPlan() ID = %v, want %v", current.ID, plan.ID)
	}
}

func TestPlanNotebook_UpdateSubtaskState(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryPlanStorage()
	notebook := NewPlanNotebook(storage)

	subtasks := []*SubTask{
		NewSubTask("Task 1", "Description 1", "Outcome 1"),
	}
	plan, _ := notebook.CreatePlan(ctx, "Test Plan", "Description", "Outcome", subtasks)

	t.Run("set in progress", func(t *testing.T) {
		err := notebook.UpdateSubtaskState(ctx, plan.SubTasks[0].ID, SubTaskStateInProgress)
		if err != nil {
			t.Fatalf("UpdateSubtaskState() error = %v", err)
		}

		current := notebook.GetCurrentPlan()
		if current.SubTasks[0].State != SubTaskStateInProgress {
			t.Errorf("SubTask State = %v, want %v", current.SubTasks[0].State, SubTaskStateInProgress)
		}
	})

	t.Run("finish subtask", func(t *testing.T) {
		err := notebook.FinishSubtask(ctx, plan.SubTasks[0].ID, "Task completed")
		if err != nil {
			t.Fatalf("FinishSubtask() error = %v", err)
		}

		current := notebook.GetCurrentPlan()
		if current.SubTasks[0].State != SubTaskStateDone {
			t.Errorf("SubTask State = %v, want %v", current.SubTasks[0].State, SubTaskStateDone)
		}
		if current.SubTasks[0].Outcome != "Task completed" {
			t.Errorf("SubTask Outcome = %v, want %v", current.SubTasks[0].Outcome, "Task completed")
		}
	})

	t.Run("update non-existent subtask", func(t *testing.T) {
		err := notebook.UpdateSubtaskState(ctx, "non-existent-id", SubTaskStateInProgress)
		if err == nil {
			t.Error("UpdateSubtaskState() should return error for non-existent ID")
		}
	})
}

func TestPlanNotebook_FinishPlan(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryPlanStorage()
	notebook := NewPlanNotebook(storage)

	subtasks := []*SubTask{
		NewSubTask("Task 1", "Description 1", "Outcome 1"),
	}
	plan, _ := notebook.CreatePlan(ctx, "Test Plan", "Description", "Outcome", subtasks)

	// Mark all subtasks as done
	notebook.FinishSubtask(ctx, plan.SubTasks[0].ID, "Done")

	err := notebook.FinishPlan(ctx, PlanStateDone, "Plan completed successfully")
	if err != nil {
		t.Fatalf("FinishPlan() error = %v", err)
	}

	current := notebook.GetCurrentPlan()
	if current.State != PlanStateDone {
		t.Errorf("Plan State = %v, want %v", current.State, PlanStateDone)
	}
	if current.Outcome != "Plan completed successfully" {
		t.Errorf("Plan Outcome = %v, want %v", current.Outcome, "Plan completed successfully")
	}
}

func TestPlanNotebook_GenerateHint(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryPlanStorage()
	notebook := NewPlanNotebook(storage)

	t.Run("hint when no plan", func(t *testing.T) {
		hint := notebook.GenerateHint()
		if hint == "" {
			t.Error("GenerateHint() should not be empty when no plan")
		}
		if !contains(hint, "system-hint") {
			t.Error("GenerateHint() should contain system-hint tag")
		}
	})

	t.Run("hint when plan created but no subtask in progress", func(t *testing.T) {
		subtasks := []*SubTask{
			NewSubTask("Task 1", "Description 1", "Outcome 1"),
			NewSubTask("Task 2", "Description 2", "Outcome 2"),
		}
		notebook.CreatePlan(ctx, "Test Plan", "Description", "Outcome", subtasks)

		hint := notebook.GenerateHint()
		if !contains(hint, "system-hint") {
			t.Error("GenerateHint() should contain system-hint tag")
		}
		if !contains(hint, "Test Plan") {
			t.Error("GenerateHint() should contain plan name")
		}
	})

	t.Run("hint when subtask in progress", func(t *testing.T) {
		subtasks := []*SubTask{
			NewSubTask("Task 1", "Description 1", "Outcome 1"),
		}
		plan, _ := notebook.CreatePlan(ctx, "Test Plan 2", "Description", "Outcome", subtasks)
		notebook.UpdateSubtaskState(ctx, plan.SubTasks[0].ID, SubTaskStateInProgress)

		hint := notebook.GenerateHint()
		if !contains(hint, "in_progress") {
			t.Error("GenerateHint() should mention in_progress state")
		}
		if !contains(hint, "Task 1") {
			t.Error("GenerateHint() should contain subtask name")
		}
	})

	t.Run("hint when all subtasks done", func(t *testing.T) {
		subtasks := []*SubTask{
			NewSubTask("Task 1", "Description 1", "Outcome 1"),
		}
		plan, _ := notebook.CreatePlan(ctx, "Test Plan 3", "Description", "Outcome", subtasks)
		notebook.FinishSubtask(ctx, plan.SubTasks[0].ID, "Done")

		hint := notebook.GenerateHint()
		if !contains(hint, "done") {
			t.Error("GenerateHint() should mention completion")
		}
	})
}

func TestPlanNotebook_RevisePlan(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryPlanStorage()
	notebook := NewPlanNotebook(storage)

	subtasks := []*SubTask{
		NewSubTask("Task 1", "Description 1", "Outcome 1"),
	}
	_, _ = notebook.CreatePlan(ctx, "Original Plan", "Description", "Outcome", subtasks)

	// Add another subtask and revise
	newSubtasks := append(subtasks, NewSubTask("Task 2", "Description 2", "Outcome 2"))
	err := notebook.RevisePlan(ctx, "Revised Plan", "New description", "New outcome", newSubtasks)
	if err != nil {
		t.Fatalf("RevisePlan() error = %v", err)
	}

	current := notebook.GetCurrentPlan()
	if current.Name != "Revised Plan" {
		t.Errorf("Plan Name = %v, want %v", current.Name, "Revised Plan")
	}
	if len(current.SubTasks) != 2 {
		t.Errorf("Plan SubTasks length = %v, want 2", len(current.SubTasks))
	}
}

func TestPlanNotebook_DeletePlan(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryPlanStorage()
	notebook := NewPlanNotebook(storage)

	subtasks := []*SubTask{
		NewSubTask("Task 1", "Description 1", "Outcome 1"),
	}
	plan, _ := notebook.CreatePlan(ctx, "To delete", "Description", "Outcome", subtasks)

	err := notebook.DeletePlan(ctx, plan.ID)
	if err != nil {
		t.Fatalf("DeletePlan() error = %v", err)
	}

	current := notebook.GetCurrentPlan()
	if current != nil {
		t.Error("GetCurrentPlan() should be nil after deleting current plan")
	}
}

func TestPlanNotebook_StateDict(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryPlanStorage()
	notebook := NewPlanNotebook(storage)

	t.Run("state dict when no plan", func(t *testing.T) {
		state := notebook.StateDict()
		if state["has_current_plan"] != false {
			t.Errorf("has_current_plan = %v, want false", state["has_current_plan"])
		}
	})

	t.Run("state dict with plan", func(t *testing.T) {
		subtasks := []*SubTask{
			NewSubTask("Task 1", "Description 1", "Outcome 1"),
		}
		notebook.CreatePlan(ctx, "Test Plan", "Description", "Outcome", subtasks)

		state := notebook.StateDict()
		if state["has_current_plan"] != true {
			t.Errorf("has_current_plan = %v, want true", state["has_current_plan"])
		}
		if state["subtask_count"] != 1 {
			t.Errorf("subtask_count = %v, want 1", state["subtask_count"])
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
