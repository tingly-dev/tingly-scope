package stats

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-scope/pkg/embedding"
)

func TestProvider_Embed(t *testing.T) {
	p := NewDefault()

	ctx := context.Background()

	// Test single embedding
	emb, err := p.Embed(ctx, "hello world")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(emb) != defaultDimension {
		t.Errorf("Expected dimension %d, got %d", defaultDimension, len(emb))
	}

	// Check normalization
	var norm float32
	for _, v := range emb {
		norm += v * v
	}
	if norm < 0.99 || norm > 1.01 {
		t.Errorf("Embedding not normalized, norm = %f", norm)
	}
}

func TestProvider_EmbedBatch(t *testing.T) {
	p := NewDefault()

	ctx := context.Background()
	texts := []string{"hello", "world", "test"}

	embeddings, err := p.EmbedBatch(ctx, texts)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) != defaultDimension {
			t.Errorf("Embedding %d: expected dimension %d, got %d", i, defaultDimension, len(emb))
		}
	}
}

func TestProvider_Dimension(t *testing.T) {
	tests := []struct {
		dim      int
		expected int
	}{
		{0, defaultDimension},
		{-1, defaultDimension},
		{64, 64},
		{256, 256},
	}

	for _, tt := range tests {
		p := New(tt.dim)
		if p.Dimension() != tt.expected {
			t.Errorf("New(%d): expected dimension %d, got %d", tt.dim, tt.expected, p.Dimension())
		}
	}
}

func TestProvider_ModelName(t *testing.T) {
	p := NewDefault()
	if p.ModelName() != modelName {
		t.Errorf("Expected model name %s, got %s", modelName, p.ModelName())
	}
}

func TestProvider_Similarity(t *testing.T) {
	p := NewDefault()
	ctx := context.Background()

	// Similar texts should have high similarity
	emb1, _ := p.Embed(ctx, "fix bug in authentication")
	emb2, _ := p.Embed(ctx, "fix bug in login")

	sim := embedding.CosineSimilarity(emb1, emb2)
	if sim < 0.5 {
		t.Errorf("Similar texts should have high similarity, got %f", sim)
	}

	// Different texts should have lower similarity
	emb3, _ := p.Embed(ctx, "write documentation for API")
	sim2 := embedding.CosineSimilarity(emb1, emb3)
	if sim2 > sim {
		t.Errorf("Different texts should have lower similarity than similar texts")
	}
}

func TestProvider_EmptyInput(t *testing.T) {
	p := NewDefault()
	ctx := context.Background()

	emb, err := p.Embed(ctx, "")
	if err != nil {
		t.Fatalf("Empty input should not error: %v", err)
	}

	// Empty input should produce zero vector
	var sum float32
	for _, v := range emb {
		sum += v
	}
	if sum != 0 {
		t.Errorf("Empty input should produce zero vector, sum = %f", sum)
	}
}

func BenchmarkProvider_Embed(b *testing.B) {
	p := NewDefault()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Embed(ctx, "hello world test benchmark")
	}
}

func BenchmarkProvider_EmbedBatch(b *testing.B) {
	p := NewDefault()
	ctx := context.Background()
	texts := []string{"hello", "world", "test", "benchmark", "embedding"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.EmbedBatch(ctx, texts)
	}
}
