package embedding

import (
	"context"

	"github.com/tingly-dev/tingly-scope/pkg/rag"
)

// MockModel implements the Model interface for testing
type MockModel struct {
	modelName string
	dimension int
}

// NewMockModel creates a new mock embedding model
func NewMockModel(dimension int) *MockModel {
	if dimension <= 0 {
		dimension = 1536 // default
	}
	return &MockModel{
		modelName: "mock-model",
		dimension: dimension,
	}
}

// Embed generates a mock embedding for a single text
func (m *MockModel) Embed(ctx context.Context, text string) (rag.Embedding, error) {
	return m.generateEmbedding(), nil
}

// EmbedBatch generates mock embeddings for multiple texts
func (m *MockModel) EmbedBatch(ctx context.Context, texts []string) ([]rag.Embedding, error) {
	embeddings := make([]rag.Embedding, len(texts))
	for i := range texts {
		embeddings[i] = m.generateEmbedding()
	}
	return embeddings, nil
}

// Dimension returns the dimension of the embeddings
func (m *MockModel) Dimension() int {
	return m.dimension
}

// ModelName returns the name of the model
func (m *MockModel) ModelName() string {
	return m.modelName
}

// generateEmbedding generates a deterministic but unique embedding based on length
func (m *MockModel) generateEmbedding() rag.Embedding {
	embedding := make(rag.Embedding, m.dimension)
	// Generate simple pattern for testing
	for i := 0; i < m.dimension; i++ {
		embedding[i] = 0.1
	}
	return embedding
}

// SetModelName sets the model name for testing
func (m *MockModel) SetModelName(name string) {
	m.modelName = name
}
