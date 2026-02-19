package embedding

import (
	"context"
	"testing"
)

func TestMockModel(t *testing.T) {
	model := NewMockModel(1536)

	if model == nil {
		t.Fatal("NewMockModel returned nil")
	}

	if model.ModelName() != "mock-model" {
		t.Errorf("Expected model name 'mock-model', got '%s'", model.ModelName())
	}

	if model.Dimension() != 1536 {
		t.Errorf("Expected dimension 1536, got %d", model.Dimension())
	}
}

func TestMockModel_Embed(t *testing.T) {
	model := NewMockModel(128)
	ctx := context.Background()

	text := "test text"
	embedding, err := model.Embed(ctx, text)

	if err != nil {
		t.Fatalf("Embed() returned error: %v", err)
	}

	if len(embedding) != 128 {
		t.Errorf("Expected embedding length 128, got %d", len(embedding))
	}
}

func TestMockModel_EmbedBatch(t *testing.T) {
	model := NewMockModel(64)
	ctx := context.Background()

	texts := []string{"text1", "text2", "text3"}
	embeddings, err := model.EmbedBatch(ctx, texts)

	if err != nil {
		t.Fatalf("EmbedBatch() returned error: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) != 64 {
			t.Errorf("Embedding %d: expected length 64, got %d", i, len(emb))
		}
	}
}

func TestMockModel_EmbedBatchEmpty(t *testing.T) {
	model := NewMockModel(64)
	ctx := context.Background()

	embeddings, err := model.EmbedBatch(ctx, []string{})

	if err != nil {
		t.Fatalf("EmbedBatch() returned error: %v", err)
	}

	if len(embeddings) != 0 {
		t.Errorf("Expected 0 embeddings, got %d", len(embeddings))
	}
}

func TestMockModel_SetModelName(t *testing.T) {
	model := NewMockModel(64)
	model.SetModelName("custom-model")

	if model.ModelName() != "custom-model" {
		t.Errorf("Expected model name 'custom-model', got '%s'", model.ModelName())
	}
}

func TestMockModel_ZeroDimension(t *testing.T) {
	model := NewMockModel(0)

	if model.Dimension() != 1536 {
		t.Errorf("Expected default dimension 1536, got %d", model.Dimension())
	}

	model = NewMockModel(-10)

	if model.Dimension() != 1536 {
		t.Errorf("Expected default dimension 1536, got %d", model.Dimension())
	}
}

func TestNewOpenAIModel_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig("test-key")

	if cfg.APIKey != "test-key" {
		t.Errorf("Expected APIKey 'test-key', got '%s'", cfg.APIKey)
	}

	if cfg.Model != "text-embedding-3-small" {
		t.Errorf("Expected Model 'text-embedding-3-small', got '%s'", cfg.Model)
	}

	if cfg.BaseURL != "" {
		t.Errorf("Expected empty BaseURL, got '%s'", cfg.BaseURL)
	}
}

// TestOpenAIModel_Embed requires an actual API key, so it's skipped in normal tests
func TestOpenAIModel_Embed(t *testing.T) {
	t.Skip("Requires OpenAI API key")

	cfg := DefaultConfig("your-api-key-here")
	model, err := NewOpenAIModel(cfg)
	if err != nil {
		t.Fatalf("NewOpenAIModel() returned error: %v", err)
	}

	ctx := context.Background()
	embedding, err := model.Embed(ctx, "test")

	if err != nil {
		t.Fatalf("Embed() returned error: %v", err)
	}

	if len(embedding) == 0 {
		t.Error("Expected non-empty embedding")
	}
}
