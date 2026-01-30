package tool

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// OutputConstraint defines the output limit policy for tools
type OutputConstraint interface {
	// MaxBytes returns the maximum output bytes (0 = unlimited)
	MaxBytes() int

	// MaxLines returns the maximum output lines (0 = unlimited)
	MaxLines() int

	// MaxItems returns the maximum list items (0 = unlimited)
	MaxItems() int

	// TruncateHint returns whether to add truncation notice
	TruncateHint() bool

	// Apply applies the constraint to the output string
	Apply(output string) string
}

// DefaultConstraint provides a standard constraint implementation
type DefaultConstraint struct {
	maxBytes     int
	maxLines     int
	maxItems     int
	truncateHint bool
}

// NewDefaultConstraint creates a new constraint with specified limits
// Use 0 for any dimension to indicate no limit
func NewDefaultConstraint(maxBytes, maxLines, maxItems int) *DefaultConstraint {
	return &DefaultConstraint{
		maxBytes:     maxBytes,
		maxLines:     maxLines,
		maxItems:     maxItems,
		truncateHint: true,
	}
}

// MaxBytes returns the maximum output bytes
func (dc *DefaultConstraint) MaxBytes() int {
	return dc.maxBytes
}

// MaxLines returns the maximum output lines
func (dc *DefaultConstraint) MaxLines() int {
	return dc.maxLines
}

// MaxItems returns the maximum list items
func (dc *DefaultConstraint) MaxItems() int {
	return dc.maxItems
}

// TruncateHint returns whether to add truncation notice
func (dc *DefaultConstraint) TruncateHint() bool {
	return dc.truncateHint
}

// Apply applies the constraint to the output string
func (dc *DefaultConstraint) Apply(output string) string {
	if dc == nil || (dc.maxBytes == 0 && dc.maxLines == 0) {
		return output
	}

	originalBytes := len(output)
	originalRunes := utf8.RuneCountInString(output)
	lines := strings.Split(output, "\n")

	truncated := false
	result := output

	// Apply byte limit
	if dc.maxBytes > 0 && originalBytes > dc.maxBytes {
		result = truncateBytes(result, dc.maxBytes)
		truncated = true
	}

	// Apply line limit
	if dc.maxLines > 0 && len(lines) > dc.maxLines {
		result = strings.Join(lines[:dc.maxLines], "\n")
		truncated = true
	}

	// Add truncation hint if needed
	if truncated && dc.truncateHint {
		hint := fmt.Sprintf("\n\n[Output truncated: %d bytes, %d lines total. Showing partial results.]",
			originalBytes, originalRunes)
		result += hint
	}

	return result
}

// NoConstraint returns a constraint that applies no limits
func NoConstraint() OutputConstraint {
	return &DefaultConstraint{
		maxBytes:     0,
		maxLines:     0,
		maxItems:     0,
		truncateHint: false,
	}
}

// truncateBytes truncates a string to a maximum number of bytes
// Handles UTF-8 correctly by cutting at rune boundaries
func truncateBytes(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return s
	}
	if len(s) <= maxBytes {
		return s
	}

	// Find the last valid UTF-8 boundary before maxBytes
	byteCount := 0
	for _, r := range s {
		runeLen := utf8.RuneLen(r)
		if byteCount+runeLen > maxBytes {
			return s[:byteCount]
		}
		byteCount += runeLen
	}
	return s
}

// Global constraint registry for configuration-based constraints
var (
	globalConstraints       = make(map[string]OutputConstraint)
	globalDefaultConstraint OutputConstraint
)

// SetGlobalConstraint sets a constraint for a specific tool name
func SetGlobalConstraint(toolName string, c OutputConstraint) {
	globalConstraints[toolName] = c
}

// GetGlobalConstraint retrieves the global constraint for a tool
func GetGlobalConstraint(toolName string) (OutputConstraint, bool) {
	c, ok := globalConstraints[toolName]
	return c, ok
}

// ClearGlobalConstraints removes all global constraints
func ClearGlobalConstraints() {
	globalConstraints = make(map[string]OutputConstraint)
}

// SetGlobalDefaultConstraint sets the default constraint for all tools
// that don't have a specific constraint
func SetGlobalDefaultConstraint(c OutputConstraint) {
	globalDefaultConstraint = c
}

// GetGlobalDefaultConstraint returns the global default constraint
func GetGlobalDefaultConstraint() OutputConstraint {
	return globalDefaultConstraint
}
