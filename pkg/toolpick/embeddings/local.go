// Package local provides local embedding model integration using fastembed.
package local

import (
	"context"
	"fmt"

	"github.com/tingly-dev/tingly-scope/pkg/toolpick/selector"
)

// FastEmbedder provides local embedding using fastembed.
type FastEmbedder struct {
	modelName string
	// In production, this would wrap fastembed library
}

// NewFastEmbedder creates a new fastembed-based embedder.
func NewFastEmbedder(modelName string) *FastEmbedder {
	if modelName == "" {
		modelName = "BAAI/bge-small-en-v1.5"
	}

	return &FastEmbedder{
		modelName: modelName,
	}
}

// GenerateEmbedding generates embedding for the given text.
// Note: This is a placeholder. Real implementation would use fastembed Go bindings.
func (f *FastEmbedder) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// TODO: Integrate actual fastembed library
	// For now, use the default embedder as fallback
	return DefaultEmbedder{}.GenerateEmbedding(ctx, text)
}

// FastEmbedderConfig holds configuration for fastembed.
type FastEmbedderConfig struct {
	ModelName string
	CacheDir  string
	Device    string // "cpu" or "cuda"
}

// Recommended models for fastembed:
// - BAAI/bge-small-en-v1.5: Fast, good quality (384 dim)
// - BAAI/bge-large-en-v1.5: Best quality (1024 dim)
// - BAAI/bge-base-en-v1.5: Balanced (768 dim)

// DefaultEmbedder provides fallback embedding implementation.
type DefaultEmbedder struct{}

func (d *DefaultEmbedder) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Simple word frequency embedding as fallback
	return generateWordFrequencyEmbedding(text), nil
}

// generateWordFrequencyEmbedding creates a simple word frequency vector.
func generateWordFrequencyEmbedding(text string) []float64 {
	// Tokenize
	words := tokenize(text)
	freq := make(map[string]int)
	for _, word := range words {
		freq[word]++
	}

	// Create vector
	embedding := make([]float64, 128)
	for word, count := range freq {
		idx := hashString(word) % 128
		embedding[idx] += float64(count)
	}

	// Normalize
	var norm float64
	for _, v := range embedding {
		norm += v * v
	}
	if norm > 0 {
		norm = sqrt(norm)
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}

// Helper functions
func tokenize(text string) []string {
	// Simple word tokenization
	var words []string
	currentWord := ""

	for _, c := range text {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			currentWord += string(c)
		} else {
			if currentWord != "" {
				words = append(words, toLower(currentWord))
				currentWord = ""
			}
		}
	}
	if currentWord != "" {
		words = append(words, toLower(currentWord))
	}

	return words
}

func hashString(s string) int {
	hash := 0
	for _, c := range s {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			result[i] = byte(c + 32)
		} else {
			result[i] = byte(c)
		}
	}
	return string(result)
}

func sqrt(x float64) float64 {
	z := 1.0
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

// EmbeddingProvider implements selector.EmbeddingProvider
type EmbeddingProvider struct {
	embedder *FastEmbedder
}

func (e *EmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	return e.embedder.GenerateEmbedding(ctx, text)
}

// NewEmbeddingProvider creates a new embedding provider.
func NewEmbeddingProvider(modelName string) *EmbeddingProvider {
	return &EmbeddingProvider{
		embedder: NewFastEmbedder(modelName),
	}
}

// Usage example:
//
// import (
//     "github.com/tingly-dev/tingly-scope/pkg/toolpick"
//     "github.com/tingly-dev/tingly-scope/pkg/toolpick/embeddings/local"
// )
//
// embedder := local.NewEmbeddingProvider("BAAI/bge-small-en-v1.5")
// selector := selector.NewSemanticSelector(embedder, cache)
//
// smartToolkit := toolpick.NewToolProvider(baseToolkit, &toolpick.Config{
//     DefaultStrategy: "semantic",
//     MaxTools: 20,
// })

// Model recommendations:
//
// For production use with fastembed:
// 1. BAAI/bge-small-en-v1.5
//    - Dimensions: 384
//    - Speed: ~5ms per embedding
//    - Size: ~130MB
//    - Accuracy: High for English
//
// 2. BAAI/bge-large-en-v1.5
//    - Dimensions: 1024
//    - Speed: ~15ms per embedding
//    - Size: ~1.3GB
//    - Accuracy: Best for English
//
// 3. BAAI/bge-base-en-v1.5
//    - Dimensions: 768
//    - Speed: ~10ms per embedding
//    - Size: ~500MB
//    - Accuracy: Balanced
//
// Integration with fastembed Go:
//
// When fastembed Go bindings are available, implement like this:
//
// import "github.com/qdrant/fastembed"
//
// func (f *FastEmbedder) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
//     model, err := fastembed.NewEmbeddingModel(f.modelName)
//     if err != nil {
//         return nil, err
//     }
//     defer model.Close()
//
//     embedding, err := model.Embed([]string{text})
//     if err != nil {
//         return nil, err
//     }
//
//     return embedding, nil
// }

var _ = fmt.Sprintf // import placeholder
var _ = context.Background
