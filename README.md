# AgentScope Go

A Go implementation of the AgentScope framework - a multi-agent LLM application framework inspired by [agentscope-ai/agentscope](https://github.com/agentscope-ai/agentscope).

## Overview

AgentScope Go provides a comprehensive framework for building AI agent applications with the following features:

- **Agent System**: ReActAgent, UserAgent, and extensible agent base
- **Message System**: Rich content blocks including text, images, audio, video, and tool calls
- **Model Integration**: OpenAI API support with streaming
- **Tool System**: Register and call tools with JSON schema generation
- **Pipeline System**: Sequential, fan-out, and loop pipelines
- **Memory System**: History memory and vector memory with embedding support
- **MsgHub**: Message broadcasting between agents
- **Hooks**: Pre/post hooks for reply, print, and observe operations

## Project Structure

```
pkg/agentscope/
├── agent/          # Agent implementations
│   ├── base.go         # Agent base and interfaces
│   └── react_agent.go  # ReActAgent implementation
├── message/        # Message types and content blocks
│   ├── message.go      # Core message types
│   ├── blocks.go       # Content block constructors
│   └── helpers.go      # Helper methods
├── model/          # Model interfaces and implementations
│   ├── model.go        # Core model interfaces
│   ├── openai/         # OpenAI client
│   └── response_helpers.go
├── tool/           # Tool system
│   └── toolkit.go      # Toolkit implementation
├── pipeline/       # Pipeline and orchestration
│   └── pipeline.go     # Sequential, fan-out, loop pipelines
├── memory/         # Memory implementations
│   └── memory.go       # History and vector memory
├── types/          # Core type definitions
│   └── types.go        # Role, block types, etc.
└── utils/          # Utility functions
```

## Installation

```bash
go get github.com/tingly-io/agentscope-go
```

## Quick Start

### Basic Chat

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tingly-io/agentscope-go/pkg/agentscope/agent"
    "github.com/tingly-io/agentscope-go/pkg/agentscope/message"
    "github.com/tingly-io/agentscope-go/pkg/agentscope/memory"
    "github.com/tingly-io/agentscope-go/pkg/agentscope/model"
    "github.com/tingly-io/agentscope-go/pkg/agentscope/model/openai"
    "github.com/tingly-io/agentscope-go/pkg/agentscope/types"
)

func main() {
    // Create an OpenAI client
    modelClient := openai.NewClient(&model.ChatModelConfig{
        ModelName: "gpt-4o-mini",
        APIKey:    "your-api-key",
    })

    // Create a ReActAgent
    reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
        Name:         "assistant",
        SystemPrompt: "You are a helpful assistant.",
        Model:        modelClient,
        Memory:       memory.NewHistory(100),
    })

    ctx := context.Background()

    // Create a user message
    userMsg := message.NewMsg(
        "user",
        "Hello! What's the capital of France?",
        types.RoleUser,
    )

    // Get a response
    response, err := reactAgent.Reply(ctx, userMsg)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.GetTextContent())
}
```

### Using Tools

```go
// Create a toolkit
toolkit := tool.NewToolkit()

// Register a tool
weatherTool := &WeatherTool{}
toolkit.Register(weatherTool, &tool.RegisterOptions{
    GroupName: "basic",
})

// Create agent with tools
reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
    Name:         "assistant",
    SystemPrompt: "You are a helpful assistant with weather tools.",
    Model:        modelClient,
    Toolkit:      toolkit,
    Memory:       memory.NewHistory(100),
    MaxIterations: 5,
})

// Implement a tool
type WeatherTool struct{}

func (w *WeatherTool) Call(ctx context.Context, kwargs map[string]any) (*tool.ToolResponse, error) {
    city, _ := kwargs["city"].(string)
    // Fetch weather data...
    return tool.TextResponse(fmt.Sprintf("Weather in %s: Sunny, 25°C", city)), nil
}
```

### Using Pipelines

```go
import "github.com/tingly-io/agentscope-go/pkg/agentscope/pipeline"

// Sequential pipeline
pipe := pipeline.NewSequentialPipeline("process", []agent.Agent{
    summarizerAgent,
    translatorAgent,
})

responses, err := pipe.Run(ctx, inputMsg)

// Fan-out pipeline
fanOut := pipeline.NewFanOutPipeline("parallel", []agent.Agent{
    agent1, agent2, agent3,
})

responses, err := fanOut.Run(ctx, inputMsg)
```

### Using MsgHub

```go
// Create a message hub
hub := pipeline.NewMsgHub("room", []agent.Agent{agent1, agent2, agent3})

// All agents will receive broadcasts from each other
// When an agent calls Reply(), the message is automatically
// broadcasted to all other agents in the hub

hub.Close() // Clean up
```

## Architecture

### Message System

Messages support rich content blocks:

- `TextBlock`: Plain text content
- `ThinkingBlock`: Internal reasoning (for models that support it)
- `ToolUseBlock`: Tool/function calls
- `ToolResultBlock`: Results from tool execution
- `ImageBlock`: Images (URL or base64)
- `AudioBlock`: Audio clips
- `VideoBlock`: Video clips

```go
// Simple text message
msg := message.NewMsg("user", "Hello", types.RoleUser)

// Multi-modal message
msg := message.NewMsg("user", []message.ContentBlock{
    message.Text("What's in this image?"),
    message.URLImage("https://example.com/image.jpg"),
}, types.RoleUser)
```

### Agent Types

1. **ReActAgent**: Implements the ReAct (Reasoning + Acting) pattern with tool use
2. **UserAgent**: Represents user input in conversations
3. **AgentBase**: Base class for custom agent implementations

### Memory Types

1. **History**: Simple in-memory message buffer with max size
2. **VectorMemory**: Memory with embedding-based similarity search

```go
// Simple history
mem := memory.NewHistory(100)

// Vector memory (requires embedding model)
vecMem := memory.NewVectorMemory(1000, embeddingModel)
```

### Hooks

Agents support pre/post hooks for extensibility:

```go
agent.RegisterHook(types.HookTypePreReply, "log", func(ctx context.Context, a agent.Agent, kwargs map[string]any) (map[string]any, error) {
    fmt.Println("Before reply")
    return kwargs, nil
})
```

## Examples

See `cmd/examples/main.go` for comprehensive examples including:

- Simple chat
- Agent with tools
- Sequential pipelines
- MsgHub with multiple agents

Run examples:

```bash
cd cmd/examples
go run main.go
```

## Design Principles

This Go implementation follows idiomatic Go patterns while preserving the core architecture of AgentScope:

- **Interface-based design**: Easy to extend and mock
- **Context propagation**: Proper context.Context usage throughout
- **Error handling**: Explicit error returns
- **Concurrency**: Goroutine-safe implementations with proper locking
- **Streaming**: Support for streaming responses from LLM APIs

## Roadmap

- [ ] Additional model integrations (Anthropic, Gemini, Ollama)
- [ ] RAG (Retrieval-Augmented Generation) support
- [ ] Memory persistence (file, database)
- [ ] Distributed agent communication
- [ ] Web UI / Studio
- [ ] More example agents and tools

## License

This project is inspired by and based on the [AgentScope](https://github.com/agentscope-ai/agentscope) framework.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
