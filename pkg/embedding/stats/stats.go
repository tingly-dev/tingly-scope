// Package stats provides a word frequency based embedding implementation.
// It uses simple statistical methods to generate embeddings without external dependencies.
package stats

import (
	"context"
	"strings"

	"github.com/tingly-dev/tingly-scope/pkg/embedding"
)

const (
	defaultDimension = 128
	modelName        = "stats-wordfreq"
)

// Provider implements embedding.Provider using word frequency statistics.
type Provider struct {
	dimension int
}

// New creates a new stats-based embedding provider.
func New(dimension int) *Provider {
	if dimension <= 0 {
		dimension = defaultDimension
	}
	return &Provider{dimension: dimension}
}

// NewDefault creates a stats provider with default dimension (128).
func NewDefault() *Provider {
	return New(defaultDimension)
}

// Embed generates an embedding for a single text.
func (p *Provider) Embed(ctx context.Context, text string) ([]float32, error) {
	return generateWordFrequencyEmbedding(text, p.dimension), nil
}

// EmbedBatch generates embeddings for multiple texts.
func (p *Provider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		result[i] = generateWordFrequencyEmbedding(text, p.dimension)
	}
	return result, nil
}

// Dimension returns the embedding dimension.
func (p *Provider) Dimension() int {
	return p.dimension
}

// ModelName returns the model name.
func (p *Provider) ModelName() string {
	return modelName
}

// generateWordFrequencyEmbedding creates a word frequency vector.
func generateWordFrequencyEmbedding(text string, dimension int) []float32 {
	words := tokenize(text)
	freq := make(map[string]int)
	for _, word := range words {
		freq[word]++
	}

	vec := make([]float32, dimension)
	for word, count := range freq {
		idx := hashString(word) % dimension
		if idx < 0 {
			idx = -idx
		}
		vec[idx] += float32(count)
	}

	return embedding.Normalize(vec)
}

// tokenize splits text into words.
func tokenize(text string) []string {
	var words []string
	var currentWord strings.Builder

	for _, c := range text {
		if isAlphaNumeric(c) {
			currentWord.WriteRune(toLower(c))
		} else {
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}
	}
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// isAlphaNumeric checks if a character is alphanumeric.
func isAlphaNumeric(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// toLower converts a character to lowercase.
func toLower(c rune) rune {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// hashString computes a simple hash of a string.
func hashString(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	return h
}
