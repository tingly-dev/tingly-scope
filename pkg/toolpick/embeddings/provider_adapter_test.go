package embeddings

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-scope/pkg/embedding"
	"github.com/tingly-dev/tingly-scope/pkg/toolpick/selector"
)

func TestProviderAdapter(t *testing.T) {
	provider := embedding.NewStatsProviderDefault()
	adapter := NewProviderAdapter(provider)

	// Verify interface compliance
	var _ selector.EmbeddingProvider = adapter

	ctx := context.Background()

	t.Run("GenerateEmbedding", func(t *testing.T) {
		emb, err := adapter.GenerateEmbedding(ctx, "test text")
		if err != nil {
			t.Fatalf("GenerateEmbedding failed: %v", err)
		}
		if len(emb) != provider.Dimension() {
			t.Errorf("Expected dimension %d, got %d", provider.Dimension(), len(emb))
		}
	})

	t.Run("GenerateEmbeddingsBatch", func(t *testing.T) {
		texts := []string{"hello", "world", "test"}
		embeddings, err := adapter.GenerateEmbeddingsBatch(ctx, texts)
		if err != nil {
			t.Fatalf("GenerateEmbeddingsBatch failed: %v", err)
		}
		if len(embeddings) != len(texts) {
			t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
		}
		for i, emb := range embeddings {
			if len(emb) != provider.Dimension() {
				t.Errorf("Embedding %d: expected dimension %d, got %d", i, provider.Dimension(), len(emb))
			}
		}
	})

	t.Run("Dimension", func(t *testing.T) {
		if adapter.Dimension() != provider.Dimension() {
			t.Errorf("Expected dimension %d, got %d", provider.Dimension(), adapter.Dimension())
		}
	})

	t.Run("ModelName", func(t *testing.T) {
		if adapter.ModelName() != provider.ModelName() {
			t.Errorf("Expected model name %s, got %s", provider.ModelName(), adapter.ModelName())
		}
	})
}

func TestProviderAdapter_TypeConversion(t *testing.T) {
	provider := embedding.NewStatsProviderDefault()
	adapter := NewProviderAdapter(provider)
	ctx := context.Background()

	// Verify that float32 -> float64 conversion is correct
	float32Emb, _ := provider.Embed(ctx, "test")
	float64Emb, _ := adapter.GenerateEmbedding(ctx, "test")

	if len(float32Emb) != len(float64Emb) {
		t.Fatalf("Length mismatch: %d vs %d", len(float32Emb), len(float64Emb))
	}

	for i := range float32Emb {
		expected := float64(float32Emb[i])
		if float64Emb[i] != expected {
			t.Errorf("Index %d: expected %f, got %f", i, expected, float64Emb[i])
		}
	}
}

// MockProvider for testing type conversion edge cases
type mockProvider struct {
	dimension int
}

func (m *mockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	emb := make([]float32, m.dimension)
	for i := range emb {
		emb[i] = float32(i) * 0.1
	}
	return emb, nil
}

func (m *mockProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i], _ = m.Embed(ctx, texts[i])
	}
	return result, nil
}

func (m *mockProvider) Dimension() int {
	return m.dimension
}

func (m *mockProvider) ModelName() string {
	return "mock"
}

func TestProviderAdapter_WithMockProvider(t *testing.T) {
	mock := &mockProvider{dimension: 256}
	adapter := NewProviderAdapter(mock)

	// Verify it still implements the interface
	var _ selector.EmbeddingProvider = adapter
	var _ embedding.Provider = mock

	ctx := context.Background()

	emb, err := adapter.GenerateEmbedding(ctx, "test")
	if err != nil {
		t.Fatalf("GenerateEmbedding failed: %v", err)
	}

	if len(emb) != 256 {
		t.Errorf("Expected dimension 256, got %d", len(emb))
	}
}
