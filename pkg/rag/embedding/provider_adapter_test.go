package embedding

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-scope/pkg/embedding"
	"github.com/tingly-dev/tingly-scope/pkg/rag"
)

func TestProviderAdapter(t *testing.T) {
	provider := embedding.NewStatsProviderDefault()
	adapter := NewProviderAdapter(provider)

	// Verify interface compliance
	var _ rag.EmbeddingModel = adapter

	ctx := context.Background()

	t.Run("Embed", func(t *testing.T) {
		emb, err := adapter.Embed(ctx, "test text")
		if err != nil {
			t.Fatalf("Embed failed: %v", err)
		}
		if len(emb) != provider.Dimension() {
			t.Errorf("Expected dimension %d, got %d", provider.Dimension(), len(emb))
		}
	})

	t.Run("EmbedBatch", func(t *testing.T) {
		texts := []string{"hello", "world", "test"}
		embeddings, err := adapter.EmbedBatch(ctx, texts)
		if err != nil {
			t.Fatalf("EmbedBatch failed: %v", err)
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
}

func TestModelAdapter(t *testing.T) {
	provider := embedding.NewStatsProviderDefault()
	adapter := NewModelAdapter(provider)

	// Verify interface compliance
	var _ Model = adapter

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
