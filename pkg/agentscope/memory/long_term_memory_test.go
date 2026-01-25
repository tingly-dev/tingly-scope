package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultLongTermMemoryConfig(t *testing.T) {
	config := DefaultLongTermMemoryConfig("./test_memory")

	if config.StoragePath != "./test_memory" {
		t.Errorf("StoragePath = %v, want %v", config.StoragePath, "./test_memory")
	}
	if config.MaxEntries != 1000 {
		t.Errorf("MaxEntries = %v, want %v", config.MaxEntries, 1000)
	}
	if config.TTL != 0 {
		t.Errorf("TTL = %v, want 0", config.TTL)
	}
}

func TestNewLongTermMemory(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test_memory")
	defer os.RemoveAll(tempDir)

	config := &LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  10,
		TTL:         0,
	}

	ltm, err := NewLongTermMemory(config)
	if err != nil {
		t.Fatalf("NewLongTermMemory() error = %v", err)
	}

	if ltm == nil {
		t.Fatal("NewLongTermMemory() should not return nil")
	}
}

func TestLongTermMemory_Add(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_memory")
	defer os.RemoveAll(tempDir)

	ltm, _ := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  10,
	})

	t.Run("add memory entry", func(t *testing.T) {
		id, err := ltm.Add(ctx, "test_type", "Test content", nil)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
		if id == "" {
			t.Error("Add() should return non-empty ID")
		}
	})

	t.Run("add with metadata", func(t *testing.T) {
		metadata := map[string]any{
			"priority": "high",
			"tags":     []string{"test", "important"},
		}
		id, err := ltm.Add(ctx, "test_type", "Test content with metadata", metadata)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
		if id == "" {
			t.Error("Add() should return non-empty ID")
		}
	})
}

func TestLongTermMemory_Get(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_memory")
	defer os.RemoveAll(tempDir)

	ltm, _ := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  10,
	})

	id, _ := ltm.Add(ctx, "test_type", "Test content", map[string]any{"key": "value"})

	t.Run("get existing memory", func(t *testing.T) {
		entry, err := ltm.Get(ctx, "test_type", id)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if entry.Content != "Test content" {
			t.Errorf("Content = %v, want %v", entry.Content, "Test content")
		}
		if entry.Metadata["key"] != "value" {
			t.Errorf("Metadata key = %v, want %v", entry.Metadata["key"], "value")
		}
	})

	t.Run("get non-existent memory", func(t *testing.T) {
		_, err := ltm.Get(ctx, "test_type", "non-existent-id")
		if err == nil {
			t.Error("Get() should return error for non-existent ID")
		}
	})

	t.Run("get from non-existent type", func(t *testing.T) {
		_, err := ltm.Get(ctx, "non_existent_type", "id")
		if err == nil {
			t.Error("Get() should return error for non-existent type")
		}
	})
}

func TestLongTermMemory_Search(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_memory")
	defer os.RemoveAll(tempDir)

	ltm, _ := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  10,
	})

	// Add some test memories
	ltm.Add(ctx, "user_prefs", "User prefers dark mode", nil)
	ltm.Add(ctx, "user_prefs", "User likes compact layout", nil)
	ltm.Add(ctx, "project", "Working on Go project", nil)

	t.Run("search with matching query", func(t *testing.T) {
		results, err := ltm.Search(ctx, "user_prefs", "dark", 10)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(results) == 0 {
			t.Error("Search() should find results for 'dark'")
		}
	})

	t.Run("search with no matches", func(t *testing.T) {
		results, err := ltm.Search(ctx, "user_prefs", "nonexistent", 10)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Search() should return empty results, got %v", len(results))
		}
	})

	t.Run("search with empty query returns all", func(t *testing.T) {
		results, err := ltm.Search(ctx, "user_prefs", "", 10)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Search() with empty query should return all entries, got %v", len(results))
		}
	})

	t.Run("search with limit", func(t *testing.T) {
		results, err := ltm.Search(ctx, "user_prefs", "", 1)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(results) > 1 {
			t.Errorf("Search() with limit 1 should return at most 1 result, got %v", len(results))
		}
	})
}

func TestLongTermMemory_GetRecent(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_memory")
	defer os.RemoveAll(tempDir)

	ltm, _ := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  10,
	})

	// Add memories in order
	ltm.Add(ctx, "test", "First message", nil)
	time.Sleep(10 * time.Millisecond)
	ltm.Add(ctx, "test", "Second message", nil)
	time.Sleep(10 * time.Millisecond)
	ltm.Add(ctx, "test", "Third message", nil)

	t.Run("get recent memories", func(t *testing.T) {
		results, err := ltm.GetRecent(ctx, "test", 2)
		if err != nil {
			t.Fatalf("GetRecent() error = %v", err)
		}
		if len(results) != 2 {
			t.Errorf("GetRecent() should return 2 results, got %v", len(results))
		}
		if !contains(results[0].Content, "Third") {
			t.Error("Most recent should be 'Third message'")
		}
	})

	t.Run("get recent more than available", func(t *testing.T) {
		results, err := ltm.GetRecent(ctx, "test", 10)
		if err != nil {
			t.Fatalf("GetRecent() error = %v", err)
		}
		if len(results) != 3 {
			t.Errorf("GetRecent() should return all 3 results, got %v", len(results))
		}
	})

	t.Run("get recent from non-existent type", func(t *testing.T) {
		results, err := ltm.GetRecent(ctx, "non_existent", 5)
		if err != nil {
			t.Fatalf("GetRecent() error = %v", err)
		}
		if len(results) != 0 {
			t.Errorf("GetRecent() should return empty for non-existent type, got %v", len(results))
		}
	})
}

func TestLongTermMemory_Delete(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_memory")
	defer os.RemoveAll(tempDir)

	ltm, _ := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  10,
	})

	id, _ := ltm.Add(ctx, "test", "To delete", nil)

	t.Run("delete existing memory", func(t *testing.T) {
		err := ltm.Delete(ctx, "test", id)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		_, err = ltm.Get(ctx, "test", id)
		if err == nil {
			t.Error("Get() should return error after Delete()")
		}
	})

	t.Run("delete non-existent memory", func(t *testing.T) {
		err := ltm.Delete(ctx, "test", "non-existent-id")
		if err == nil {
			t.Error("Delete() should return error for non-existent ID")
		}
	})
}

func TestLongTermMemory_Clear(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_memory")
	defer os.RemoveAll(tempDir)

	ltm, _ := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  10,
	})

	ltm.Add(ctx, "test", "Memory 1", nil)
	ltm.Add(ctx, "test", "Memory 2", nil)

	err := ltm.Clear(ctx, "test")
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	results, _ := ltm.GetRecent(ctx, "test", 10)
	if len(results) != 0 {
		t.Errorf("GetRecent() should return empty after Clear(), got %v", len(results))
	}
}

func TestLongTermMemory_MaxEntries(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_memory_max")
	defer os.RemoveAll(tempDir)

	ltm, _ := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  3,
	})

	// Add more entries than max
	for i := 0; i < 5; i++ {
		ltm.Add(ctx, "test", "Memory "+string(rune('A'+i)), nil)
	}

	results, _ := ltm.GetRecent(ctx, "test", 10)
	if len(results) != 3 {
		t.Errorf("GetRecent() should respect max entries, got %v, want 3", len(results))
	}
}

func TestLongTermMemory_GetAllTypes(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_memory_types")
	defer os.RemoveAll(tempDir)

	ltm, _ := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  10,
	})

	ltm.Add(ctx, "type1", "Memory 1", nil)
	ltm.Add(ctx, "type2", "Memory 2", nil)
	ltm.Add(ctx, "type1", "Memory 3", nil)

	types, err := ltm.GetAllTypes(ctx)
	if err != nil {
		t.Fatalf("GetAllTypes() error = %v", err)
	}

	if len(types) != 2 {
		t.Errorf("GetAllTypes() should return 2 types, got %v", len(types))
	}
}

func TestLongTermMemory_StateDict(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test_memory_state")
	defer os.RemoveAll(tempDir)

	ltm, _ := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  100,
		TTL:         time.Hour,
	})

	state := ltm.StateDict()

	if state["storage_path"] != tempDir {
		t.Errorf("storage_path = %v, want %v", state["storage_path"], tempDir)
	}
	if state["max_entries"] != 100 {
		t.Errorf("max_entries = %v, want 100", state["max_entries"])
	}
}

func TestLongTermMemory_Persistence(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_memory_persist")
	defer os.RemoveAll(tempDir)

	// Create first instance and add memory
	ltm1, _ := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  10,
	})
	id, _ := ltm1.Add(ctx, "persistent", "Persisted content", nil)

	// Create second instance - should load from disk
	ltm2, err := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  10,
	})
	if err != nil {
		t.Fatalf("NewLongTermMemory() error = %v", err)
	}

	// Verify memory was persisted
	entry, err := ltm2.Get(ctx, "persistent", id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if entry.Content != "Persisted content" {
		t.Errorf("Content = %v, want 'Persisted content'", entry.Content)
	}
}

func TestLongTermMemory_ClearAll(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Join(os.TempDir(), "test_memory_clearall")
	defer os.RemoveAll(tempDir)

	ltm, _ := NewLongTermMemory(&LongTermMemoryConfig{
		StoragePath: tempDir,
		MaxEntries:  10,
	})

	ltm.Add(ctx, "type1", "Memory 1", nil)
	ltm.Add(ctx, "type2", "Memory 2", nil)

	err := ltm.ClearAll(ctx)
	if err != nil {
		t.Fatalf("ClearAll() error = %v", err)
	}

	types, _ := ltm.GetAllTypes(ctx)
	if len(types) != 0 {
		t.Errorf("GetAllTypes() should return empty after ClearAll(), got %v", len(types))
	}
}
