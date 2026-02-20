// Package embedding provides embedding model implementations for RAG.
//
// Deprecated: Use github.com/tingly-dev/tingly-scope/pkg/embedding instead.
// The unified embedding.Provider can be adapted to rag.EmbeddingModel using:
//
//	import (
//	    "github.com/tingly-dev/tingly-scope/pkg/embedding"
//	    "github.com/tingly-dev/tingly-scope/pkg/embedding/api"
//	    ragembedding "github.com/tingly-dev/tingly-scope/pkg/rag/embedding"
//	)
//
//	p, _ := api.New(&api.Config{APIKey: apiKey})
//	adapter := ragembedding.NewProviderAdapter(p) // rag.EmbeddingModel
package embedding

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/tingly-dev/tingly-scope/pkg/rag"
)

// Config holds the configuration for OpenAI embedding model
type Config struct {
	APIKey  string
	Model   string
	BaseURL string
}

// DefaultConfig returns a default configuration
func DefaultConfig(apiKey string) *Config {
	return &Config{
		APIKey: apiKey,
		Model:  "text-embedding-3-small",
	}
}

// OpenAIModel implements the Model interface using OpenAI's API
type OpenAIModel struct {
	client    openai.Client
	modelName string
	dimension int
}

// NewOpenAIModel creates a new OpenAI embedding model
func NewOpenAIModel(cfg *Config) (*OpenAIModel, error) {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	client := openai.NewClient(opts...)

	return &OpenAIModel{
		client:    client,
		modelName: cfg.Model,
		dimension: getDimensionForModel(cfg.Model),
	}, nil
}

// Embed generates an embedding for a single text
func (m *OpenAIModel) Embed(ctx context.Context, text string) (rag.Embedding, error) {
	embeddings, err := m.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for text")
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (m *OpenAIModel) EmbedBatch(ctx context.Context, texts []string) ([]rag.Embedding, error) {
	if len(texts) == 0 {
		return []rag.Embedding{}, nil
	}

	params := openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
		Model: openai.EmbeddingModel(m.modelName),
	}

	response, err := m.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	embeddings := make([]rag.Embedding, len(response.Data))
	for i, item := range response.Data {
		// Convert []float64 to []float32
		embedding := make(rag.Embedding, len(item.Embedding))
		for j, val := range item.Embedding {
			embedding[j] = float32(val)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// Dimension returns the dimension of the embeddings
func (m *OpenAIModel) Dimension() int {
	return m.dimension
}

// ModelName returns the name of the model
func (m *OpenAIModel) ModelName() string {
	return m.modelName
}

// getDimensionForModel returns the embedding dimension for a given model name
func getDimensionForModel(model string) int {
	switch model {
	case "text-embedding-3-large":
		return 3072
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-ada-002":
		return 1536
	default:
		return 1536 // default dimension
	}
}
