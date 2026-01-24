# AgentScope Go Examples

This directory contains example applications demonstrating the AgentScope Go framework.

## Examples

### Chat (`chat/`)
A simple CLI chat assistant powered by the Tingly CC model.

**Features:**
- Single prompt mode for quick queries
- Interactive chat mode with conversation history
- Built-in commands (`/quit`, `/clear`, `/help`)

**Usage:**
```bash
cd chat
go build -o tingly-chat ./cmd/chat/main.go
./tingly-chat "what is 2+2?"
./tingly-chat  # Interactive mode
```

### ReAct Fetch (`react-fetch/`)
A ReAct (Reasoning + Acting) agent with a web_fetch tool.

**Features:**
- Multi-step reasoning with tool calling
- Web page fetching and content extraction
- Interactive CLI with example queries

**Usage:**
```bash
cd react-fetch
go build -o react-fetch ./cmd/react-fetch/main.go
./react-fetch
```

## Configuration

Both examples use the Tingly API. Configure your credentials in the respective `main.go` files:

```go
const (
    baseURL   = "http://localhost:12580/tingly/claude_code"
    modelName = "tingly/cc"
    apiToken  = "your-api-token"
)
```

## Requirements

- Go 1.16 or higher
- Access to Tingly CC model API
