package agent

import "fmt"

// DualActConfig holds the configuration for a dual act agent
type DualActConfig struct {
	// Human is the human-like decision agent that evaluates conclusions
	Human *ReActAgent

	// Reactive is the reactive execution agent that performs the work
	Reactive *ReActAgent

	// MaxHRLoops is the maximum number of H-R interaction loops
	// Default is 3
	MaxHRLoops int

	// HumanDecisionPrompt is the custom prompt for the human agent's decision making
	// If empty, a default prompt will be used
	HumanDecisionPrompt string

	// ReactiveTaskPrompt is the custom prompt template for the reactive agent's tasks
	// If empty, a default prompt will be used
	ReactiveTaskPrompt string

	// ConclusionFormatPrompt specifies how the reactive agent should format conclusions
	// If empty, a default format will be used
	ConclusionFormatPrompt string

	// EnableVerboseLogging enables detailed logging of H-R interactions
	EnableVerboseLogging bool
}

// Validate validates the dual act configuration
func (c *DualActConfig) Validate() error {
	if c.Human == nil {
		return fmt.Errorf("human agent is required")
	}
	if c.Reactive == nil {
		return fmt.Errorf("reactive agent is required")
	}
	if c.MaxHRLoops <= 0 {
		return fmt.Errorf("max HR loops must be positive")
	}
	return nil
}

// DefaultDualActConfig creates a default configuration with sensible defaults
func DefaultDualActConfig(human, reactive *ReActAgent) *DualActConfig {
	return &DualActConfig{
		Human:               human,
		Reactive:            reactive,
		MaxHRLoops:          3,
		HumanDecisionPrompt: "",
		EnableVerboseLogging: false,
	}
}

// DualActOption is a functional option for configuring DualActAgent
type DualActOption func(*DualActConfig)

// WithMaxHRLoops sets the maximum number of H-R loops
func WithMaxHRLoops(max int) DualActOption {
	return func(c *DualActConfig) {
		c.MaxHRLoops = max
	}
}

// WithHumanDecisionPrompt sets a custom prompt for human agent decisions
func WithHumanDecisionPrompt(prompt string) DualActOption {
	return func(c *DualActConfig) {
		c.HumanDecisionPrompt = prompt
	}
}

// WithReactiveTaskPrompt sets a custom prompt template for reactive agent tasks
func WithReactiveTaskPrompt(prompt string) DualActOption {
	return func(c *DualActConfig) {
		c.ReactiveTaskPrompt = prompt
	}
}

// WithConclusionFormatPrompt sets a custom format for conclusions
func WithConclusionFormatPrompt(prompt string) DualActOption {
	return func(c *DualActConfig) {
		c.ConclusionFormatPrompt = prompt
	}
}

// WithVerboseLogging enables verbose logging
func WithVerboseLogging() DualActOption {
	return func(c *DualActConfig) {
		c.EnableVerboseLogging = true
	}
}

// ApplyOptions applies the given options to the configuration
func (c *DualActConfig) ApplyOptions(opts []DualActOption) {
	for _, opt := range opts {
		opt(c)
	}
}
