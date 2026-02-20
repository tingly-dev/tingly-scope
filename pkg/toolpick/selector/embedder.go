package selector

import (
	"context"
)

// defaultEmbedder provides a simple embedding implementation using word frequency.
//
// Deprecated: Use github.com/tingly-dev/tingly-scope/pkg/embedding/stats instead.
// The stats provider can be adapted using embeddings.NewProviderAdapter.
type defaultEmbedder struct{}

// GenerateEmbedding implements EmbeddingProvider.
func (d *defaultEmbedder) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	words := tokenizeWords(text)
	freq := make(map[string]int)
	for _, word := range words {
		freq[word]++
	}

	embedding := make([]float64, 128)
	for word, count := range freq {
		idx := simpleHash(word) % 128
		embedding[idx] += float64(count)
	}

	var norm float64
	for _, v := range embedding {
		norm += v * v
	}
	if norm > 0 {
		norm = sqrtFloat(norm)
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding, nil
}

func tokenizeWords(text string) []string {
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

func simpleHash(s string) int {
	hash := 0
	for _, c := range s {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

func sqrtFloat(x float64) float64 {
	z := 1.0
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
