// Package anthropic provides an SDK client wrapper for Anthropic API using the official SDK.
package anthropic

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
)

// SDKConfig holds the configuration for the Anthropic SDK client.
// This is a separate type from the old Config to allow gradual migration.
type SDKConfig struct {
	// APIKey is the Anthropic API key
	APIKey string

	// BaseURL is the base URL for the API (optional, defaults to official API)
	BaseURL string

	// Model is the model name to use
	Model string

	// MaxTokens is the maximum number of tokens to generate (defaults to 2048)
	MaxTokens int

	// Stream enables streaming responses
	Stream bool

	// DefaultTemperature is the default temperature for requests
	DefaultTemperature *float64

	// DefaultTopP is the default top_p for requests
	DefaultTopP *float64

	// DefaultStopSequences are the default stop sequences
	DefaultStopSequences []string

	// Thinking configures Claude's extended thinking mode
	// Use SDK types directly for thinking configuration
	Thinking interface{}
}

// SDKClient wraps the official Anthropic SDK client.
// This is a new type to allow gradual migration from the old Client.
type SDKClient struct {
	client anthropic.Client
	config *SDKConfig
}

// NewSDKClient creates a new Anthropic client using the official SDK.
func NewSDKClient(cfg *SDKConfig) (*SDKClient, error) {
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 2048
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	return &SDKClient{
		client: anthropic.NewClient(opts...),
		config: cfg,
	}, nil
}

// Client returns the underlying SDK client for direct access.
func (c *SDKClient) Client() *anthropic.Client {
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

// Messages returns the Messages service from the SDK for direct access.
func (c *SDKClient) Messages() anthropic.MessageService {
	return c.client.Messages
}

// CreateMessage sends a non-streaming message creation request.
// This is a direct passthrough to the SDK's Messages.New method.
// Use SDK types (anthropic.MessageNewParams) directly.
func (c *SDKClient) CreateMessage(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	return c.client.Messages.New(ctx, params)
}

// CreateMessageStreaming sends a streaming message creation request.
// This is a direct passthrough to the SDK's Messages.NewStreaming method.
// Returns the SDK stream type directly - use ssestream.Stream[anthropic.MessageStreamEventUnion].
func (c *SDKClient) CreateMessageStreaming(ctx context.Context, params anthropic.MessageNewParams) *ssestream.Stream[anthropic.MessageStreamEventUnion] {
	return c.client.Messages.NewStreaming(ctx, params)
}

// CountTokens counts tokens in a message without creating it.
// This is a direct passthrough to the SDK's Messages.CountTokens method.
func (c *SDKClient) CountTokens(ctx context.Context, params anthropic.MessageCountTokensParams) (*anthropic.MessageTokensCount, error) {
	return c.client.Messages.CountTokens(ctx, params)
}
