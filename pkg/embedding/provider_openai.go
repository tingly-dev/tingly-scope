// Package embedding provides a unified interface for embedding providers.
// This file contains the OpenAI API provider implementation.
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL   = "https://api.openai.com/v1"
	defaultModel     = "text-embedding-3-small"
	defaultTimeout   = 30 * time.Second
	defaultDimension = 1536 // text-embedding-3-small
)

// Model dimensions
const (
	DimensionSmall = 1536 // text-embedding-3-small
	DimensionLarge = 3072 // text-embedding-3-large
)

// OpenAIConfig holds configuration for the OpenAI API provider.
type OpenAIConfig struct {
	BaseURL   string        // API base URL (default: OpenAI)
	APIKey    string        // API key
	Model     string        // Model name (default: text-embedding-3-small)
	Timeout   time.Duration // Request timeout
	Dimension int           // Embedding dimension (0 = use model default)
}

// OpenAIProvider implements Provider via OpenAI HTTP API.
type OpenAIProvider struct {
	baseURL   string
	apiKey    string
	model     string
	dimension int
	timeout   time.Duration
	client    *http.Client
}

// NewOpenAIProvider creates a new OpenAI API provider with the given configuration.
func NewOpenAIProvider(cfg *OpenAIConfig) (*OpenAIProvider, error) {
	if cfg == nil {
		cfg = &OpenAIConfig{}
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	model := cfg.Model
	if model == "" {
		model = defaultModel
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		dimension = getDimensionForModel(model)
	}

	return &OpenAIProvider{
		baseURL:   baseURL,
		apiKey:    cfg.APIKey,
		model:     model,
		dimension: dimension,
		timeout:   timeout,
		client:    &http.Client{Timeout: timeout},
	}, nil
}

// Embed generates an embedding for a single text.
func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := p.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts.
func (p *OpenAIProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrInvalidInput
	}

	req := openaiRequest{
		Input: texts,
		Model: p.model,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		p.baseURL+"/embeddings",
		bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, ErrRateLimited
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embedResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(embedResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Convert and sort by index
	result := make([][]float32, len(texts))
	for _, d := range embedResp.Data {
		if d.Index < 0 || d.Index >= len(result) {
			continue
		}
		result[d.Index] = Float64To32(d.Embedding)
	}

	// Update dimension from response
	if len(result) > 0 && len(result[0]) > 0 {
		p.dimension = len(result[0])
	}

	return result, nil
}

// Dimension returns the embedding dimension.
func (p *OpenAIProvider) Dimension() int {
	return p.dimension
}

// ModelName returns the model name.
func (p *OpenAIProvider) ModelName() string {
	return p.model
}

// openaiRequest represents the API request.
type openaiRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// openaiResponse represents the API response.
type openaiResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
}

// getDimensionForModel returns the default dimension for a model.
func getDimensionForModel(model string) int {
	switch model {
	case "text-embedding-3-large":
		return DimensionLarge
	case "text-embedding-3-small", "text-embedding-ada-002":
		return DimensionSmall
	default:
		return defaultDimension
	}
}
