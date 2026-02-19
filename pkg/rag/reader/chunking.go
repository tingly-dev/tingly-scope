package reader

import (
	"strings"
	"unicode"
)

// FixedChunkingStrategy splits text into fixed-size chunks with overlap
type FixedChunkingStrategy struct {
	chunkSize int
	overlap   int
	separator string
}

// NewFixedChunkingStrategy creates a new fixed chunking strategy
func NewFixedChunkingStrategy(chunkSize, overlap int, separator string) *FixedChunkingStrategy {
	if overlap >= chunkSize {
		overlap = chunkSize / 4 // Default to 25% overlap
	}
	return &FixedChunkingStrategy{
		chunkSize: chunkSize,
		overlap:   overlap,
		separator: separator,
	}
}

// Split splits text into chunks
func (s *FixedChunkingStrategy) Split(text string) []string {
	if text == "" {
		return []string{}
	}

	// If separator is provided and text contains it, split by separator first
	if s.separator != "" && strings.Contains(text, s.separator) {
		return s.splitWithSeparator(text)
	}

	return s.splitBySize(text)
}

// splitWithSeparator splits text using separator as hints
func (s *FixedChunkingStrategy) splitWithSeparator(text string) []string {
	// Split by separator
	sections := strings.Split(text, s.separator)

	var chunks []string
	currentChunk := ""

	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		testChunk := currentChunk
		if testChunk != "" {
			testChunk += s.separator + " " + section
		} else {
			testChunk = section
		}

		if len(testChunk) <= s.chunkSize {
			currentChunk = testChunk
		} else {
			// Save current chunk if not empty
			if currentChunk != "" {
				chunks = append(chunks, currentChunk)
			}

			// If section alone is too large, split it
			if len(section) > s.chunkSize {
				subChunks := s.splitBySize(section)
				chunks = append(chunks, subChunks...)
				currentChunk = ""
			} else {
				currentChunk = section
			}
		}
	}

	// Add last chunk
	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return s.addOverlap(chunks)
}

// splitBySize splits text into chunks by character count
func (s *FixedChunkingStrategy) splitBySize(text string) []string {
	var chunks []string

	runes := []rune(text)
	start := 0

	for start < len(runes) {
		end := start + s.chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		// Try to find a good break point (whitespace or punctuation)
		if end < len(runes) {
			// Look backwards for a break point
			breakPoint := end
			for i := end - 1; i > start+s.chunkSize/2; i-- {
				if unicode.IsSpace(runes[i]) || unicode.IsPunct(runes[i]) {
					breakPoint = i + 1
					break
				}
			}
			end = breakPoint
		}

		chunk := string(runes[start:end])
		chunks = append(chunks, chunk)

		start = end - s.overlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

// addOverlap adds overlap between consecutive chunks
func (s *FixedChunkingStrategy) addOverlap(chunks []string) []string {
	if len(chunks) <= 1 || s.overlap <= 0 {
		return chunks
	}

	result := make([]string, 0, len(chunks))
	result = append(result, chunks[0])

	for i := 1; i < len(chunks); i++ {
		prev := chunks[i-1]
		curr := chunks[i]

		// Add overlap from previous chunk
		overlapText := ""
		if len(prev) > s.overlap {
			// Get last overlap characters from previous chunk
			runes := []rune(prev)
			overlapText = string(runes[len(runes)-s.overlap:])
		} else {
			overlapText = prev
		}

		combined := overlapText + "\n\n" + curr
		result = append(result, combined)
	}

	return result
}

// Name returns the name of the strategy
func (s *FixedChunkingStrategy) Name() string {
	return "fixed"
}

// NoChunkingStrategy returns the entire text as a single chunk
type NoChunkingStrategy struct{}

// NewNoChunkingStrategy creates a new no-chunking strategy
func NewNoChunkingStrategy() *NoChunkingStrategy {
	return &NoChunkingStrategy{}
}

// Split returns the text as a single chunk
func (s *NoChunkingStrategy) Split(text string) []string {
	if text == "" {
		return []string{}
	}
	return []string{text}
}

// Name returns the name of the strategy
func (s *NoChunkingStrategy) Name() string {
	return "none"
}
