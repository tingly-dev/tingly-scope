# Tingly Loop

An autonomous AI agent **loop controller** based on the [Ralph pattern](https://ghuntley.com/ralph/). Tingly-loop manages the iteration loop and calls **agents** to do the actual work.

## Architecture

```
tingly-loop (Loop Controller)
    ├── Manages tasks state (docs/loop/tasks.json)
    ├── Tracks progress (docs/loop/progress.md)
    ├── Handles git branch switching
    └── Calls Agent each iteration
            │
            ├── claude CLI (default, like ralph)
            ├── tingly-code
            └── custom subprocess
                    │
                    └── Has full tool access (file, bash, etc.)
```

Unlike ralph which only supports external CLI tools, tingly-loop supports multiple agent types while providing the same loop control pattern.

## Installation

```bash
cd example/tingly-loop
go build -o tingly-loop .
```

## Usage

### Basic Usage (claude CLI agent - like ralph)

```bash
# In a project directory with docs/loop/tasks.json
cd /path/to/project
tingly-loop run

# The claude CLI will be called with --dangerously-skip-permissions --print
```

### Using tingly-code as agent

```bash
tingly-loop run --agent tingly-code

# Or specify binary path
tingly-loop run --agent tingly-code --agent-binary /path/to/tingly-code
```

### Using custom subprocess

```bash
tingly-loop run --agent subprocess --agent-binary ./my-agent --agent-arg "--flag"
```

### CLI Commands

```bash
# Run the loop
tingly-loop run [options]

# Show status without running
tingly-loop status [options]

# Interactively create tasks.json
tingly-loop init [options]

# Generate tasks.json from description using AI
tingly-loop generate "<feature description>" [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--tasks, -t` | `docs/loop/tasks.json` | Path to tasks JSON file |
| `--progress` | `docs/loop/progress.md` | Path to progress log |
| `--max-iterations, -n` | `10` | Maximum loop iterations |
| `--agent` | `claude` | Agent type: claude, tingly-code, subprocess |
| `--agent-binary` | (auto-detect) | Path to agent binary |
| `--agent-arg` | (none) | Additional args for subprocess (repeatable) |
| `--config, -c` | (none) | Config file for agent |
| `--instructions, -i` | (embedded) | Custom instructions for claude agent |
| `--workdir, -w` | (current dir) | Working directory |

## Tasks Format

Create a `docs/loop/tasks.json` file in your project:

```json
{
  "project": "MyProject",
  "branchName": "feature/my-feature",
  "description": "Feature description",
  "userStories": [
    {
      "id": "US-001",
      "title": "Story title",
      "description": "As a user, I want X so that Y",
      "acceptanceCriteria": [
        "Specific criterion 1",
        "Typecheck passes"
      ],
      "priority": 1,
      "passes": false,
      "notes": ""
    }
  ]
}
```

### Tasks Fields

- `project`: Project name
- `branchName`: Git branch to work on (created if doesn't exist)
- `description`: Overall feature description
- `userStories`: List of stories to implement
  - `id`: Unique identifier (e.g., US-001)
  - `title`: Short title
  - `description`: Full story description
  - `acceptanceCriteria`: List of verifiable criteria
  - `priority`: Execution order (lower = higher priority)
  - `passes`: Whether the story is complete (agent sets to true)
  - `notes`: Optional notes

## Progress Tracking

The `docs/loop/progress.md` file tracks iterations. The agent appends to this file after completing each story.

## Completion

The loop terminates when:

1. **Success**: Agent outputs `<promise>COMPLETE</promise>` (all stories pass)
2. **Max iterations**: Reached the maximum iteration limit

## Agent Types

### claude (Default)

Calls the claude CLI directly, similar to ralph. The instructions are passed via stdin.

```bash
tingly-loop run --agent claude
```

Requirements:
- `claude` CLI must be installed and in PATH

### tingly-code

Calls tingly-code in `auto` mode with the iteration prompt.

```bash
tingly-loop run --agent tingly-code --config /path/to/tingly-config.toml
```

Requirements:
- tingly-code binary built or available

### subprocess

Calls any custom binary that accepts the prompt via stdin.

```bash
tingly-loop run --agent subprocess \
  --agent-binary ./my-agent \
  --agent-arg "--verbose"
```

## Example Workflow

1. Create tasks for your feature (`tingly-loop init` or `tingly-loop generate`)
2. Run `tingly-loop run`
3. Each iteration:
   - tingly-loop builds the prompt (tasks + progress)
   - Agent executes with full tool access
   - Agent commits changes and updates tasks
   - tingly-loop checks for completion signal
4. When all stories pass, loop exits successfully

## Comparison with Ralph

| Feature | Ralph | Tingly Loop |
|---------|-------|-------------|
| Implementation | Bash script | Go program |
| Agent Types | claude, amp | claude, tingly-code, subprocess |
| State Management | File I/O | File I/O + Go structs |
| Error Handling | Basic | Structured with Go errors |
| Loop Control | for loop in bash | Go loop controller |
| Extensibility | Limited | Pluggable Agent interface |
| Default Paths | `prd.json`, `progress.txt` | `docs/loop/tasks.json`, `docs/loop/progress.md` |

## License

MIT
