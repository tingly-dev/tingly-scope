package model

// GetToolUseBlocks returns all tool use blocks from the response
func (r *ChatResponse) GetToolUseBlocks() []*ToolUseBlockFromResponse {
	var blocks []*ToolUseBlockFromResponse

	for _, block := range r.Content {
		if tb, ok := block.(*ToolUseBlockFromResponse); ok {
			blocks = append(blocks, tb)
		}
	}

	return blocks
}

// ToolUseBlockFromResponse represents a tool use block in a response
type ToolUseBlockFromResponse struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Input map[string]any        `json:"input"`
}

// Type returns the block type
func (t *ToolUseBlockFromResponse) Type() ContentBlockType {
	return BlockTypeToolUse
}
