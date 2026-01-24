package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/tingly-io/agentscope-go/pkg/agentscope/message"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/types"
)

// HookFunc represents a hook function
type HookFunc func(ctx context.Context, agent Agent, kwargs map[string]any) (map[string]any, error)

// PostHookFunc represents a post-hook function
type PostHookFunc func(ctx context.Context, agent Agent, kwargs map[string]any, msg *message.Msg) (*message.Msg, error)

// Agent is the base interface that all agents must implement
type Agent interface {
	// Reply generates a response to the given message
	Reply(ctx context.Context, msg *message.Msg) (*message.Msg, error)

	// Observe receives a message without generating a response
	Observe(ctx context.Context, msg *message.Msg) error

	// Name returns the agent's name
	Name() string

	// ID returns the agent's unique identifier
	ID() string

	// Print outputs a message
	Print(ctx context.Context, msg *message.Msg) error

	// SetConsoleOutputEnabled enables or disables console output
	SetConsoleOutputEnabled(enabled bool)

	// RegisterHook registers a hook function
	RegisterHook(hookType types.HookType, name string, fn any) error

	// RemoveHook removes a hook function
	RemoveHook(hookType types.HookType, name string) error
}

// AgentBase provides common functionality for all agents
type AgentBase struct {
	id                   string
	name                 string
	systemPrompt         string
	disableConsoleOutput bool

	mu sync.RWMutex

	preReplyHooks   map[string]HookFunc
	postReplyHooks  map[string]PostHookFunc
	prePrintHooks   map[string]HookFunc
	postPrintHooks  map[string]PostHookFunc
	preObserveHooks map[string]HookFunc
	postObserveHooks map[string]PostHookFunc

	subscribers map[string][]Agent // msghub name -> list of subscribers
}

// NewAgentBase creates a new agent base
func NewAgentBase(name string, systemPrompt string) *AgentBase {
	return &AgentBase{
		id:                   types.GenerateID(),
		name:                 name,
		systemPrompt:         systemPrompt,
		disableConsoleOutput: false,
		preReplyHooks:        make(map[string]HookFunc),
		postReplyHooks:       make(map[string]PostHookFunc),
		prePrintHooks:        make(map[string]HookFunc),
		postPrintHooks:       make(map[string]PostHookFunc),
		preObserveHooks:      make(map[string]HookFunc),
		postObserveHooks:     make(map[string]PostHookFunc),
		subscribers:          make(map[string][]Agent),
	}
}

// ID returns the agent's unique identifier
func (a *AgentBase) ID() string {
	return a.id
}

// Name returns the agent's name
func (a *AgentBase) Name() string {
	return a.name
}

// SystemPrompt returns the agent's system prompt
func (a *AgentBase) SystemPrompt() string {
	return a.systemPrompt
}

// SetSystemPrompt sets the agent's system prompt
func (a *AgentBase) SetSystemPrompt(prompt string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.systemPrompt = prompt
}

// SetConsoleOutputEnabled enables or disables console output
func (a *AgentBase) SetConsoleOutputEnabled(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.disableConsoleOutput = !enabled
}

// Print outputs a message
func (a *AgentBase) Print(ctx context.Context, msg *message.Msg) error {
	a.mu.RLock()
	disable := a.disableConsoleOutput
	a.mu.RUnlock()

	if disable {
		return nil
	}

	// Run pre-print hooks
	if err := a.runPreHooks(ctx, types.HookTypePrePrint, msg, nil); err != nil {
		return err
	}

	// Print the message
	fmt.Printf("[%s] %s: %s\n", msg.Role, msg.Name, msg.GetTextContent())

	// Run post-print hooks
	if err := a.runPostHooks(ctx, types.HookTypePostPrint, msg, nil); err != nil {
		return err
	}

	return nil
}

// Observe receives a message without generating a response
func (a *AgentBase) Observe(ctx context.Context, msg *message.Msg) error {
	// Run pre-observe hooks
	kwargs := map[string]any{"message": msg}
	if err := a.runPreHooks(ctx, types.HookTypePreObserve, msg, kwargs); err != nil {
		return err
	}

	// Default implementation does nothing
	// Subclasses can override to store messages in memory

	// Run post-observe hooks
	if err := a.runPostHooks(ctx, types.HookTypePostObserve, msg, kwargs); err != nil {
		return err
	}

	return nil
}

// RegisterHook registers a hook function
func (a *AgentBase) RegisterHook(hookType types.HookType, name string, fn any) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch hookType {
	case types.HookTypePreReply:
		if fn, ok := fn.(HookFunc); ok {
			a.preReplyHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for pre_reply")
		}
	case types.HookTypePostReply:
		if fn, ok := fn.(PostHookFunc); ok {
			a.postReplyHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for post_reply")
		}
	case types.HookTypePrePrint:
		if fn, ok := fn.(HookFunc); ok {
			a.prePrintHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for pre_print")
		}
	case types.HookTypePostPrint:
		if fn, ok := fn.(PostHookFunc); ok {
			a.postPrintHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for post_print")
		}
	case types.HookTypePreObserve:
		if fn, ok := fn.(HookFunc); ok {
			a.preObserveHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for pre_observe")
		}
	case types.HookTypePostObserve:
		if fn, ok := fn.(PostHookFunc); ok {
			a.postObserveHooks[name] = fn
		} else {
			return fmt.Errorf("invalid hook function type for post_observe")
		}
	default:
		return fmt.Errorf("unknown hook type: %s", hookType)
	}

	return nil
}

// RemoveHook removes a hook function
func (a *AgentBase) RemoveHook(hookType types.HookType, name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch hookType {
	case types.HookTypePreReply:
		delete(a.preReplyHooks, name)
	case types.HookTypePostReply:
		delete(a.postReplyHooks, name)
	case types.HookTypePrePrint:
		delete(a.prePrintHooks, name)
	case types.HookTypePostPrint:
		delete(a.postPrintHooks, name)
	case types.HookTypePreObserve:
		delete(a.preObserveHooks, name)
	case types.HookTypePostObserve:
		delete(a.postObserveHooks, name)
	default:
		return fmt.Errorf("unknown hook type: %s", hookType)
	}

	return nil
}

// ResetSubscribers resets the subscribers for a given msghub
func (a *AgentBase) ResetSubscribers(msghubName string, subscribers []Agent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var filtered []Agent
	for _, sub := range subscribers {
		if sub.ID() != a.id {
			filtered = append(filtered, sub)
		}
	}
	a.subscribers[msghubName] = filtered
}

// RemoveSubscribers removes subscribers for a given msghub
func (a *AgentBase) RemoveSubscribers(msghubName string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.subscribers, msghubName)
}

// BroadcastToSubscribers broadcasts a message to all subscribers
func (a *AgentBase) BroadcastToSubscribers(ctx context.Context, msg *message.Msg) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, subscribers := range a.subscribers {
		for _, sub := range subscribers {
			if err := sub.Observe(ctx, msg); err != nil {
				return err
			}
		}
	}

	return nil
}

// runPreHooks runs all pre-hooks for a given type
func (a *AgentBase) runPreHooks(ctx context.Context, hookType types.HookType, msg *message.Msg, kwargs map[string]any) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var hooks map[string]HookFunc
	switch hookType {
	case types.HookTypePreReply:
		hooks = a.preReplyHooks
	case types.HookTypePrePrint:
		hooks = a.prePrintHooks
	case types.HookTypePreObserve:
		hooks = a.preObserveHooks
	default:
		return nil
	}

	for _, hook := range hooks {
		if kwargs == nil {
			kwargs = make(map[string]any)
		}
		kwargs["message"] = msg
		result, err := hook(ctx, a, kwargs)
		if err != nil {
			return err
		}
		if result != nil {
			kwargs = result
		}
	}

	return nil
}

// runPostHooks runs all post-hooks for a given type
func (a *AgentBase) runPostHooks(ctx context.Context, hookType types.HookType, msg *message.Msg, kwargs map[string]any) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var hooks map[string]PostHookFunc
	switch hookType {
	case types.HookTypePostReply:
		hooks = a.postReplyHooks
	case types.HookTypePostPrint:
		hooks = a.postPrintHooks
	case types.HookTypePostObserve:
		hooks = a.postObserveHooks
	default:
		return nil
	}

	currentMsg := msg
	for _, hook := range hooks {
		if kwargs == nil {
			kwargs = make(map[string]any)
		}
		kwargs["message"] = msg
		result, err := hook(ctx, a, kwargs, currentMsg)
		if err != nil {
			return err
		}
		if result != nil {
			currentMsg = result
		}
	}

	return nil
}
