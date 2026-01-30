package tool

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNewDefaultConstraint(t *testing.T) {
	c := NewDefaultConstraint(100, 10, 5, 30)

	if c.MaxBytes() != 100 {
		t.Errorf("Expected MaxBytes() = 100, got %d", c.MaxBytes())
	}
	if c.MaxLines() != 10 {
		t.Errorf("Expected MaxLines() = 10, got %d", c.MaxLines())
	}
	if c.MaxItems() != 5 {
		t.Errorf("Expected MaxItems() = 5, got %d", c.MaxItems())
	}
	if c.Timeout() != 30 {
		t.Errorf("Expected Timeout() = 30, got %d", c.Timeout())
	}
	if !c.TruncateHint() {
		t.Error("Expected TruncateHint() = true")
	}
}

func TestNoConstraint(t *testing.T) {
	c := NoConstraint()

	if c.MaxBytes() != 0 {
		t.Errorf("Expected MaxBytes() = 0, got %d", c.MaxBytes())
	}
	if c.MaxLines() != 0 {
		t.Errorf("Expected MaxLines() = 0, got %d", c.MaxLines())
	}
	if c.MaxItems() != 0 {
		t.Errorf("Expected MaxItems() = 0, got %d", c.MaxItems())
	}
	if c.Timeout() != 0 {
		t.Errorf("Expected Timeout() = 0, got %d", c.Timeout())
	}
	if c.TruncateHint() {
		t.Error("Expected TruncateHint() = false")
	}
}

func TestApplyConstraint_NoLimit(t *testing.T) {
	c := NewDefaultConstraint(0, 0, 0, 0)
	input := strings.Repeat("a", 10000)
	output := c.Apply(input)

	if output != input {
		t.Error("Expected output to be unchanged when no limits are set")
	}
}

func TestApplyConstraint_ByteLimit(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		maxBytes      int
		wantTruncated bool
		wantContains  string
	}{
		{
			name:          "small output unchanged",
			input:         strings.Repeat("a", 100),
			maxBytes:      1000,
			wantTruncated: false,
		},
		{
			name:          "large output truncated",
			input:         strings.Repeat("a", 10000),
			maxBytes:      100,
			wantTruncated: true,
			wantContains:  "truncated",
		},
		{
			name:          "output at limit unchanged",
			input:         strings.Repeat("a", 100),
			maxBytes:      100,
			wantTruncated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewDefaultConstraint(tt.maxBytes, 0, 0, 0)
			output := c.Apply(tt.input)

			if tt.wantTruncated {
				if !strings.Contains(output, "truncated") {
					t.Error("Expected output to contain truncation hint")
				}
				if len(output) > tt.maxBytes+200 { // +200 for the hint text
					t.Errorf("Output too long: got %d bytes", len(output))
				}
			} else {
				if output != tt.input {
					t.Error("Expected output to be unchanged")
				}
			}
		})
	}
}

func TestApplyConstraint_LineLimit(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		maxLines      int
		wantTruncated bool
	}{
		{
			name:          "small output unchanged",
			input:         "line1\nline2\nline3",
			maxLines:      10,
			wantTruncated: false,
		},
		{
			name:          "large output truncated",
			input:         strings.Repeat("line\n", 1000),
			maxLines:      10,
			wantTruncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewDefaultConstraint(0, tt.maxLines, 0, 0)
			output := c.Apply(tt.input)

			if tt.wantTruncated {
				if !strings.Contains(output, "truncated") {
					t.Error("Expected output to contain truncation hint")
				}
				lines := strings.Split(output, "\n")
				// Should have maxLines + 2 for the hint lines
				if len(lines) > tt.maxLines+3 {
					t.Errorf("Too many lines: got %d", len(lines))
				}
			} else {
				if output != tt.input {
					t.Error("Expected output to be unchanged")
				}
			}
		})
	}
}

func TestApplyConstraint_Combined(t *testing.T) {
	input := strings.Repeat("a\n", 1000) // 1000 lines, ~2000 bytes

	c := NewDefaultConstraint(500, 100, 0, 0)
	output := c.Apply(input)

	if !strings.Contains(output, "truncated") {
		t.Error("Expected output to contain truncation hint")
	}
}

func TestTruncateBytes_UTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxBytes int
		wantLen  int
	}{
		{
			name:     "ASCII string",
			input:    "hello world",
			maxBytes: 5,
			wantLen:  5,
		},
		{
			name:     "UTF-8 string",
			input:    "你好世界", // 4 Chinese characters, 12 bytes
			maxBytes: 6,
			wantLen:  6, // 2 complete characters
		},
		{
			name:     "mixed ASCII and UTF-8",
			input:    "hello你好", // "hello" (5) + "你" (3) + "好" (3) = 11 bytes
			maxBytes: 8,
			wantLen:  8, // "hello" (5) + "你" (3) = 8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := truncateBytes(tt.input, tt.maxBytes)
			if len(output) != tt.wantLen {
				t.Errorf("Expected length %d, got %d", tt.wantLen, len(output))
			}
			// Verify valid UTF-8
			if !utf8.ValidString(output) {
				t.Error("Output is not valid UTF-8")
			}
		})
	}
}

func TestGlobalConstraintRegistry(t *testing.T) {
	// Clean up after test
	defer ClearGlobalConstraints()

	SetGlobalConstraint("test_tool", NewDefaultConstraint(100, 10, 5, 30))

	c, ok := GetGlobalConstraint("test_tool")
	if !ok {
		t.Error("Expected to find constraint for test_tool")
	}
	if c.MaxBytes() != 100 {
		t.Errorf("Expected MaxBytes() = 100, got %d", c.MaxBytes())
	}

	// Test non-existent tool
	_, ok = GetGlobalConstraint("nonexistent")
	if ok {
		t.Error("Expected not to find constraint for nonexistent tool")
	}
}

func TestGlobalDefaultConstraint(t *testing.T) {
	c := GetGlobalDefaultConstraint()
	if c != nil {
		t.Error("Expected nil default constraint initially")
	}

	SetGlobalDefaultConstraint(NewDefaultConstraint(50, 5, 2, 0))
	c = GetGlobalDefaultConstraint()
	if c == nil {
		t.Error("Expected default constraint to be set")
	}
	if c.MaxBytes() != 50 {
		t.Errorf("Expected MaxBytes() = 50, got %d", c.MaxBytes())
	}
}
