package agent

import (
	"context"
	"fmt"

	"github.com/tingly-io/agentscope-go/pkg/agentscope/message"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/model"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/tool"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/types"
)

// ReActAgentConfig holds the configuration for a ReActAgent
type ReActAgentConfig struct {
	Name            string
	SystemPrompt    string
	Model           model.ChatModel
	Toolkit         tool.ToolProvider
	Memory          Memory
	MaxIterations   int
	Temperature     *float64
	MaxTokens       *int
}

// ReActAgent implements the ReAct (Reasoning + Acting) pattern
type ReActAgent struct {
	*AgentBase
	config       *ReActAgentConfig
	messages     []*message.Msg
	lastResponse *model.ChatResponse
}

// NewReActAgent creates a new ReAct agent
func NewReActAgent(config *ReActAgentConfig) *ReActAgent {
	base := NewAgentBase(config.Name, config.SystemPrompt)

	return &ReActAgent{
		AgentBase: base,
		config:    config,
		messages:  make([]*message.Msg, 0),
	}
}

// Reply generates a response to the given message
func (r *ReActAgent) Reply(ctx context.Context, input *message.Msg) (*message.Msg, error) {
	// Add input to memory
	if r.config.Memory != nil {
		if err := r.config.Memory.Add(ctx, input); err != nil {
			return nil, fmt.Errorf("failed to add message to memory: %w", err)
		}
	}

	// Build message history
	messages := r.buildMessageHistory(input)

	// Run pre-reply hooks
	kwargs := map[string]any{"message": input}
	if err := r.runPreHooks(ctx, types.HookTypePreReply, input, kwargs); err != nil {
		return nil, err
	}

	var response *message.Msg
	var err error

	// Check if we have tools and need to do ReAct loop
	if r.config.Toolkit != nil && len(r.config.Toolkit.GetSchemas()) > 0 {
		response, err = r.reactLoop(ctx, messages)
		if err != nil {
			return nil, err
		}
	} else {
		// Simple chat without tools
		resp, err := r.callModel(ctx, messages)
		if err != nil {
			return nil, err
		}
		response = r.createResponseMessage(resp)
	}

	// Add response to memory
	if r.config.Memory != nil {
		if err := r.config.Memory.Add(ctx, response); err != nil {
			return nil, fmt.Errorf("failed to add response to memory: %w", err)
		}
	}

	// Print the response
	if err := r.Print(ctx, response); err != nil {
		return nil, err
	}

	// Run post-reply hooks
	if err := r.runPostHooks(ctx, types.HookTypePostReply, input, kwargs); err != nil {
		return nil, err
	}

	// Broadcast to subscribers
	if err := r.BroadcastToSubscribers(ctx, response); err != nil {
		return nil, err
	}

	return response, nil
}

// reactLoop implements the ReAct loop of thought, action, observation
func (r *ReActAgent) reactLoop(ctx context.Context, initialMessages []*message.Msg) (*message.Msg, error) {
	messages := make([]*message.Msg, len(initialMessages))
	copy(messages, initialMessages)

	var thoughtContent []message.ContentBlock

	for i := 0; i < r.config.MaxIterations; i++ {
		// Get tools schema
		tools := r.config.Toolkit.GetSchemas()

		// Call model with tools
		resp, err := r.callModelWithTools(ctx, messages, tools)
		if err != nil {
			return nil, fmt.Errorf("iteration %d: %w", i, err)
		}

		r.lastResponse = resp

		// Check for tool use blocks
		toolBlocks := resp.GetToolUseBlocks()
		if len(toolBlocks) == 0 {
			// No more tools to use, return the final response
			thoughtContent = append(thoughtContent, resp.Content...)
			return r.createResponseMessage(resp), nil
		}

		// Accumulate content
		thoughtContent = append(thoughtContent, resp.Content...)

		// Execute each tool
		for _, toolBlock := range toolBlocks {
			// toolBlock is already *message.ToolUseBlock, no conversion needed
			// Add tool use to messages
			toolMsg := message.NewMsg(
				r.Name(),
				[]message.ContentBlock{toolBlock},
				types.RoleAssistant,
			)
			messages = append(messages, toolMsg)

			// Execute tool
			toolResp, err := r.config.Toolkit.Call(ctx, toolBlock)
			if err != nil {
				// Tool execution failed, add error as observation
				errorMsg := message.NewMsg(
					toolBlock.Name,
					[]message.ContentBlock{message.Text(fmt.Sprintf("Error: %v", err))},
					types.RoleUser,
				)
				messages = append(messages, errorMsg)
				continue
			}

			// Convert tool response content blocks
			resultBlocks := make([]message.ContentBlock, 0)
			for _, block := range toolResp.Content {
				resultBlocks = append(resultBlocks, block)
			}

			// Add tool result to messages
			resultMsg := message.NewMsg(
				toolBlock.Name,
				resultBlocks,
				types.RoleUser,
			)
			messages = append(messages, resultMsg)
		}
	}

	// Max iterations reached, return accumulated content
	return message.NewMsg(
		r.Name(),
		thoughtContent,
		types.RoleAssistant,
	), nil
}

// callModel calls the model with the given messages
func (r *ReActAgent) callModel(ctx context.Context, messages []*message.Msg) (*model.ChatResponse, error) {
	options := &model.CallOptions{
		Temperature: r.config.Temperature,
		MaxTokens:   r.config.MaxTokens,
	}

	if r.config.Model.IsStreaming() {
		ch, err := r.config.Model.Stream(ctx, messages, options)
		if err != nil {
			return nil, err
		}

		// Collect all chunks
		var content []message.ContentBlock
		for chunk := range ch {
			if chunk.Response != nil {
				content = append(content, chunk.Response.Content...)
			}
			if chunk.IsLast {
				break
			}
		}

		return &model.ChatResponse{
			ID:        types.GenerateID(),
			CreatedAt: types.Timestamp(),
			Type:      "chat",
			Content:   content,
		}, nil
	}

	return r.config.Model.Call(ctx, messages, options)
}

// callModelWithTools calls the model with tools enabled
func (r *ReActAgent) callModelWithTools(ctx context.Context, messages []*message.Msg, tools []model.ToolDefinition) (*model.ChatResponse, error) {
	options := &model.CallOptions{
		ToolChoice: types.ToolChoiceAuto,
		Tools:      tools,
		Temperature: r.config.Temperature,
		MaxTokens:   r.config.MaxTokens,
	}

	if r.config.Model.IsStreaming() {
		ch, err := r.config.Model.Stream(ctx, messages, options)
		if err != nil {
			return nil, err
		}

		// Collect all chunks
		var content []message.ContentBlock
		for chunk := range ch {
			if chunk.Response != nil {
				content = append(content, chunk.Response.Content...)
			}
			if chunk.IsLast {
				break
			}
		}

		return &model.ChatResponse{
			ID:        types.GenerateID(),
			CreatedAt: types.Timestamp(),
			Type:      "chat",
			Content:   content,
		}, nil
	}

	return r.config.Model.Call(ctx, messages, options)
}

// buildMessageHistory builds the message history for the model call
func (r *ReActAgent) buildMessageHistory(input *message.Msg) []*message.Msg {
	messages := make([]*message.Msg, 0, len(r.messages)+2)

	// Add system prompt
	if r.config.SystemPrompt != "" {
		sysMsg := message.NewMsg(
			"system",
			r.buildSystemPrompt(),
			types.RoleSystem,
		)
		messages = append(messages, sysMsg)
	}

	// Add memory messages
	if r.config.Memory != nil {
		memMessages := r.config.Memory.GetMessages()
		messages = append(messages, memMessages...)
	}

	// Add current input
	messages = append(messages, input)

	return messages
}

// buildSystemPrompt builds the full system prompt including tool descriptions
func (r *ReActAgent) buildSystemPrompt() string {
	prompt := r.config.SystemPrompt

	if r.config.Toolkit != nil && len(r.config.Toolkit.GetSchemas()) > 0 {
		prompt += "\n\n# Tools\n\nYou have access to the following tools:\n\n"

		for _, schema := range r.config.Toolkit.GetSchemas() {
			fn := schema.Function
			prompt += fmt.Sprintf("## %s\n", fn.Name)
			if fn.Description != "" {
				prompt += fn.Description + "\n"
			}

			// Add parameters info
			if fn.Parameters != nil {
				if params, ok := fn.Parameters["properties"].(map[string]any); ok {
					for name, param := range params {
						if pm, ok := param.(map[string]any); ok {
							prompt += fmt.Sprintf("- %s", name)
							if desc, ok := pm["description"].(string); ok && desc != "" {
								prompt += fmt.Sprintf(": %s", desc)
							}
							if t, ok := pm["type"].(string); ok {
								prompt += fmt.Sprintf(" (%s)", t)
							}
							prompt += "\n"
						}
					}
				}
			}
			prompt += "\n"
		}

		prompt += "To use a tool, respond with a tool_use block containing the tool name and parameters.\n"
	}

	return prompt
}

// createResponseMessage creates a response message from a chat response
func (r *ReActAgent) createResponseMessage(resp *model.ChatResponse) *message.Msg {
	if resp == nil {
		return message.NewMsg(
			r.Name(),
			[]message.ContentBlock{message.Text("No response generated")},
			types.RoleAssistant,
		)
	}

	if len(resp.Content) == 0 {
		return message.NewMsg(
			r.Name(),
			[]message.ContentBlock{message.Text("Empty response from model")},
			types.RoleAssistant,
		)
	}

	return message.NewMsg(
		r.Name(),
		resp.Content,
		types.RoleAssistant,
	)
}

// GetLastResponse returns the last response from the model
func (r *ReActAgent) GetLastResponse() *model.ChatResponse {
	return r.lastResponse
}

// ClearMemory clears the agent's message history
func (r *ReActAgent) ClearMemory() {
	r.messages = make([]*message.Msg, 0)
	if r.config.Memory != nil {
		r.config.Memory.Clear()
	}
}

// Memory is the interface for agent memory
type Memory interface {
	Add(ctx context.Context, msg *message.Msg) error
	GetMessages() []*message.Msg
	Clear()
}

// SimpleMemory implements an in-memory message store
type SimpleMemory struct {
	messages []*message.Msg
	maxSize  int
}

// NewSimpleMemory creates a new simple memory
func NewSimpleMemory(maxSize int) *SimpleMemory {
	return &SimpleMemory{
		messages: make([]*message.Msg, 0, maxSize),
		maxSize:  maxSize,
	}
}

// Add adds a message to memory
func (m *SimpleMemory) Add(ctx context.Context, msg *message.Msg) error {
	m.messages = append(m.messages, msg)

	// Trim if over max size
	if m.maxSize > 0 && len(m.messages) > m.maxSize {
		m.messages = m.messages[len(m.messages)-m.maxSize:]
	}

	return nil
}

// GetMessages returns all messages in memory
func (m *SimpleMemory) GetMessages() []*message.Msg {
	return m.messages
}

// Clear clears all messages from memory
func (m *SimpleMemory) Clear() {
	m.messages = make([]*message.Msg, 0)
}

// AddMessage adds a message to the agent's memory
func (r *ReActAgent) AddMessage(ctx context.Context, msg *message.Msg) error {
	if r.config.Memory != nil {
		return r.config.Memory.Add(ctx, msg)
	}
	return nil
}

// GetMemory returns the agent's memory
func (r *ReActAgent) GetMemory() Memory {
	return r.config.Memory
}

// StateDict returns the agent's state for serialization
func (r *ReActAgent) StateDict() map[string]any {
	state := r.StateModuleBase.StateDict()
	state["name"] = r.Name()
	state["system_prompt"] = r.SystemPrompt()
	if r.config.Memory != nil {
		state["memory"] = r.config.Memory
	}
	return state
}

// LoadStateDict loads the agent's state
func (r *ReActAgent) LoadStateDict(ctx context.Context, state map[string]any) error {
	if err := r.StateModuleBase.LoadStateDict(ctx, state); err != nil {
		return err
	}
	return nil
}

// GetConfig returns the agent's configuration
func (r *ReActAgent) GetConfig() *ReActAgentConfig {
	return r.config
}

// GetModel returns the agent's model
func (r *ReActAgent) GetModel() model.ChatModel {
	return r.config.Model
}

// GetToolkit returns the agent's toolkit
func (r *ReActAgent) GetToolkit() tool.ToolProvider {
	return r.config.Toolkit
}

// SetSystemPrompt sets a new system prompt
func (r *ReActAgent) SetSystemPrompt(prompt string) {
	r.config.SystemPrompt = prompt
	if r.AgentBase != nil {
		r.AgentBase.SetSystemPrompt(prompt)
	}
}
