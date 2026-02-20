// Package embeddings provides adapters for using rag embedding models with toolpick.
//
// For new code, prefer using github.com/tingly-dev/tingly-scope/pkg/embedding.Provider
// and adapting it with NewProviderAdapter defined in provider_adapter.go.
//
// The adapters in this file remain for backward compatibility with existing
// rag.EmbeddingModel implementations.
package embeddings

import (
	"context"

	"github.com/tingly-dev/tingly-scope/pkg/rag"
)

// RAGEmbeddingAdapter wraps a rag.EmbeddingModel to implement selector.EmbeddingProvider.
type RAGEmbeddingAdapter struct {
	model rag.EmbeddingModel
}

// NewRAGEmbeddingAdapter creates a new adapter for a rag embedding model.
func NewRAGEmbeddingAdapter(model rag.EmbeddingModel) *RAGEmbeddingAdapter {
	return &RAGEmbeddingAdapter{model: model}
}

// GenerateEmbedding implements selector.EmbeddingProvider.
// It converts rag.Embedding ([]float32) to []float64.
func (a *RAGEmbeddingAdapter) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	embedding, err := a.model.Embed(ctx, text)
	if err != nil {
		return nil, err
	}

	// Convert []float32 to []float64
	result := make([]float64, len(embedding))
	for i, v := range embedding {
		result[i] = float64(v)
	}

	return result, nil
}

// GenerateEmbeddingsBatch generates embeddings for multiple texts.
func (a *RAGEmbeddingAdapter) GenerateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings, err := a.model.EmbedBatch(ctx, texts)
	if err != nil {
		return nil, err
	}

	// Convert each embedding
	result := make([][]float64, len(embeddings))
	for i, emb := range embeddings {
		result[i] = make([]float64, len(emb))
		for j, v := range emb {
			result[i][j] = float64(v)
		}
	}

	return result, nil
}

// EmbeddingModelAdapter wraps a rag embedding.Model to implement selector.EmbeddingProvider.
// This is for the more detailed embedding.Model interface from pkg/rag/embedding.
type EmbeddingModelAdapter struct {
	model interface {
		Embed(ctx context.Context, text string) (rag.Embedding, error)
	}
}

// NewEmbeddingModelAdapter creates a new adapter.
func NewEmbeddingModelAdapter(model interface {
	Embed(ctx context.Context, text string) (rag.Embedding, error)
}) *EmbeddingModelAdapter {
	return &EmbeddingModelAdapter{model: model}
}

// GenerateEmbedding implements selector.EmbeddingProvider.
func (a *EmbeddingModelAdapter) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	embedding, err := a.model.Embed(ctx, text)
	if err != nil {
		return nil, err
	}

	result := make([]float64, len(embedding))
	for i, v := range embedding {
		result[i] = float64(v)
	}

	return result, nil
}
