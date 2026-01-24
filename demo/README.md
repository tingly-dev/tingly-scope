# Tingly Chat

A simple and elegant CLI chat assistant powered by Tingly's CC model.

## Features

- 🚀 **Single Prompt Mode**: Quick one-off queries from command line
- 💬 **Interactive Chat Mode**: Full conversation history with context
- 🎨 **Color-Coded Output**: Easy-to-read colored responses
- 📝 **Conversation History**: Maintains context across multiple turns
- ✨ **Built-in Commands**: Clear history, get help, exit gracefully

## Installation

```bash
cd demo
go build -o tingly-chat ./cmd/chat/main.go
```

## Usage

### Single Prompt Mode

Quick one-off queries:

```bash
./tingly-chat "what is 2+2?"
./tingly-chat "write a haiku about programming"
./tingly-chat "explain quantum computing in simple terms"
```

### Interactive Mode

```bash
./tingly-chat
```

In interactive mode:
- Type your message and press Enter to chat
- Type `/quit`, `/exit`, or `/q` to exit
- Type `/clear` or `/c` to clear conversation history
- Type `/help` or `/h` to see available commands

### Help

```bash
./tingly-chat --help
```

## Configuration

The application uses the following environment variables (hardcoded in main.go):

- `ANTHROPIC_BASE_URL`: API base URL
- `ANTHROPIC_MODEL`: Model name (tingly/cc)
- `ANTHROPIC_AUTH_TOKEN`: API authentication token

To customize, edit the constants in `cmd/chat/main.go`:

```go
const (
    baseURL   = "your-api-url"
    modelName = "your-model-name"
    apiToken  = "your-api-token"
)
```

## Examples

```bash
# Quick question
./tingly-chat "what's the capital of France?"

# Creative writing
./tingly-chat "write a short story about a robot"

# Technical help
./tingly-chat "how do I reverse a string in Go?"

# Interactive session
./tingly-chat
```

## Requirements

- Go 1.16 or higher
- Access to Tingly CC model API

## License

MIT
