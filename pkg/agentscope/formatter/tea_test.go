package formatter

import (
	"testing"

	"github.com/tingly-io/agentscope-go/pkg/agentscope/message"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/types"
)

func TestTeaFormatter_FormatMessage(t *testing.T) {
	f := NewTeaFormatter()

	t.Run("text only message", func(t *testing.T) {
		msg := message.NewMsg("test", "Hello, world!", types.RoleUser)
		output := f.FormatMessage(msg)

		// Check that role and content are present
		if !contains(output, "USER") {
			t.Errorf("Expected 'USER' role in output, got: %s", output)
		}
		if !contains(output, "Hello, world!") {
			t.Errorf("Expected 'Hello, world!' in output, got: %s", output)
		}
	})

	t.Run("message with tool use", func(t *testing.T) {
		input := map[string]types.JSONSerializable{
			"path":    "/tmp/test.txt",
			"limit":   float64(10),
			"pattern": "*.go",
		}

		blocks := []message.ContentBlock{
			message.Text("I'll search for Go files"),
			&message.ToolUseBlock{
				ID:    "tool_123456",
				Name:  "grep_files",
				Input: input,
			},
		}

		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		output := f.FormatMessage(msg)

		// Check that tool use is formatted
		if !contains(output, "grep_files") {
			t.Errorf("Expected 'grep_files' in output, got: %s", output)
		}
		if !contains(output, "▶") {
			t.Errorf("Expected tool use indicator '▶' in output, got: %s", output)
		}
	})

	t.Run("message with tool result", func(t *testing.T) {
		blocks := []message.ContentBlock{
			&message.ToolResultBlock{
				ID:     "tool_123",
				Name:   "grep_files",
				Output: []message.ContentBlock{message.Text("Found 5 files")},
			},
		}

		msg := message.NewMsg("grep_files", blocks, types.RoleUser)
		output := f.FormatMessage(msg)

		// Check that tool result is formatted
		if !contains(output, "grep_files") {
			t.Errorf("Expected 'grep_files' in output, got: %s", output)
		}
		if !contains(output, "◀") {
			t.Errorf("Expected tool result indicator '◀' in output, got: %s", output)
		}
	})

	t.Run("compact mode", func(t *testing.T) {
		f.Compact = true

		input := map[string]types.JSONSerializable{"path": "/tmp/test.txt"}
		blocks := []message.ContentBlock{
			message.Text("Searching..."),
			&message.ToolUseBlock{ID: "t1", Name: "view_file", Input: input},
		}

		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		output := f.FormatMessage(msg)

		if !contains(output, "view_file") {
			t.Errorf("Expected 'view_file' in compact output, got: %s", output)
		}
	})

	t.Run("no color mode", func(t *testing.T) {
		f.NoColor = true

		blocks := []message.ContentBlock{
			&message.ToolUseBlock{ID: "t1", Name: "test_tool", Input: map[string]types.JSONSerializable{}},
		}

		msg := message.NewMsg("agent", blocks, types.RoleAssistant)
		output := f.FormatMessage(msg)

		// No color mode should not contain ANSI codes
		if contains(output, "\x1b[") {
			t.Errorf("Did not expect ANSI codes in no-color output, got: %s", output)
		}
	})
}

func TestTeaFormatter_FormatContentBlock(t *testing.T) {
	f := NewTeaFormatter()

	t.Run("text block", func(t *testing.T) {
		block := message.Text("Hello, world!")
		output := f.FormatContentBlock(block)

		if !contains(output, "Hello, world!") {
			t.Errorf("Expected text content in output, got: %s", output)
		}
	})

	t.Run("thinking block", func(t *testing.T) {
		block := &message.ThinkingBlock{Thinking: "Let me think..."}
		output := f.FormatContentBlock(block)

		if !contains(output, "💭") {
			t.Errorf("Expected thinking icon in output, got: %s", output)
		}
	})
}

func TestNewCompactTeaFormatter(t *testing.T) {
	f := NewCompactTeaFormatter()

	if !f.Compact {
		t.Error("Expected compact mode to be enabled")
	}
	if f.ShowToolIDs {
		t.Error("Expected tool IDs to be hidden in compact mode")
	}
	if f.ShowTimestamps {
		t.Error("Expected timestamps to be hidden in compact mode")
	}
}

func TestNewMonochromeTeaFormatter(t *testing.T) {
	f := NewMonochromeTeaFormatter()

	if !f.NoColor {
		t.Error("Expected no color mode to be enabled")
	}
}
