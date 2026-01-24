package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tingly-io/agentscope-go/pkg/agentscope/message"
)

// Memory is the interface for agent memory implementations
type Memory interface {
	// Add adds a message to memory
	Add(ctx context.Context, msg *message.Msg) error

	// GetMessages returns all messages in memory
	GetMessages() []*message.Msg

	// GetLastN returns the last n messages
	GetLastN(n int) []*message.Msg

	// Clear clears all messages from memory
	Clear()

	// Size returns the number of messages in memory
	Size() int
}

// Config holds the configuration for memory
type Config struct {
	MaxSize int `json:"max_size"`
}

// DefaultConfig returns the default memory configuration
func DefaultConfig() *Config {
	return &Config{
		MaxSize: 1000,
	}
}

// MemoryWithEmbedding represents memory that supports embedding-based retrieval
type MemoryWithEmbedding interface {
	Memory

	// AddEmbedding adds a message with its embedding to memory
	AddEmbedding(ctx context.Context, msg *message.Msg, embedding []float32) error

	// Search searches for similar messages based on embedding
	Search(ctx context.Context, queryEmbedding []float32, topK int) []*message.Msg
}

// History implements an in-memory message store
type History struct {
	mu       sync.RWMutex
	messages []*message.Msg
	maxSize  int
}

// NewHistory creates a new history memory
func NewHistory(maxSize int) *History {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &History{
		messages: make([]*message.Msg, 0, maxSize),
		maxSize:  maxSize,
	}
}

// Add adds a message to memory
func (h *History) Add(ctx context.Context, msg *message.Msg) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.messages = append(h.messages, msg)

	// Trim if over max size
	if h.maxSize > 0 && len(h.messages) > h.maxSize {
		h.messages = h.messages[len(h.messages)-h.maxSize:]
	}

	return nil
}

// GetMessages returns all messages in memory
func (h *History) GetMessages() []*message.Msg {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*message.Msg, len(h.messages))
	copy(result, h.messages)
	return result
}

// GetLastN returns the last n messages
func (h *History) GetLastN(n int) []*message.Msg {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if n <= 0 {
		return []*message.Msg{}
	}

	start := len(h.messages) - n
	if start < 0 {
		start = 0
	}

	result := make([]*message.Msg, len(h.messages)-start)
	copy(result, h.messages[start:])
	return result
}

// Clear clears all messages from memory
func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.messages = make([]*message.Msg, 0, h.maxSize)
}

// Size returns the number of messages in memory
func (h *History) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.messages)
}

// GetMessagesByRole returns messages filtered by role
func (h *History) GetMessagesByRole(role message.Role) []*message.Msg {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []*message.Msg
	for _, msg := range h.messages {
		if msg.Role == role {
			result = append(result, msg)
		}
	}
	return result
}

// GetMessagesAfter returns messages after the given timestamp
func (h *History) GetMessagesAfter(timestamp string) []*message.Msg {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []*message.Msg
	for _, msg := range h.messages {
		if msg.Timestamp > timestamp {
			result = append(result, msg)
		}
	}
	return result
}

// GetMessagesBetween returns messages between two timestamps
func (h *History) GetMessagesBetween(start, end string) []*message.Msg {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []*message.Msg
	for _, msg := range h.messages {
		if msg.Timestamp >= start && msg.Timestamp <= end {
			result = append(result, msg)
		}
	}
	return result
}

// MessageWithMeta extends a message with metadata for memory
type MessageWithMeta struct {
	Message   *message.Msg `json:"message"`
	Embedding []float32    `json:"embedding,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

// VectorMemory implements memory with embedding-based search
type VectorMemory struct {
	mu        sync.RWMutex
	messages  []*MessageWithMeta
	maxSize   int
	embedding EmbeddingModel
}

// EmbeddingModel is the interface for embedding models
type EmbeddingModel interface {
	// Encode generates an embedding for the given text
	Encode(ctx context.Context, text string) ([]float32, error)

	// BatchEncode generates embeddings for multiple texts
	BatchEncode(ctx context.Context, texts []string) ([][]float32, error)
}

// NewVectorMemory creates a new vector memory
func NewVectorMemory(maxSize int, embeddingModel EmbeddingModel) *VectorMemory {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &VectorMemory{
		messages: make([]*MessageWithMeta, 0, maxSize),
		maxSize:  maxSize,
		embedding: embeddingModel,
	}
}

// Add adds a message to memory
func (v *VectorMemory) Add(ctx context.Context, msg *message.Msg) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Generate embedding
	embedding, err := v.embedding.Encode(ctx, msg.GetTextContent())
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	meta := &MessageWithMeta{
		Message:   msg,
		Embedding: embedding,
		Timestamp: time.Now(),
	}

	v.messages = append(v.messages, meta)

	// Trim if over max size
	if v.maxSize > 0 && len(v.messages) > v.maxSize {
		v.messages = v.messages[len(v.messages)-v.maxSize:]
	}

	return nil
}

// AddEmbedding adds a message with a pre-computed embedding
func (v *VectorMemory) AddEmbedding(ctx context.Context, msg *message.Msg, embedding []float32) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	meta := &MessageWithMeta{
		Message:   msg,
		Embedding: embedding,
		Timestamp: time.Now(),
	}

	v.messages = append(v.messages, meta)

	// Trim if over max size
	if v.maxSize > 0 && len(v.messages) > v.maxSize {
		v.messages = v.messages[len(v.messages)-v.maxSize:]
	}

	return nil
}

// GetMessages returns all messages in memory
func (v *VectorMemory) GetMessages() []*message.Msg {
	v.mu.RLock()
	defer v.mu.RUnlock()

	result := make([]*message.Msg, len(v.messages))
	for i, meta := range v.messages {
		result[i] = meta.Message
	}
	return result
}

// GetLastN returns the last n messages
func (v *VectorMemory) GetLastN(n int) []*message.Msg {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if n <= 0 {
		return []*message.Msg{}
	}

	start := len(v.messages) - n
	if start < 0 {
		start = 0
	}

	result := make([]*message.Msg, len(v.messages)-start)
	for i := start; i < len(v.messages); i++ {
		result[i-start] = v.messages[i].Message
	}
	return result
}

// Clear clears all messages from memory
func (v *VectorMemory) Clear() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.messages = make([]*MessageWithMeta, 0, v.maxSize)
}

// Size returns the number of messages in memory
func (v *VectorMemory) Size() int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return len(v.messages)
}

// Search searches for similar messages based on embedding
func (v *VectorMemory) Search(ctx context.Context, queryEmbedding []float32, topK int) []*message.Msg {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if topK <= 0 || len(v.messages) == 0 {
		return []*message.Msg{}
	}

	// Calculate cosine similarity
	type scoreResult struct {
		message *message.Msg
		score   float32
	}

	results := make([]scoreResult, 0, len(v.messages))

	for _, meta := range v.messages {
		if len(meta.Embedding) != len(queryEmbedding) {
			continue
		}

		similarity := cosineSimilarity(queryEmbedding, meta.Embedding)
		results = append(results, scoreResult{
			message: meta.Message,
			score:   similarity,
		})
	}

	// Sort by similarity (descending)
	// Simple bubble sort (for small datasets)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Return top K
	if topK > len(results) {
		topK = len(results)
	}

	output := make([]*message.Msg, topK)
	for i := 0; i < topK; i++ {
		output[i] = results[i].message
	}

	return output
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt32(normA) * sqrt32(normB))
}

// sqrt32 calculates square root for float32
func sqrt32(x float32) float32 {
	return float32(sqrt(float64(x)))
}

// sqrt is a simple square root implementation
func sqrt(x float64) float64 {
	// Newton's method
	z := 1.0
	for i := 0; i < 20; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

// TemporaryBuffer is a temporary memory that clears after a certain number of operations
type TemporaryBuffer struct {
	mu       sync.RWMutex
	messages []*message.Msg
	maxOps   int
	opCount  int
}

// NewTemporaryBuffer creates a new temporary buffer
func NewTemporaryBuffer(maxOps int) *TemporaryBuffer {
	return &TemporaryBuffer{
		messages: make([]*message.Msg, 0),
		maxOps:   maxOps,
	}
}

// Add adds a message to the temporary buffer
func (t *TemporaryBuffer) Add(ctx context.Context, msg *message.Msg) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.messages = append(t.messages, msg)
	t.opCount++

	// Clear after max operations
	if t.opCount >= t.maxOps {
		t.messages = make([]*message.Msg, 0)
		t.opCount = 0
	}

	return nil
}

// GetMessages returns all messages in the buffer
func (t *TemporaryBuffer) GetMessages() []*message.Msg {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*message.Msg, len(t.messages))
	copy(result, t.messages)
	return result
}

// GetLastN returns the last n messages
func (t *TemporaryBuffer) GetLastN(n int) []*message.Msg {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if n <= 0 {
		return []*message.Msg{}
	}

	start := len(t.messages) - n
	if start < 0 {
		start = 0
	}

	result := make([]*message.Msg, len(t.messages)-start)
	copy(result, t.messages[start:])
	return result
}

// Clear clears the buffer
func (t *TemporaryBuffer) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.messages = make([]*message.Msg, 0)
	t.opCount = 0
}

// Size returns the number of messages in the buffer
func (t *TemporaryBuffer) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.messages)
}
