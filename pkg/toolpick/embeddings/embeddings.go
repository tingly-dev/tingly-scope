// Package embeddings provides embedding adapters for tool-pick.
package embeddings

// This package provides adapters for using the unified embedding.Provider
// with toolpick's selector interface.
//
// Usage (Recommended):
//
//	import (
//	    "github.com/tingly-dev/tingly-scope/pkg/embedding"
//	    toolembeddings "github.com/tingly-dev/tingly-scope/pkg/toolpick/embeddings"
//	)
//
//	p := embedding.NewStatsProviderDefault()
//	adapter := toolembeddings.NewProviderAdapter(p)
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
