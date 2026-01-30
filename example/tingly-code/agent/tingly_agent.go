package agent

import (
	"context"
	"fmt"
	"os"

	"example/tingly-code/config"
	"example/tingly-code/tools"
	"github.com/tingly-dev/tingly-scope/pkg/agent"
	"github.com/tingly-dev/tingly-scope/pkg/formatter"
	"github.com/tingly-dev/tingly-scope/pkg/message"
	"github.com/tingly-dev/tingly-scope/pkg/model"
	"github.com/tingly-dev/tingly-scope/pkg/model/anthropic"
	"github.com/tingly-dev/tingly-scope/pkg/model/openai"
	"github.com/tingly-dev/tingly-scope/pkg/tool"
	"github.com/tingly-dev/tingly-scope/pkg/types"
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

### Shell Management
- task_output: Get output from a running or completed background shell
- kill_shell: Kill a running background shell process

### Task Management
- task_create: Create a new task in the task list
- task_get: Get a task by ID from the task list
- task_update: Update a task in the task list
- task_list: List all tasks in the task list

### User Interaction
- ask_user_question: Ask the user questions during execution

### Jupyter Notebook
- read_notebook: Read Jupyter notebook contents
- notebook_edit_cell: Edit notebook cell

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

5. Use task management tools to track progress on complex multi-step tasks.

6. Use ask_user_question when you need clarification or user input during execution.

7. Be concise in your responses - the user sees output in a terminal.

8. Provide code references in the format "path/to/file.py:42" for easy navigation.

9. Call job_done if the task completed.

Always respond in English.
Always respond with exactly one tool call.`

// ModelFactory creates a ChatModel based on configuration
type ModelFactory struct{}

// NewModelFactory creates a new model factory
func NewModelFactory() *ModelFactory {
	return &ModelFactory{}
}

// CreateModel creates a model client from the given configuration.
// Returns a model.ChatModel interface implemented by SDK adapters:
// - For Anthropic: *anthropic.SDKAdapter
// - For OpenAI: *openai.SDKAdapter
func (mf *ModelFactory) CreateModel(cfg *config.ModelConfig) (model.ChatModel, error) {
	return createModelFromConfig(cfg)
}

// CreateTinglyAgent creates a TinglyAgent from configuration
func CreateTinglyAgent(cfg *config.AgentConfig, toolsConfig *config.ToolsConfig, workDir string) (*agent.ReActAgent, error) {
	// Create model
	factory := NewModelFactory()
	chatModel, err := factory.CreateModel(&cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Create type-safe toolkit and register all tools
	tt := tools.NewTypedToolkit()

	// Register file tools
	fileTools := tools.NewFileTools(workDir)
	tt.RegisterAll(fileTools, map[string]string{
		"ViewFile":      tools.ToolDescViewFile,
		"ReplaceFile":   tools.ToolDescReplaceFile,
		"EditFile":      tools.ToolDescEditFile,
		"GlobFiles":     tools.ToolDescGlobFiles,
		"GrepFiles":     tools.ToolDescGrepFiles,
		"ListDirectory": tools.ToolDescListDirectory,
	})

	// Register bash tools
	bashSession := tools.GetGlobalBashSession()
	tools.ConfigureBash(cfg.Shell.InitCommands, cfg.Shell.VerboseInit)
	bashTools := tools.NewBashTools(bashSession)
	tt.RegisterAll(bashTools, map[string]string{
		"ExecuteBash": tools.ToolDescExecuteBash,
		"JobDone":     tools.ToolDescJobDone,
	})

	// Register notebook tools
	notebookTools := tools.NewNotebookTools(workDir)
	tt.RegisterAll(notebookTools, map[string]string{
		"ReadNotebook":     tools.ToolDescReadNotebook,
		"NotebookEditCell": tools.ToolDescNotebookEditCell,
	})

	// Register shell management tools
	shellManagementTools := tools.NewShellManagementTools()
	tt.RegisterAll(shellManagementTools, map[string]string{
		"TaskOutput": tools.ToolDescTaskOutput,
		"KillShell":  tools.ToolDescKillShell,
	})

	// Register task management tools
	taskManagementTools := tools.NewTaskManagementTools()
	tt.RegisterAll(taskManagementTools, map[string]string{
		"TaskCreate": tools.ToolDescTaskCreate,
		"TaskGet":    tools.ToolDescTaskGet,
		"TaskUpdate": tools.ToolDescTaskUpdate,
		"TaskList":   tools.ToolDescTaskList,
	})

	// Register user interaction tools
	userInteractionTools := tools.NewUserInteractionTools()
	tt.RegisterAll(userInteractionTools, map[string]string{
		"AskUserQuestion": tools.ToolDescAskUserQuestion,
	})

	// Apply tool filtering from config if specified
	if toolsConfig != nil && len(toolsConfig.Enabled) > 0 {
		tt.Filter(toolsConfig.Enabled)
	}

	// Get system prompt
	systemPrompt := cfg.Prompt.System
	if systemPrompt == "" {
		systemPrompt = defaultSystemPrompt
	}

	// Create memory
	memory := agent.NewSimpleMemory(100)

	// Create ReAct agent with type-safe toolkit
	reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          cfg.Name,
		SystemPrompt:  systemPrompt,
		Model:         chatModel,
		Toolkit:       &TypedToolkitAdapter{tt: tt},
		Memory:        memory,
		MaxIterations: 20,
		Temperature:   &cfg.Model.Temperature,
		MaxTokens:     &cfg.Model.MaxTokens,
	})

	// Set TeaFormatter as the default formatter for rich output
	reactAgent.SetFormatter(formatter.NewTeaFormatter())

	return reactAgent, nil
}

// TinglyAgent wraps ReActAgent with Tingly-specific functionality
type TinglyAgent struct {
	*agent.ReActAgent
	fileTools   *tools.FileTools
	bashTools   *tools.BashTools
	workDir     string
	toolsConfig *config.ToolsConfig
}

// NewTinglyAgent creates a new TinglyAgent
func NewTinglyAgent(cfg *config.AgentConfig, workDir string) (*TinglyAgent, error) {
	return NewTinglyAgentWithToolsConfig(cfg, nil, workDir)
}

// NewTinglyAgentWithToolsConfig creates a new TinglyAgent with tool filtering
func NewTinglyAgentWithToolsConfig(cfg *config.AgentConfig, toolsConfig *config.ToolsConfig, workDir string) (*TinglyAgent, error) {
	reactAgent, err := CreateTinglyAgent(cfg, toolsConfig, workDir)
	if err != nil {
		return nil, err
	}

	fileTools := tools.NewFileTools(workDir)
	bashSession := tools.GetGlobalBashSession()
	tools.ConfigureBash(cfg.Shell.InitCommands, cfg.Shell.VerboseInit)
	bashTools := tools.NewBashTools(bashSession)

	return &TinglyAgent{
		ReActAgent:  reactAgent,
		fileTools:   fileTools,
		bashTools:   bashTools,
		workDir:     workDir,
		toolsConfig: toolsConfig,
	}, nil
}

// NewTinglyAgentFromConfigFile creates a TinglyAgent from a config file
func NewTinglyAgentFromConfigFile(configPath, workDir string) (*TinglyAgent, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewTinglyAgentWithToolsConfig(&cfg.Agent, &cfg.Tools, workDir)
}

// NewTinglyAgentFromDefaultConfig creates a TinglyAgent from default config locations
func NewTinglyAgentFromDefaultConfig(workDir string) (*TinglyAgent, error) {
	cfg, err := config.LoadConfigFromDefaultLocations()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewTinglyAgentWithToolsConfig(&cfg.Agent, &cfg.Tools, workDir)
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

// createModelFromConfig creates a model from config using SDK adapters (NEW)
// This uses the official Anthropic and OpenAI SDKs with adapters to implement model.ChatModel.
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
		return anthropic.NewSDKAdapter(&anthropic.SDKConfig{
			Model:     cfg.ModelName,
			APIKey:    apiKey,
			BaseURL:   baseURL,
			MaxTokens: cfg.MaxTokens,
			Stream:    false,
		})

	case "openai":
		return openai.NewSDKAdapter(&openai.SDKConfig{
			Model:              cfg.ModelName,
			APIKey:             apiKey,
			BaseURL:            baseURL,
			Stream:             false,
			DefaultMaxTokens:   &cfg.MaxTokens,
			DefaultTemperature: &cfg.Temperature,
		})

	default:
		// Default to Anthropic-compatible SDK adapter for custom endpoints (like Tingly)
		return anthropic.NewSDKAdapter(&anthropic.SDKConfig{
			Model:     cfg.ModelName,
			APIKey:    apiKey,
			BaseURL:   baseURL,
			MaxTokens: cfg.MaxTokens,
			Stream:    false,
		})
	}
}

// TypedToolkitAdapter adapts TypedToolkit to implement tool.ToolProvider interface
type TypedToolkitAdapter struct {
	tt *tools.TypedToolkit
}

// GetSchemas returns tool schemas for the model
func (a *TypedToolkitAdapter) GetSchemas() []model.ToolDefinition {
	return a.tt.GetModelSchemas()
}

// Call executes a tool by name
func (a *TypedToolkitAdapter) Call(ctx context.Context, toolBlock *message.ToolUseBlock) (*tool.ToolResponse, error) {
	return a.tt.CallToolBlock(ctx, toolBlock)
}
