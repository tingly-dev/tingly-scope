// Package openai provides an SDK client wrapper for OpenAI API using the official SDK.
package openai

import (
	"context"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
)

// SDKConfig holds the configuration for the OpenAI SDK client.
// This is a separate type from the old Config to allow gradual migration.
type SDKConfig struct {
	// APIKey is the OpenAI API key
	APIKey string

	// BaseURL is the base URL for the API (optional, defaults to official API)
	BaseURL string

	// Model is the model name to use
	Model string

	// Stream enables streaming responses
	Stream bool

	// DefaultMaxTokens is the default max tokens for requests
	DefaultMaxTokens *int

	// DefaultTemperature is the default temperature for requests
	DefaultTemperature *float64

	// DefaultTopP is the default top_p for requests
	DefaultTopP *float64

	// DefaultStop are the default stop sequences
	DefaultStop []string
}

// SDKClient wraps the official OpenAI SDK client.
// This is a new type to allow gradual migration from the old Client.
type SDKClient struct {
	client openai.Client
	config *SDKConfig
}

// NewSDKClient creates a new OpenAI client using the official SDK.
func NewSDKClient(cfg *SDKConfig) (*SDKClient, error) {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	return &SDKClient{
		client: openai.NewClient(opts...),
		config: cfg,
	}, nil
}

// Client returns the underlying SDK client for direct access.
func (c *SDKClient) Client() *openai.Client {
	return &c.client
}

// Config returns the client configuration.
func (c *SDKClient) Config() *SDKConfig {
	return c.config
}

// ModelName returns the model name.
func (c *SDKClient) ModelName() string {
	return c.config.Model
}

// IsStreaming returns whether streaming is enabled.
func (c *SDKClient) IsStreaming() bool {
	return c.config.Stream
}

// ChatCompletions returns the ChatCompletion service from the SDK for direct access.
func (c *SDKClient) ChatCompletions() openai.ChatCompletionService {
	return c.client.Chat.Completions
}

// CreateChatCompletion sends a non-streaming chat completion request.
// This is a direct passthrough to the SDK's Chat.Completions.New method.
// Use SDK types (openai.ChatCompletionNewParams) directly.
func (c *SDKClient) CreateChatCompletion(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	return c.client.Chat.Completions.New(ctx, params)
}

// CreateChatCompletionStreaming sends a streaming chat completion request.
// This is a direct passthrough to the SDK's Chat.Completions.NewStreaming method.
// Returns the SDK stream type directly - use ssestream.Stream[openai.ChatCompletionChunk].
func (c *SDKClient) CreateChatCompletionStreaming(ctx context.Context, params openai.ChatCompletionNewParams) *ssestream.Stream[openai.ChatCompletionChunk] {
	return c.client.Chat.Completions.NewStreaming(ctx, params)
}
