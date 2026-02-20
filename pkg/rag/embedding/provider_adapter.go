// Package embedding provides embedding model implementations for RAG.
// This file provides adapters to use embedding.Provider with RAG interfaces.
package embedding

import (
	"context"

	"github.com/tingly-dev/tingly-scope/pkg/embedding"
	"github.com/tingly-dev/tingly-scope/pkg/rag"
)

// ProviderAdapter wraps an embedding.Provider to implement rag.EmbeddingModel.
// Use this to integrate the unified embedding.Provider with RAG's KnowledgeBase.
//
// Example:
//
//	p, _ := api.New(&api.Config{APIKey: apiKey})
//	adapter := embedding.NewProviderAdapter(p)
//	kb := rag.NewSimpleKnowledgeBase(adapter, store)
type ProviderAdapter struct {
	provider embedding.Provider
}

// NewProviderAdapter creates a new adapter that wraps an embedding.Provider.
func NewProviderAdapter(p embedding.Provider) *ProviderAdapter {
	return &ProviderAdapter{provider: p}
}

// Embed implements rag.EmbeddingModel.
func (a *ProviderAdapter) Embed(ctx context.Context, text string) (rag.Embedding, error) {
	return a.provider.Embed(ctx, text)
}

// EmbedBatch implements rag.EmbeddingModel.
func (a *ProviderAdapter) EmbedBatch(ctx context.Context, texts []string) ([]rag.Embedding, error) {
	embeddings, err := a.provider.EmbedBatch(ctx, texts)
	if err != nil {
		return nil, err
	}
	// Convert [][]float32 to []rag.Embedding
	result := make([]rag.Embedding, len(embeddings))
	for i, emb := range embeddings {
		result[i] = rag.Embedding(emb)
	}
	return result, nil
}

// ModelAdapter wraps an embedding.Provider to implement the full Model interface
// (which includes Dimension and ModelName methods).
type ModelAdapter struct {
	ProviderAdapter
}

// NewModelAdapter creates a new adapter that implements the full Model interface.
func NewModelAdapter(p embedding.Provider) *ModelAdapter {
	return &ModelAdapter{
		ProviderAdapter: ProviderAdapter{provider: p},
	}
}

// Dimension implements Model.Dimension.
func (a *ModelAdapter) Dimension() int {
	return a.provider.Dimension()
}

// ModelName implements Model.ModelName.
func (a *ModelAdapter) ModelName() string {
	return a.provider.ModelName()
}

// Ensure interfaces are satisfied.
var (
	_ rag.EmbeddingModel = (*ProviderAdapter)(nil)
	_ Model              = (*ModelAdapter)(nil)
)
