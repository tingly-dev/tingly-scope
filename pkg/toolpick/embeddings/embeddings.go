// Package embeddings provides embedding model integrations for tool-pick.
package embeddings

// This package provides various embedding implementations:
//
// RECOMMENDED (New Code):
// - provider_adapter.go: Use unified embedding.Provider via NewProviderAdapter
//
// DEPRECATED (Legacy):
// - openai: OpenAI embedding API integration (use pkg/embedding/api instead)
// - local: Local model support (use pkg/embedding/stats instead)
// - rag_adapter: Legacy adapter for rag.EmbeddingModel
//
// Usage (Recommended):
//
//	import (
//	    "github.com/tingly-dev/tingly-scope/pkg/embedding/stats"
//	    "github.com/tingly-dev/tingly-scope/pkg/toolpick/embeddings"
//	)
//
//	p := stats.NewDefault()
//	adapter := embeddings.NewProviderAdapter(p)
//	selector := selector.NewSemanticSelector(adapter, cache)

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
