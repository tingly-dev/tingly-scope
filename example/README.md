# Tingly Scope Examples

> Example applications for the Tingly Scope multi-agent LLM framework in Go.

This directory contains example applications demonstrating the Tingly Scope Go Agent framework.

## Examples

| Example | Purpose | Key Concepts |
|---------|---------|--------------|
| [Chat](#chat) | CLI chat assistant | Interactive conversation |
| [React Fetch](#react-fetch) | Web-fetching agent | ReAct pattern, tool calling |
| [Tingly Code](#tingly-code) | AI programming assistant | Full coding workflow, file tools |
| [Dual Act Demo](#dual-act-demo) | Two-agent collaboration | Planner + Executor pattern |
| [Simple](#simple) | Core framework demo | Agent, Pipeline, MsgHub |
| [Formatter Demo](#formatter-demo) | Console output formatting | Message formatting |
| [Tea Formatter Demo](#tea-formatter-demo) | Advanced output formatting | Tea-based formatting |

---

### Chat (`chat/`)
A simple CLI chat assistant powered by the Tingly CC model.

**Features:**
- Single prompt mode for quick queries
- Interactive chat mode with conversation history
- Built-in commands: `/quit`, `/exit`, `/q`, `/clear`, `/c`, `/help`, `/h`
- Colored terminal output with ANSI codes

**Usage:**
```bash
cd chat
go build -o tingly-chat ./cmd/chat/main.go
./tingly-chat "what is 2+2?"  # Single prompt mode
./tingly-chat                 # Interactive mode
./tingly-chat --help          # Show help
```

---

### React Fetch (`react-fetch/`)
A ReAct (Reasoning + Acting) agent with a web_fetch tool.

**Features:**
- Multi-step reasoning with tool calling
- Web page fetching and content extraction
- HTML parsing to extract titles, headings, and main content
- Interactive CLI with example queries
- Shows thinking process and tool execution

**Usage:**
```bash
cd react-fetch
go build -o react-fetch ./cmd/react-fetch/main.go
./react-fetch
# Example queries:
#   what's the title of https://example.com?
#   fetch https://www.python.org and tell me the latest Python version
```

---

### Tingly Code (`tingly-code/`)
A full-featured AI programming assistant based on the Python tinglyagent project, migrated to Go.

**Features:**
- ReAct agent with file and bash tools
- Interactive chat mode with `/quit`, `/help`, `/clear` commands
- Automated task resolution with `auto` command
- Dual mode with planner and executor agents (`dual` command)
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
./tingly-code chat              # Interactive mode
./tingly-code auto "task"       # Automated mode
./tingly-code dual "task"       # Dual mode (planner + executor)
./tingly-code diff              # Create patch file
./tingly-code init-config       # Generate config
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

---

### Dual Act Demo (`dualact-demo/`)
Demonstrates the DualActAgent pattern which splits thinking and acting into separate LLM calls.

**Features:**
- Two-agent collaboration: Planner (Human) + Developer (Reactive)
- Planner reviews work and decides: TERMINATE/CONTINUE/REDIRECT
- Developer writes code and runs tests
- TeaFormatter for beautiful console output

**Usage:**
```bash
cd dualact-demo
go run ./cmd/dualact-demo/main.go
```

---

### Simple (`simple/`)
A minimal example demonstrating the core Tingly Scope framework concepts using OpenAI.

**Features:**
- Simple chat with ReActAgent
- ReActAgent with custom tools (CalculatorTool)
- Sequential pipeline (multiple agents in sequence)
- MsgHub with multiple agents

**Usage:**
```bash
cd simple
OPENAI_API_KEY=your-key go run main.go
```

---

### Formatter Demo (`formatter_demo/`)
Demonstrates the ConsoleFormatter for formatting agent messages with rich output.

**Features:**
- User message formatting
- Assistant messages with tool use blocks
- Tool result formatting
- Complete tool call flow demonstration
- Verbose/Compact modes
- Colorize on/off modes

**Usage:**
```bash
cd formatter_demo
go run main.go
```

---

### Tea Formatter Demo (`tea_formatter_demo/`)
Demonstrates the TeaFormatter - an advanced formatter for richer terminal output.

**Features:**
- Advanced console formatting
- Complete tool call flow visualization
- Compact TeaFormatter variant
- Monochrome (no colors) mode
- Color-coded role indicators

**Usage:**
```bash
cd tea_formatter_demo
go run main.go
```

---

## Configuration

### Tingly CC API (chat, react-fetch, dualact-demo)

Configure credentials in the respective `main.go` files:

```go
const (
    baseURL   = "http://localhost:12580/tingly/claude_code"
    modelName = "tingly/cc"
    apiToken  = "your-api-token"
)
```

### OpenAI/Anthropic (tingly-code, simple)

The tingly-code example uses a TOML configuration file with environment variable substitution:

```bash
export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"
```

The simple example requires the `OPENAI_API_KEY` environment variable.

---

## Requirements

- Go 1.16 or higher
- Access to Tingly CC model API (for chat, react-fetch, dualact-demo)
- OpenAI or Anthropic API key (for tingly-code, simple)
