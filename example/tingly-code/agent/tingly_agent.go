package agent

import (
	"context"
	"fmt"
	"os"

	"example/tingly-code/config"
	"example/tingly-code/tools"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/agent"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/message"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/model"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/model/anthropic"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/model/openai"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/tool"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/types"
)

// Default system prompt for the Tingly agent
const defaultSystemPrompt = `You are Tingly, a professional AI programming assistant.

You have access to various tools to help with software engineering tasks. Use them proactively to assist the user and complete task.

## Available Tools

### File Operations
- view_file: Read file contents with line numbers
- replace_file: Create or overwrite a file with content
- edit_file: Replace a specific text in a file (requires exact match)
- glob_files: Find files by name pattern (e.g., **/*.py, src/**/*.ts)
- grep_files: Search file contents using regex
- list_directory: List files and directories

### Bash Execution
- execute_bash: Run shell commands (avoid using for file operations - use dedicated tools instead)

### Task Completion
- job_done: Mark the task as complete when you have successfully finished the user's request

## Guidelines

1. Use specialized tools over bash commands:
   - Use View/LS instead of cat/head/tail/ls
   - Use GlobTool instead of find
   - Use GrepTool instead of grep
   - Use Edit/Replace instead of sed/awk
   - Use Write instead of echo redirection

2. Before editing files, always read them first to understand context.

3. For unique string replacement in Edit, provide at least 3-5 lines of context.

4. Use batch_tool when you need to run multiple independent operations.

5. Be concise in your responses - the user sees output in a terminal.

6. Provide code references in the format "path/to/file.py:42" for easy navigation.

7. Call job_done if the task completed.

Always respond in English.
Always respond with exactly one tool call.`

// ModelFactory creates a ChatModel based on configuration
type ModelFactory struct{}

// NewModelFactory creates a new model factory
func NewModelFactory() *ModelFactory {
	return &ModelFactory{}
}

// CreateModel creates a ChatModel from the given configuration
func (mf *ModelFactory) CreateModel(cfg *config.ModelConfig) (model.ChatModel, error) {
	return createModelFromConfig(cfg)
}

// CreateTinglyAgent creates a TinglyAgent from configuration
func CreateTinglyAgent(cfg *config.AgentConfig, workDir string) (*agent.ReActAgent, error) {
	// Create model
	factory := NewModelFactory()
	chatModel, err := factory.CreateModel(&cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Create toolkit
	tk := tool.NewToolkit()

	// Create file tools
	fileTools := tools.NewFileTools(workDir)
	registerFileTools(tk, fileTools)

	// Create and register bash tools
	bashSession := tools.GetGlobalBashSession()
	tools.ConfigureBash(cfg.Shell.InitCommands, cfg.Shell.VerboseInit)
	bashTools := tools.NewBashTools(bashSession)
	registerBashTools(tk, bashTools)

	// Create and register notebook tools
	notebookTools := tools.NewNotebookTools(workDir)
	registerNotebookTools(tk, notebookTools)

	// Create and register batch tool
	batchTool := tools.GetGlobalBatchTool()
	registerBatchTools(tk, batchTool, fileTools, bashTools, notebookTools)

	// Get system prompt
	systemPrompt := cfg.Prompt.System
	if systemPrompt == "" {
		systemPrompt = defaultSystemPrompt
	}

	// Create memory
	memory := agent.NewSimpleMemory(100)

	// Create ReAct agent
	reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:         cfg.Name,
		SystemPrompt: systemPrompt,
		Model:        chatModel,
		Toolkit:      tk,
		Memory:       memory,
		MaxIterations: 20,
		Temperature:   &cfg.Model.Temperature,
		MaxTokens:     &cfg.Model.MaxTokens,
	})

	return reactAgent, nil
}

// TinglyAgent wraps ReActAgent with Tingly-specific functionality
type TinglyAgent struct {
	*agent.ReActAgent
	fileTools *tools.FileTools
	bashTools *tools.BashTools
	workDir   string
}

// NewTinglyAgent creates a new TinglyAgent
func NewTinglyAgent(cfg *config.AgentConfig, workDir string) (*TinglyAgent, error) {
	reactAgent, err := CreateTinglyAgent(cfg, workDir)
	if err != nil {
		return nil, err
	}

	fileTools := tools.NewFileTools(workDir)
	bashSession := tools.GetGlobalBashSession()
	tools.ConfigureBash(cfg.Shell.InitCommands, cfg.Shell.VerboseInit)
	bashTools := tools.NewBashTools(bashSession)

	return &TinglyAgent{
		ReActAgent: reactAgent,
		fileTools:  fileTools,
		bashTools:  bashTools,
		workDir:    workDir,
	}, nil
}

// NewTinglyAgentFromConfigFile creates a TinglyAgent from a config file
func NewTinglyAgentFromConfigFile(configPath, workDir string) (*TinglyAgent, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewTinglyAgent(&cfg.Agent, workDir)
}

// NewTinglyAgentFromDefaultConfig creates a TinglyAgent from default config locations
func NewTinglyAgentFromDefaultConfig(workDir string) (*TinglyAgent, error) {
	cfg, err := config.LoadConfigFromDefaultLocations()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewTinglyAgent(&cfg.Agent, workDir)
}

// Reply handles a user message
func (ta *TinglyAgent) Reply(ctx context.Context, msg *message.Msg) (*message.Msg, error) {
	return ta.ReActAgent.Reply(ctx, msg)
}

// RunSinglePrompt runs the agent with a single prompt and returns the response
func (ta *TinglyAgent) RunSinglePrompt(ctx context.Context, prompt string) (string, error) {
	msg := message.NewMsg(
		"user",
		[]message.ContentBlock{message.Text(prompt)},
		types.RoleUser,
	)

	resp, err := ta.Reply(ctx, msg)
	if err != nil {
		return "", err
	}

	// Check for nil response
	if resp == nil {
		return "", fmt.Errorf("agent returned nil response")
	}

	// Extract text from response
	var result string
	blocks, ok := resp.Content.([]message.ContentBlock)
	if ok {
		for _, block := range blocks {
			if textBlock, ok := block.(*message.TextBlock); ok {
				result += textBlock.Text
			}
		}
	}

	return result, nil
}

// IsJobDone checks if the agent has completed the task
func (ta *TinglyAgent) IsJobDone(msg *message.Msg) bool {
	if msg == nil {
		return true
	}

	// Check metadata for interrupted flag
	if msg.Metadata != nil {
		if interrupted, ok := msg.Metadata["_is_interrupted"].(bool); ok && interrupted {
			return true
		}
	}

	return false
}

// SetWorkDir sets the working directory for file operations
func (ta *TinglyAgent) SetWorkDir(dir string) {
	ta.workDir = dir
	ta.fileTools.SetWorkDir(dir)
}

// GetWorkDir returns the current working directory
func (ta *TinglyAgent) GetWorkDir() string {
	return ta.workDir
}

// createModelFromConfig creates a model from config using tingly-scope library clients
func createModelFromConfig(cfg *config.ModelConfig) (model.ChatModel, error) {
	// Get API key from config or environment
	apiKey := cfg.APIKey
	if apiKey == "" || (len(apiKey) > 0 && apiKey[0] == '$') {
		// Try to get from environment
		envKey := ""
		if cfg.ModelType == "openai" {
			envKey = "OPENAI_API_KEY"
		} else if cfg.ModelType == "anthropic" {
			envKey = "ANTHROPIC_API_KEY"
		}
		if envKey != "" {
			apiKey = os.Getenv(envKey)
		}
	}

	// Determine base URL
	baseURL := cfg.BaseURL
	if baseURL == "" {
		// Use default base URL based on model type
		if cfg.ModelType == "openai" {
			baseURL = "https://api.openai.com/v1"
		} else if cfg.ModelType == "anthropic" {
			baseURL = "https://api.anthropic.com"
		}
	}

	// Create appropriate model client based on type
	switch cfg.ModelType {
	case "anthropic":
		return anthropic.NewClient(&anthropic.Config{
			ModelName: cfg.ModelName,
			APIKey:    apiKey,
			BaseURL:   baseURL,
			MaxTokens: cfg.MaxTokens,
			Stream:    false,
		}), nil

	case "openai":
		return openai.NewClient(&model.ChatModelConfig{
			ModelName: cfg.ModelName,
			APIKey:    apiKey,
			BaseURL:   baseURL,
			Stream:    false,
		}), nil

	default:
		// Default to Anthropic-compatible client for custom endpoints (like Tingly)
		return anthropic.NewClient(&anthropic.Config{
			ModelName: cfg.ModelName,
			APIKey:    apiKey,
			BaseURL:   baseURL,
			MaxTokens: cfg.MaxTokens,
			Stream:    false,
		}), nil
	}
}

// registerFileTools registers file tools with the toolkit
func registerFileTools(tk *tool.Toolkit, ft *tools.FileTools) {
	tools := []struct {
		name        string
		fn          any
		description string
		params      map[string]any
	}{
		{
			name:        "view_file",
			fn:          ft.ViewFile,
			description: "Read file contents with line numbers. Provide the file path, and optionally limit (max lines to show) and offset (starting line number, 1-indexed).",
			params: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to the file to read",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of lines to show (optional)",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Starting line number (1-indexed, optional)",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			name:        "replace_file",
			fn:          ft.ReplaceFile,
			description: "Create or overwrite a file with content. Use this for writing new files or completely replacing existing files.",
			params: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to the file to create or overwrite",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Content to write to the file",
					},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			name:        "edit_file",
			fn:          ft.EditFile,
			description: "Replace a specific text in a file. The old_text must match exactly. Use at least 3-5 lines of context for unique matches.",
			params: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to the file to edit",
					},
					"old_text": map[string]any{
						"type":        "string",
						"description": "Text to replace (must match exactly)",
					},
					"new_text": map[string]any{
						"type":        "string",
						"description": "Replacement text",
					},
				},
				"required": []string{"path", "old_text", "new_text"},
			},
		},
		{
			name:        "glob_files",
			fn:          ft.GlobFiles,
			description: "Find files by name pattern. Supports glob patterns like **/*.go, src/**/*.ts, etc.",
			params: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "Glob pattern to match files (e.g., **/*.go, src/**/*.ts)",
					},
				},
				"required": []string{"pattern"},
			},
		},
		{
			name:        "grep_files",
			fn:          ft.GrepFiles,
			description: "Search file contents using a text pattern. Returns matching lines with file:line format.",
			params: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "Text pattern to search for in files",
					},
					"glob": map[string]any{
						"type":        "string",
						"description": "Glob pattern to filter files (default: **/*.go)",
					},
				},
				"required": []string{"pattern"},
			},
		},
		{
			name:        "list_directory",
			fn:          ft.ListDirectory,
			description: "List files and directories in a path",
			params: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to list (default: current directory)",
					},
				},
				"required": []string{},
			},
		},
	}

	for _, t := range tools {
		tk.Register(t.fn, &tool.RegisterOptions{
			GroupName:  "basic",
			FuncName:   t.name,
			JSONSchema: &model.ToolDefinition{
				Type:     "function",
				Function: model.FunctionDefinition{Name: t.name, Description: t.description, Parameters: t.params},
			},
		})
	}
}

// registerBashTools registers bash tools with the toolkit
func registerBashTools(tk *tool.Toolkit, bt *tools.BashTools) {
	tools := []struct {
		name        string
		fn          any
		description string
		params      map[string]any
	}{
		{
			name:        "execute_bash",
			fn:          bt.ExecuteBash,
			description: "Run a shell command. Avoid using for file operations - use dedicated file tools instead. Commands run in a bash shell with state preserved across calls.",
			params: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "Shell command to execute",
					},
					"timeout": map[string]any{
						"type":        "number",
						"description": "Timeout in seconds (default: 120)",
					},
				},
				"required": []string{"command"},
			},
		},
		{
			name:        "job_done",
			fn:          bt.JobDone,
			description: "Mark the task as complete. Call this when you have successfully finished the user's request.",
			params: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
	}

	for _, t := range tools {
		tk.Register(t.fn, &tool.RegisterOptions{
			GroupName:  "basic",
			FuncName:   t.name,
			JSONSchema: &model.ToolDefinition{
				Type:     "function",
				Function: model.FunctionDefinition{Name: t.name, Description: t.description, Parameters: t.params},
			},
		})
	}
}

// registerNotebookTools registers notebook tools with the toolkit
func registerNotebookTools(tk *tool.Toolkit, nt *tools.NotebookTools) {
	tools := []struct {
		name        string
		fn          any
		description string
		params      map[string]any
	}{
		{
			name:        "read_notebook",
			fn:          nt.ReadNotebook,
			description: "Read a Jupyter notebook (.ipynb file) and return all cells with their outputs.",
			params: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"notebook_path": map[string]any{
						"type":        "string",
						"description": "Path to the Jupyter notebook file",
					},
				},
				"required": []string{"notebook_path"},
			},
		},
		{
			name:        "notebook_edit_cell",
			fn:          nt.NotebookEditCell,
			description: "Edit a cell in a Jupyter notebook. Supports replace, insert, and delete modes.",
			params: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"notebook_path": map[string]any{
						"type":        "string",
						"description": "Path to the Jupyter notebook file",
					},
					"cell_number": map[string]any{
						"type":        "integer",
						"description": "The index of the cell to edit (0-based)",
					},
					"new_source": map[string]any{
						"type":        "string",
						"description": "The new source for the cell",
					},
					"edit_mode": map[string]any{
						"type":        "string",
						"description": "The type of edit (replace, insert, delete)",
						"enum":         []string{"replace", "insert", "delete"},
					},
					"cell_type": map[string]any{
						"type":        "string",
						"description": "The type of cell (code or markdown), required for insert mode",
						"enum":         []string{"code", "markdown"},
					},
				},
				"required": []string{"notebook_path", "cell_number", "new_source"},
			},
		},
	}

	for _, t := range tools {
		tk.Register(t.fn, &tool.RegisterOptions{
			GroupName:  "basic",
			FuncName:   t.name,
			JSONSchema: &model.ToolDefinition{
				Type:     "function",
				Function: model.FunctionDefinition{Name: t.name, Description: t.description, Parameters: t.params},
			},
		})
	}
}

// registerBatchTools registers batch tool with the toolkit
func registerBatchTools(tk *tool.Toolkit, bt *tools.BatchTool, ft *tools.FileTools, bsht *tools.BashTools, nt *tools.NotebookTools) {
	// Register all tools with batch tool for parallel execution
	bt.Register("view_file", ft.ViewFile)
	bt.Register("replace_file", ft.ReplaceFile)
	bt.Register("edit_file", ft.EditFile)
	bt.Register("glob_files", ft.GlobFiles)
	bt.Register("grep_files", ft.GrepFiles)
	bt.Register("list_directory", ft.ListDirectory)
	bt.Register("execute_bash", bsht.ExecuteBash)
	bt.Register("read_notebook", nt.ReadNotebook)
	bt.Register("notebook_edit_cell", nt.NotebookEditCell)

	// Register batch tool itself
	tk.Register(bt.Batch, &tool.RegisterOptions{
		GroupName:  "basic",
		FuncName:   "batch_tool",
		JSONSchema: &model.ToolDefinition{
			Type: "function",
			Function: model.FunctionDefinition{
				Name: "batch_tool",
				Description: "Execute multiple tool calls in parallel. Reduces latency by running independent operations concurrently.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"description": map[string]any{
							"type":        "string",
							"description": "A short (3-5 word) description of the batch operation",
						},
						"invocations": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"tool_name": map[string]any{
										"type":        "string",
										"description": "The name of the tool to invoke",
									},
									"input": map[string]any{
										"type":        "object",
										"description": "Dictionary of input parameters for the tool",
									},
								},
								"required": []string{"tool_name"},
							},
							"description": "List of tool invocations to execute in parallel",
						},
					},
					"required": []string{"description", "invocations"},
				},
			},
		},
	})
}
