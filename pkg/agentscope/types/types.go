package types

import (
	"crypto/md5"
	"fmt"
	"math/rand"

	"github.com/tingly-io/agentscope-go/pkg/agentscope/utils"
)

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
	ToolChoiceAuto     ToolChoiceMode = "auto"
	ToolChoiceNone     ToolChoiceMode = "none"
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
	return utils.Timestamp()
}

// TimestampWithRandom returns a timestamp with a random suffix
func TimestampWithRandom() string {
	return utils.TimestampWithRandom()
}

// GenerateID creates a unique identifier
func GenerateID() string {
	return utils.GenerateID()
}

// GenerateIDFromText creates a deterministic ID from text
func GenerateIDFromText(text string) string {
	return utils.GenerateIDFromText(text)
}

// GenerateUUID creates a UUID-like identifier
func GenerateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Uint32(),
		uint16(rand.Uint32()&0xffff)|0x4000, // Version 4
		uint16(rand.Uint32()&0x3fff)|0x8000, // Variant
		uint16(rand.Uint32()&0xffff),
		rand.Uint32(), rand.Uint32()>>16,
	)
}

// GenerateUUIDFromText creates a deterministic UUID from text
func GenerateUUIDFromText(text string) string {
	h := md5.New()
	h.Write([]byte(text))
	data := h.Sum(nil)

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint(data[0])<<24|uint(data[1])<<16|uint(data[2])<<8|uint(data[3]),
		uint(data[4])<<8|uint(data[5]),
		(uint(data[6])&0x0fff)|0x3000, // Version 3
		(uint(data[6])&0x3fff)|0x8000, // Variant - fix: use separate bytes
		uint(data[7])<<8|uint(data[8]),
		uint(data[9])<<24|uint(data[10])<<16|uint(data[11])<<8|uint(data[12]),
		uint(data[13])<<24|uint(data[14])<<16|uint(data[15])<<8,
	)
}
