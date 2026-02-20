// Package embedding provides a unified interface for embedding providers.
package embedding

import (
	"context"
)

// MockProvider implements the Provider interface for testing.
// This is a lightweight mock for unit tests and integration tests.
// For production, use actual providers like OpenAIProvider or SidecarProvider.
type MockProvider struct {
	modelName string
	dimension int
}

// NewMockProvider creates a new mock embedding provider.
// If dimension <= 0, defaults to 1536 (OpenAI text-embedding-ada-002 dimension).
func NewMockProvider(dimension int) *MockProvider {
	if dimension <= 0 {
		dimension = 1536 // default
	}
	return &MockProvider{
		modelName: "mock-provider",
		dimension: dimension,
	}
}

// Embed generates a mock embedding for a single text.
func (m *MockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return m.generateEmbedding(), nil
}

// EmbedBatch generates mock embeddings for multiple texts.
func (m *MockProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embeddings[i] = m.generateEmbedding()
	}
	return embeddings, nil
}

// Dimension returns the dimension of the embeddings.
func (m *MockProvider) Dimension() int {
	return m.dimension
}

// ModelName returns the name of the model.
func (m *MockProvider) ModelName() string {
	return m.modelName
}

// SetModelName sets the model name for testing purposes.
func (m *MockProvider) SetModelName(name string) {
	m.modelName = name
}

// generateEmbedding generates a deterministic but unique embedding based on length.
// For testing purposes, returns a simple pattern.
func (m *MockProvider) generateEmbedding() []float32 {
	embedding := make([]float32, m.dimension)
	// Generate simple pattern for testing
	for i := 0; i < m.dimension; i++ {
		embedding[i] = 0.1
	}
	return embedding
}
