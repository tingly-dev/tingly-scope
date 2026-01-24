package types

import "time"

// Role represents the message role type
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// ContentBlockType represents the type of content block
type ContentBlockType string

const (
	BlockTypeText       ContentBlockType = "text"
	BlockTypeThinking   ContentBlockType = "thinking"
	BlockTypeToolUse    ContentBlockType = "tool_use"
	BlockTypeToolResult ContentBlockType = "tool_result"
	BlockTypeImage      ContentBlockType = "image"
	BlockTypeAudio      ContentBlockType = "audio"
	BlockTypeVideo      ContentBlockType = "video"
)

// MediaType represents the media type for content sources
type MediaType string

const (
	MediaTypeImageJPEG MediaType = "image/jpeg"
	MediaTypeImagePNG  MediaType = "image/png"
	MediaTypeImageGIF  MediaType = "image/gif"
	MediaTypeAudioMPEG MediaType = "audio/mpeg"
	MediaTypeAudioWAV  MediaType = "audio/wav"
	MediaTypeVideoMP4  MediaType = "video/mp4"
)

// ToolChoiceMode represents the tool choice mode
type ToolChoiceMode string

const (
	ToolChoiceAuto    ToolChoiceMode = "auto"
	ToolChoiceNone    ToolChoiceMode = "none"
	ToolChoiceRequired ToolChoiceMode = "required"
)

// JSONSerializable represents a value that can be serialized to JSON
type JSONSerializable interface{}

// ToolFunction represents a function that can be used as a tool
type ToolFunction interface{}

// StreamType represents the streaming response type
type StreamType int

const (
	StreamTypeNone StreamType = iota
	StreamTypeChat
	StreamTypeTool
)

// HookType represents the type of hook
type HookType string

const (
	HookTypePreReply   HookType = "pre_reply"
	HookTypePostReply  HookType = "post_reply"
	HookTypePrePrint   HookType = "pre_print"
	HookTypePostPrint  HookType = "post_print"
	HookTypePreObserve HookType = "pre_observe"
	HookTypePostObserve HookType = "post_observe"
)

// Timestamp returns a formatted timestamp string
func Timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05.000")
}

// GenerateID creates a unique identifier
func GenerateID() string {
	return time.Now().Format("20060102150405") + "-" + randString(8)
}

func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
