// Package embedding provides a unified interface for embedding providers.
package embedding

import "context"

// Provider is the unified interface for embedding providers.
type Provider interface {
	// Embed generates an embedding for a single text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts efficiently.
	// Default implementation calls Embed for each text if not overridden.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension returns the dimension of embeddings.
	Dimension() int

	// ModelName returns the model identifier.
	ModelName() string
}
