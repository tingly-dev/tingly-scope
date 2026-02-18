// Package embeddings provides embedding model integrations for tool-pick.
package embeddings

// This package provides various embedding implementations:
//
// - openai: OpenAI embedding API integration
// - local: Local model support (fastembed compatibility)
// - cohere: Cohere embedding API
// - huggingface: HuggingFace inference API
//
// Usage:
//
//   import "github.com/tingly-dev/tingly-scope/pkg/toolpick/embeddings/openai"
//
//   embedder := openai.NewEmbeddingClient(apiKey, "text-embedding-3-small")
//   selector := selector.NewSemanticSelector(embedder, cache)

// Model recommendations:
//
// For most use cases:
//   - Production: OpenAI text-embedding-3-small (fast, cost-effective)
//   - High accuracy: OpenAI text-embedding-3-large
//   - Local deployment: BAAI/bge-small-en-v1.5
//
// Performance comparison (semantic search accuracy):
//   - Word frequency (fallback): ~40%
//   - BAAI/bge-small-en-v1.5: ~75%
//   - OpenAI text-embedding-3-small: ~85%
//   - OpenAI text-embedding-3-large: ~90%
