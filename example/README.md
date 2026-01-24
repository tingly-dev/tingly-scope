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

### Tingly Code (`tingly-code/`)
A coding agent based on the Python tinglyagent project, migrated to Go.

**Features:**
- ReAct agent with file and bash tools
- Interactive chat mode with `/quit`, `/help`, `/clear` commands
- Automated task resolution with `auto` command
- Patch creation from git changes with `diff` command
- TOML configuration with environment variable substitution
- Persistent bash session across tool calls

**Tools:**
- `view_file`: Read file contents with line numbers
- `replace_file`: Create or overwrite files
- `edit_file`: Replace specific text (requires exact match)
- `glob_files`: Find files by pattern
- `grep_files`: Search file contents
- `list_directory`: List files and directories
- `execute_bash`: Run shell commands
- `job_done`: Mark task completion

**Usage:**
```bash
cd tingly-code
go build -o tingly-code ./cmd/tingly-code
./tingly-code chat        # Interactive mode
./tingly-code auto "task" # Automated mode
./tingly-code diff        # Create patch file
./tingly-code init-config # Generate config
```

**Configuration:**
Create a `tingly-config.toml` file or use the `init-config` command:

```toml
[agent]
name = "tingly"

[agent.model]
model_type = "openai"
model_name = "gpt-4o"
api_key = "${OPENAI_API_KEY}"
base_url = ""
temperature = 0.3
max_tokens = 8000

[agent.prompt]
system = "Custom system prompt (optional)"

[agent.shell]
init_commands = []
verbose_init = false
```

## Configuration

Both chat and react-fetch examples use the Tingly API. Configure your credentials in the respective `main.go` files:

```go
const (
    baseURL   = "http://localhost:12580/tingly/claude_code"
    modelName = "tingly/cc"
    apiToken  = "your-api-token"
)
```

The tingly-code example uses a TOML configuration file with environment variable substitution (e.g., `${OPENAI_API_KEY}`).

## Requirements

- Go 1.16 or higher
- Access to Tingly CC model API (or compatible API)

