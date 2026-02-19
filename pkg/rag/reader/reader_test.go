package reader

import (
	"context"
	"testing"
)

func TestNewTextReader(t *testing.T) {
	reader := NewTextReader()

	if reader == nil {
		t.Fatal("NewTextReader returned nil")
	}

	if reader.chunkingStrategy == nil {
		t.Error("Expected default chunking strategy")
	}
}

func TestNewTextReaderWithStrategy(t *testing.T) {
	strategy := NewNoChunkingStrategy()
	reader := NewTextReaderWithStrategy(strategy)

	if reader.chunkingStrategy != strategy {
		t.Error("Expected provided chunking strategy")
	}
}

func TestTextReader_Read(t *testing.T) {
	reader := NewTextReader()
	ctx := context.Background()

	text := "This is a test document.\n\nIt has multiple paragraphs.\n\nEach paragraph should be chunked separately."

	docs, err := reader.Read(ctx, text)

	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	if len(docs) == 0 {
		t.Error("Expected at least one document")
	}

	// Verify document structure
	for i, doc := range docs {
		if doc.GetTextContent() == "" {
			t.Errorf("Document %d: expected non-empty content", i)
		}

		if doc.Metadata.DocID == "" {
			t.Errorf("Document %d: expected non-empty DocID", i)
		}

		if doc.Metadata.ChunkID < 0 || doc.Metadata.ChunkID >= len(docs) {
			t.Errorf("Document %d: invalid ChunkID", i)
		}

		if doc.Metadata.TotalChunks != len(docs) {
			t.Errorf("Document %d: expected TotalChunks %d, got %d", i, len(docs), doc.Metadata.TotalChunks)
		}
	}
}

func TestTextReader_ReadEmpty(t *testing.T) {
	reader := NewTextReader()
	ctx := context.Background()

	_, err := reader.Read(ctx, "")

	if err == nil {
		t.Error("Expected error for empty source")
	}
}

func TestTextReader_SetChunkingStrategy(t *testing.T) {
	reader := NewTextReader()
	strategy := NewNoChunkingStrategy()

	reader.SetChunkingStrategy(strategy)

	if reader.chunkingStrategy != strategy {
		t.Error("Expected chunking strategy to be updated")
	}
}

func TestTextReader_GetDocID(t *testing.T) {
	reader := NewTextReader()

	shortText := "short text"
	docID := reader.GetDocID(shortText)

	if docID == "" {
		t.Error("Expected non-empty DocID")
	}

	longText := "This is a very long text that exceeds fifty characters in length and should be truncated when generating the document identifier"
	docID2 := reader.GetDocID(longText)

	if docID2 == "" {
		t.Error("Expected non-empty DocID for long text")
	}
}

func TestFixedChunkingStrategy_Split(t *testing.T) {
	strategy := NewFixedChunkingStrategy(20, 5, "\n\n")

	text := "This is paragraph one.\n\nThis is paragraph two.\n\nThis is paragraph three."

	chunks := strategy.Split(text)

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Verify all chunks have content
	for i, chunk := range chunks {
		if chunk == "" {
			t.Errorf("Chunk %d is empty", i)
		}
	}
}

func TestFixedChunkingStrategy_SplitEmpty(t *testing.T) {
	strategy := NewFixedChunkingStrategy(100, 10, "\n\n")

	chunks := strategy.Split("")

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestFixedChunkingStrategy_SplitLargeChunk(t *testing.T) {
	strategy := NewFixedChunkingStrategy(50, 10, "")

	// Create text that's larger than chunk size
	text := ""
	for i := 0; i < 200; i++ {
		text += "word "
	}

	chunks := strategy.Split(text)

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Verify chunks are approximately the right size
	for i, chunk := range chunks {
		// Account for overlap
		if len(chunk) > 70 {
			t.Errorf("Chunk %d: size %d exceeds expected max (with overlap)", i, len(chunk))
		}
	}
}

func TestNoChunkingStrategy_Split(t *testing.T) {
	strategy := NewNoChunkingStrategy()

	text := "This is a test document.\n\nIt has multiple paragraphs."

	chunks := strategy.Split(text)

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}

	if chunks[0] != text {
		t.Errorf("Expected chunk to equal input text")
	}
}

func TestNoChunkingStrategy_SplitEmpty(t *testing.T) {
	strategy := NewNoChunkingStrategy()

	chunks := strategy.Split("")

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestFixedChunkingStrategy_Name(t *testing.T) {
	strategy := NewFixedChunkingStrategy(100, 10, "\n\n")

	if strategy.Name() != "fixed" {
		t.Errorf("Expected name 'fixed', got '%s'", strategy.Name())
	}
}

func TestNoChunkingStrategy_Name(t *testing.T) {
	strategy := NewNoChunkingStrategy()

	if strategy.Name() != "none" {
		t.Errorf("Expected name 'none', got '%s'", strategy.Name())
	}
}

func TestFixedChunkingStrategy_LargeOverlap(t *testing.T) {
	// Overlap >= chunk size should be handled
	strategy := NewFixedChunkingStrategy(100, 100, "\n\n")

	if strategy.overlap >= strategy.chunkSize {
		t.Error("Expected overlap to be less than chunk size")
	}
}
