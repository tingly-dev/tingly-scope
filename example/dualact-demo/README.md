# Dual Act Agent Demo

This example demonstrates the **Dual Act Agent** framework - a two-tier interaction pattern for AI agents.

## Overview

The Dual Act Agent coordinates two ReAct agents:

1. **Human Agent (H)** - The Planner: Evaluates work and makes decisions
2. **Reactive Agent (R)** - The Executor: Performs tasks using tools

## How It Works

```
User Task → Dual Act Agent
                │
                ▼
    ┌─────────────────────────────┐
    │   H-R Interaction Loop      │
    │                             │
    │  1. H receives task/conclusion
    │  2. H evaluates progress
    │  3. H decides:              │
    │     - TERMINATE (done)      │
    │     - CONTINUE (more work)  │
    │     - REDIRECT (new approach)│
    │  4. If continuing:          │
    │     → R executes with tools │
    │     → R reports conclusion  │
    │     → Loop back to H        │
    └─────────────────────────────┘
                │
                ▼
           Final Result
```

## The Task

This demo implements a **bracket matching validator**:
- Function that checks if brackets `()`, `{}`, `[]` are properly matched
- Handles nested brackets and edge cases
- Includes comprehensive tests

## Running the Demo

This demo uses the **built-in test API** - no API keys required!

```bash
cd example/dualact-demo
go run cmd/dualact-demo/main.go
```

## What You'll See

1. **Initial Task**: The user asks for a bracket validator
2. **Loop 1**: R writes the code and tests → H evaluates
3. **Loop 2+**: If needed, H asks R to fix issues → R iterates
4. **Final**: H is satisfied → Returns complete solution

### Sample Output

```
======================================================================
DUAL ACT AGENT DEMO
======================================================================

Using built-in test API (localhost:12580)

📝 Task:
Create a Go function that validates bracket matching...

----------------------------------------------------------------------
Starting Dual Act execution...
----------------------------------------------------------------------

[DualAct] === H-R Loop 1 ===
[DualAct] Starting reactive execution...
📄 Writing file: validator.go (234 bytes)
🔧 Executing: go test
   ✅ All tests passed!
[DualAct] Reactive conclusion: Implementation complete (confidence: 0.90)
[DualAct] Human decision: TERMINATE

======================================================================
🎉 FINAL RESULT
======================================================================

## Task: Create a Go function that validates bracket matching...

**Summary:** Implementation complete with passing tests

**Steps Taken:**
  1. Created validator.go with IsValid function
  2. Created validator_test.go with comprehensive tests
  3. All tests passing (5/5)

**Final Decision:** Task completed successfully
```

## Key Features Demonstrated

### 1. Agent Configuration

```go
humanAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
    Name:    "planner",
    Model:   modelClient,
    SystemPrompt: "You are a technical planner...",
})

reactiveAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
    Name:    "developer",
    Model:   modelClient,
    Toolkit: toolkit,
    MaxIterations: 8,
})
```

### 2. Dual Act Agent Creation

```go
dualAct := agent.NewDualActAgentWithOptions(
    humanAgent,
    reactiveAgent,
    agent.WithMaxHRLoops(5),
    agent.WithVerboseLogging(),
)
```

### 3. Standard Agent Interface

```go
response, err := dualAct.Reply(ctx, userMsg)
```

### 4. Custom Tools

```go
toolkit.Register(&WriteFileTool{}, &tool.RegisterOptions{GroupName: "file"})
toolkit.Register(&RunCodeTool{}, &tool.RegisterOptions{GroupName: "execution"})
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithMaxHRLoops(n)` | Maximum H-R iterations | 3 |
| `WithVerboseLogging()` | Enable detailed logs | false |
| `WithHumanDecisionPrompt(p)` | Custom H prompt | built-in |
| `WithReactiveTaskPrompt(p)` | Custom R prompt | built-in |

## Use Cases

The Dual Act pattern is ideal for:

1. **Code Development**: Write code → Test → Fix → Iterate
2. **Data Analysis**: Analyze → Evaluate → Refine → Conclude
3. **Research**: Investigate → Review → Redirect if needed
4. **Multi-step Tasks**: Any task requiring planning + execution

## Architecture Benefits

- ✅ **Separation of Concerns**: Planning vs Execution
- ✅ **Iterative Refinement**: Multiple passes at quality
- ✅ **Self-Correction**: Redirect when approach is wrong
- ✅ **Tool Use**: Only executor needs tools
- ✅ **Standard Interface**: Works with existing Agent ecosystem
