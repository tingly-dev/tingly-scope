// Package embedding provides embedding model implementations for RAG.
//
// For new code, prefer using github.com/tingly-dev/tingly-scope/pkg/embedding.Provider
// and adapting it with NewProviderAdapter or NewModelAdapter defined in provider_adapter.go.
package embedding

import (
	"context"

	"github.com/tingly-dev/tingly-scope/pkg/rag"
)

// Model is the interface that embedding models must implement
type Model interface {
	// Embed generates embeddings for a single text
	Embed(ctx context.Context, text string) (rag.Embedding, error)

	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(ctx context.Context, texts []string) ([]rag.Embedding, error)

	// Dimension returns the dimension of the embeddings
	Dimension() int

	// ModelName returns the name of the model
	ModelName() string
}

// EmbeddingRequest represents a request to generate embeddings
type EmbeddingRequest struct {
	Texts []string `json:"texts"`
}

// EmbeddingResponse represents a response from an embedding model
type EmbeddingResponse struct {
	Embeddings []rag.Embedding `json:"embeddings"`
	Model      string          `json:"model"`
	Usage      *Usage          `json:"usage,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	TotalTokens  int `json:"total_tokens"`
	PromptTokens int `json:"prompt_tokens"`
}
