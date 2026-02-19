// Package local provides local embedding model integration using fastembed.
package embeddings

import (
	"context"

	"github.com/tingly-dev/tingly-scope/pkg/toolpick/selector"
)

// FastEmbedder provides local embedding using fastembed.
type FastEmbedder struct {
	modelName string
	fallback  selector.EmbeddingProvider
}

// NewFastEmbedder creates a new fastembed-based embedder.
func NewFastEmbedder(modelName string) *FastEmbedder {
	if modelName == "" {
		modelName = "BAAI/bge-small-en-v1.5"
	}

	return &FastEmbedder{
		modelName: modelName,
		fallback:  &defaultEmbedder{},
	}
}

// GenerateEmbedding generates embedding for the given text.
// Note: This is a placeholder. Real implementation would use fastembed Go bindings.
func (f *FastEmbedder) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// TODO: Integrate actual fastembed library
	// For now, use the default embedder as fallback
	return f.fallback.GenerateEmbedding(ctx, text)
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

// defaultEmbedder provides fallback embedding implementation (local copy).
type defaultEmbedder struct{}

func (d *defaultEmbedder) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	return generateWordFrequencyEmbedding(text), nil
}

// generateWordFrequencyEmbedding creates a simple word frequency vector.
func generateWordFrequencyEmbedding(text string) []float64 {
	words := tokenize(text)
	freq := make(map[string]int)
	for _, word := range words {
		freq[word]++
	}

	embedding := make([]float64, 128)
	for word, count := range freq {
		idx := hashString(word) % 128
		embedding[idx] += float64(count)
	}

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

func tokenize(text string) []string {
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

// Usage example:
//
// embedder := local.NewFastEmbedder("BAAI/bge-small-en-v1.5")
// selector := selector.NewSemanticSelector(embedder, cache)
