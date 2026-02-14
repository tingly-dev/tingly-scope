# Tingly Loop

An autonomous AI agent **loop controller** based on the [Ralph pattern](https://ghuntley.com/ralph/). Tingly-loop manages the iteration loop and calls **worker agents** to do the actual work.

## Architecture

```
tingly-loop (Loop Controller)
    ├── Manages PRD state (prd.json)
    ├── Tracks progress (progress.txt)
    ├── Handles git branch switching
    └── Calls Worker Agent each iteration
            │
            ├── claude CLI (default, like ralph)
            ├── tingly-code
            └── custom subprocess
                    │
                    └── Has full tool access (file, bash, etc.)
```

Unlike ralph which only supports external CLI tools, tingly-loop supports multiple worker types while providing the same loop control pattern.

## Installation

```bash
cd example/tingly-loop
go build -o tingly-loop .
```

## Usage

### Basic Usage (claude CLI worker - like ralph)

```bash
# In a project directory with prd.json
cd /path/to/project
tingly-loop run

# The claude CLI will be called with --dangerously-skip-permissions --print
```

### Using tingly-code as worker

```bash
tingly-loop run --worker tingly-code

# Or specify binary path
tingly-loop run --worker tingly-code --worker-binary /path/to/tingly-code
```

### Using custom subprocess

```bash
tingly-loop run --worker subprocess --worker-binary ./my-agent --worker-arg "--flag"
```

### CLI Commands

```bash
# Run the loop
tingly-loop run [options]

# Show status without running
tingly-loop status [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--prd, -p` | `prd.json` | Path to PRD JSON file |
| `--progress` | `progress.txt` | Path to progress log |
| `--max-iterations, -n` | `10` | Maximum loop iterations |
| `--worker` | `claude` | Worker type: claude, tingly-code, subprocess |
| `--worker-binary` | (auto-detect) | Path to worker binary |
| `--worker-arg` | (none) | Additional args for subprocess (repeatable) |
| `--config, -c` | (none) | Config file for worker |
| `--instructions, -i` | (embedded) | Custom instructions for claude worker |
| `--workdir, -w` | (current dir) | Working directory |

## PRD Format

Create a `prd.json` file in your project:

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

### PRD Fields

- `project`: Project name
- `branchName`: Git branch to work on (created if doesn't exist)
- `description`: Overall feature description
- `userStories`: List of stories to implement
  - `id`: Unique identifier (e.g., US-001)
  - `title`: Short title
  - `description`: Full story description
  - `acceptanceCriteria`: List of verifiable criteria
  - `priority`: Execution order (lower = higher priority)
  - `passes`: Whether the story is complete (worker sets to true)
  - `notes`: Optional notes

## Progress Tracking

The `progress.txt` file tracks iterations. The worker agent appends to this file after completing each story.

## Completion

The loop terminates when:

1. **Success**: Worker outputs `<promise>COMPLETE</promise>` (all stories pass)
2. **Max iterations**: Reached the maximum iteration limit

## Worker Types

### claude (Default)

Calls the claude CLI directly, similar to ralph. The instructions are passed via stdin.

```bash
tingly-loop run --worker claude
```

Requirements:
- `claude` CLI must be installed and in PATH

### tingly-code

Calls tingly-code in `auto` mode with the iteration prompt.

```bash
tingly-loop run --worker tingly-code --config /path/to/tingly-config.toml
```

Requirements:
- tingly-code binary built or available

### subprocess

Calls any custom binary that accepts the prompt via stdin.

```bash
tingly-loop run --worker subprocess \
  --worker-binary ./my-agent \
  --worker-arg "--verbose"
```

## Example Workflow

1. Create a PRD for your feature
2. Run `tingly-loop run`
3. Each iteration:
   - tingly-loop builds the prompt (PRD + progress)
   - Worker agent executes with full tool access
   - Worker commits changes and updates PRD
   - tingly-loop checks for completion signal
4. When all stories pass, loop exits successfully

## Comparison with Ralph

| Feature | Ralph | Tingly Loop |
|---------|-------|-------------|
| Implementation | Bash script | Go program |
| Worker Types | claude, amp | claude, tingly-code, subprocess |
| State Management | File I/O | File I/O + Go structs |
| Error Handling | Basic | Structured with Go errors |
| Loop Control | for loop in bash | Go loop controller |
| Extensibility | Limited | Pluggable Worker interface |

## License

MIT
