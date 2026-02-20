// Package embedding provides a unified interface for embedding providers.
package embedding

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMockProvider(t *testing.T) {
	t.Run("with positive dimension", func(t *testing.T) {
		provider := NewMockProvider(768)
		assert.Equal(t, "mock-provider", provider.ModelName())
		assert.Equal(t, 768, provider.Dimension())
	})

	t.Run("with zero or negative dimension defaults to 1536", func(t *testing.T) {
		provider1 := NewMockProvider(0)
		assert.Equal(t, 1536, provider1.Dimension())

		provider2 := NewMockProvider(-100)
		assert.Equal(t, 1536, provider2.Dimension())
	})
}

func TestMockProvider_Embed(t *testing.T) {
	ctx := context.Background()
	provider := NewMockProvider(384)

	embedding, err := provider.Embed(ctx, "test text")
	require.NoError(t, err)
	assert.Len(t, embedding, 384)
	assert.Equal(t, float32(0.1), embedding[0])
}

func TestMockProvider_EmbedBatch(t *testing.T) {
	ctx := context.Background()
	provider := NewMockProvider(512)

	texts := []string{"text1", "text2", "text3"}
	embeddings, err := provider.EmbedBatch(ctx, texts)
	require.NoError(t, err)
	assert.Len(t, embeddings, 3)

	for _, emb := range embeddings {
		assert.Len(t, emb, 512)
		assert.Equal(t, float32(0.1), emb[0])
	}
}

func TestMockProvider_SetModelName(t *testing.T) {
	provider := NewMockProvider(256)
	assert.Equal(t, "mock-provider", provider.ModelName())

	provider.SetModelName("custom-model")
	assert.Equal(t, "custom-model", provider.ModelName())
}

func TestMockProvider_ImplementsProvider(t *testing.T) {
	// Compile-time interface check
	var _ Provider = (*MockProvider)(nil)
}
