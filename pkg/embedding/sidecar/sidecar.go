// Package sidecar provides embedding via external gRPC sidecar services.
// It connects to local inference servers (e.g., Rust candle, Python FastEmbed) via gRPC.
package sidecar

import (
	"context"
	"fmt"
	"time"

	"github.com/tingly-dev/tingly-scope/pkg/embedding"
	"github.com/tingly-dev/tingly-scope/pkg/embedding/sidecar/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultAddress   = "localhost:50051"
	defaultModel     = "TaylorAI/bge-micro-v2"
	defaultTimeout   = 30 * time.Second
	defaultDimension = 384
)

// Config holds configuration for the sidecar provider.
type Config struct {
	Address   string        // gRPC address (default: localhost:50051)
	ModelName string        // Model identifier
	ModelPath string        // Path for model initialization
	Timeout   time.Duration // Request timeout
	Dimension int           // Embedding dimension (0 = auto-detect)
}

// Provider implements embedding.Provider via gRPC sidecar.
type Provider struct {
	client    pb.LLMServiceClient
	conn      *grpc.ClientConn
	modelName string
	dimension int
	timeout   time.Duration
}

// New creates a new sidecar provider with the given configuration.
func New(ctx context.Context, cfg *Config) (*Provider, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	address := cfg.Address
	if address == "" {
		address = defaultAddress
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	// Connect to sidecar
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sidecar at %s: %w", address, err)
	}

	client := pb.NewLLMServiceClient(conn)

	modelName := cfg.ModelName
	if modelName == "" {
		modelName = defaultModel
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		dimension = defaultDimension
	}

	// Check health
	healthCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	health, err := client.Health(healthCtx, &pb.HealthRequest{})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("sidecar health check failed: %w", err)
	}
	if !health.Healthy {
		conn.Close()
		return nil, embedding.ErrUnavailable
	}

	// Initialize model if path provided
	if cfg.ModelPath != "" {
		initCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		_, err := client.InitModel(initCtx, &pb.InitRequest{
			ModelPath:   cfg.ModelPath,
			ContextSize: 2048,
		})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to initialize model: %w", err)
		}
	}

	return &Provider{
		client:    client,
		conn:      conn,
		modelName: modelName,
		dimension: dimension,
		timeout:   timeout,
	}, nil
}

// Embed generates an embedding for a single text.
func (p *Provider) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, embedding.ErrInvalidInput
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	resp, err := p.client.Embed(ctx, &pb.EmbedRequest{Text: text})
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}

	// Update dimension if server returns different value
	if resp.Dim > 0 && int(resp.Dim) != p.dimension {
		p.dimension = int(resp.Dim)
	}

	return resp.Vector, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (p *Provider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := p.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		result[i] = emb
	}
	return result, nil
}

// Dimension returns the embedding dimension.
func (p *Provider) Dimension() int {
	return p.dimension
}

// ModelName returns the model name.
func (p *Provider) ModelName() string {
	return p.modelName
}

// Close closes the gRPC connection.
func (p *Provider) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
