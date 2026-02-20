// Package openai provides OpenAI embedding integration for tool-pick.
//
// Deprecated: Use github.com/tingly-dev/tingly-scope/pkg/embedding/api instead.
// The unified embedding.Provider can be adapted to selector.EmbeddingProvider using:
//
//	import (
//	    "github.com/tingly-dev/tingly-scope/pkg/embedding"
//	    "github.com/tingly-dev/tingly-scope/pkg/embedding/api"
//	    toolembeddings "github.com/tingly-dev/tingly-scope/pkg/toolpick/embeddings"
//	)
//
//	p, _ := api.New(&api.Config{APIKey: apiKey})
//	adapter := toolembeddings.NewProviderAdapter(p) // selector.EmbeddingProvider
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// EmbeddingClient provides OpenAI embedding API integration.
type EmbeddingClient struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewEmbeddingClient creates a new OpenAI embedding client.
func NewEmbeddingClient(apiKey, model string) *EmbeddingClient {
	baseURL := "https://api.openai.com/v1"
	if model == "" {
		model = "text-embedding-3-small"
	}

	return &EmbeddingClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// EmbeddingRequest represents the API request.
type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// EmbeddingResponse represents the API response.
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
}

// GenerateEmbedding generates embedding for the given text.
func (c *EmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	req := EmbeddingRequest{
		Input: []string{text},
		Model: c.model,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/embeddings",
		bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var embedResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(embedResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embedResp.Data[0].Embedding, nil
}

// Config holds configuration for OpenAI embedding.
type Config struct {
	APIKey  string
	Model   string
	BaseURL string
}

// Recommended models:
// - text-embedding-3-small: Fast, cost-effective (recommended)
// - text-embedding-3-large: Higher accuracy
// - text-embedding-ada-002: Legacy model
