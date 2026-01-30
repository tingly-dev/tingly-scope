package agent

// DecisionAction represents the action the human agent decides to take
type DecisionAction int

const (
	// DecisionActionContinue means the human agent wants the reactive agent to continue with a new instruction
	DecisionActionContinue DecisionAction = iota

	// DecisionActionTerminate means the human agent is satisfied and wants to terminate the workflow
	DecisionActionTerminate

	// DecisionActionRedirect means the human agent wants to change the approach
	DecisionActionRedirect
)

// String returns the string representation of the decision action
func (d DecisionAction) String() string {
	switch d {
	case DecisionActionContinue:
		return "CONTINUE"
	case DecisionActionTerminate:
		return "TERMINATE"
	case DecisionActionRedirect:
		return "REDIRECT"
	default:
		return "UNKNOWN"
	}
}

// Conclusion represents the reactive agent's conclusion returned to the human agent
type Conclusion struct {
	// Summary is a brief summary of what was accomplished
	Summary string

	// Steps lists the intermediate steps taken during execution
	Steps []string

	// SuggestedNextAction is an optional suggestion for what to do next
	SuggestedNextAction string

	// Confidence is the confidence level (0-1) in the completion
	Confidence float64

	// Artifacts contains any artifacts generated during execution
	Artifacts map[string]any

	// OriginalInput preserves the original user input for context
	OriginalInput string

	// Iterations counts how many ReAct iterations were used
	Iterations int
}

// NewConclusion creates a new conclusion with default values
func NewConclusion() *Conclusion {
	return &Conclusion{
		Steps:      make([]string, 0),
		Confidence: 0.0,
		Artifacts:  make(map[string]any),
	}
}

// AddStep adds a step to the conclusion
func (c *Conclusion) AddStep(step string) {
	c.Steps = append(c.Steps, step)
}

// AddArtifact adds an artifact to the conclusion
func (c *Conclusion) AddArtifact(key string, value any) {
	if c.Artifacts == nil {
		c.Artifacts = make(map[string]any)
	}
	c.Artifacts[key] = value
}

// IsComplete returns true if the conclusion indicates completion
func (c *Conclusion) IsComplete() bool {
	return c.Confidence >= 0.8
}

// HumanDecision represents the human agent's decision after evaluating a conclusion
type HumanDecision struct {
	// Action is the decision to take
	Action DecisionAction

	// NewInstruction contains the new instruction for the reactive agent (if continuing)
	NewInstruction string

	// Reasoning explains the decision
	Reasoning string

	// ModifiedApproach contains the new approach (if redirecting)
	ModifiedApproach string
}

// NewHumanDecision creates a new human decision
func NewHumanDecision(action DecisionAction) *HumanDecision {
	return &HumanDecision{
		Action: action,
	}
}

// ShouldContinue returns true if the workflow should continue
func (d *HumanDecision) ShouldContinue() bool {
	return d.Action == DecisionActionContinue || d.Action == DecisionActionRedirect
}

// ShouldTerminate returns true if the workflow should terminate
func (d *HumanDecision) ShouldTerminate() bool {
	return d.Action == DecisionActionTerminate
}

// IsRedirect returns true if this is a redirect decision
func (d *HumanDecision) IsRedirect() bool {
	return d.Action == DecisionActionRedirect
}
