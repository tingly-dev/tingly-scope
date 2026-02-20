// Package embeddings provides embedding adapters for tool-pick.
// This file provides adapters to use embedding.Provider with toolpick's selector.
package embeddings

import (
	"context"

	"github.com/tingly-dev/tingly-scope/pkg/embedding"
	"github.com/tingly-dev/tingly-scope/pkg/toolpick/selector"
)

// ProviderAdapter wraps an embedding.Provider to implement selector.EmbeddingProvider.
// Use this to integrate the unified embedding.Provider with toolpick's selectors.
//
// Example:
//
//	p := stats.NewDefault()
//	adapter := embeddings.NewProviderAdapter(p)
//	selector := selector.NewSemanticSelector(adapter, cache)
type ProviderAdapter struct {
	provider embedding.Provider
}

// NewProviderAdapter creates a new adapter that wraps an embedding.Provider.
func NewProviderAdapter(p embedding.Provider) *ProviderAdapter {
	return &ProviderAdapter{provider: p}
}

// GenerateEmbedding implements selector.EmbeddingProvider.
// Converts []float32 from Provider to []float64 expected by selector.
func (a *ProviderAdapter) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	emb, err := a.provider.Embed(ctx, text)
	if err != nil {
		return nil, err
	}
	return embedding.Float32To64(emb), nil
}

// GenerateEmbeddingsBatch generates embeddings for multiple texts.
func (a *ProviderAdapter) GenerateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings, err := a.provider.EmbedBatch(ctx, texts)
	if err != nil {
		return nil, err
	}

	result := make([][]float64, len(embeddings))
	for i, emb := range embeddings {
		result[i] = embedding.Float32To64(emb)
	}
	return result, nil
}

// Dimension returns the embedding dimension.
func (a *ProviderAdapter) Dimension() int {
	return a.provider.Dimension()
}

// ModelName returns the model name.
func (a *ProviderAdapter) ModelName() string {
	return a.provider.ModelName()
}

// Ensure interface is satisfied.
var _ selector.EmbeddingProvider = (*ProviderAdapter)(nil)
