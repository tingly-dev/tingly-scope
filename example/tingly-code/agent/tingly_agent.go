package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"example/tingly-code/config"
	"example/tingly-code/tools"

	"github.com/tingly-dev/tingly-scope/pkg/agent"
	"github.com/tingly-dev/tingly-scope/pkg/formatter"
	"github.com/tingly-dev/tingly-scope/pkg/memory"
	"github.com/tingly-dev/tingly-scope/pkg/message"
	"github.com/tingly-dev/tingly-scope/pkg/model"
	"github.com/tingly-dev/tingly-scope/pkg/model/anthropic"
	"github.com/tingly-dev/tingly-scope/pkg/model/openai"
	"github.com/tingly-dev/tingly-scope/pkg/module"
	"github.com/tingly-dev/tingly-scope/pkg/session"
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

	// Get max iterations with default
	maxIterations := cfg.MaxIterations
	if maxIterations <= 0 {
		maxIterations = config.DefaultMaxIterations
	}

	// Get memory size with default
	memorySize := cfg.MemorySize
	if memorySize <= 0 {
		memorySize = config.DefaultMemorySize
	}

	// Create memory with state persistence support (History instead of SimpleMemory)
	mem := memory.NewHistory(memorySize)

	// Create ReAct agent with type-safe toolkit
	reactAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          cfg.Name,
		SystemPrompt:  systemPrompt,
		Model:         chatModel,
		Toolkit:       &TypedToolkitAdapter{tt: tt},
		Memory:        mem,
		MaxIterations: maxIterations,
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
	fileTools      *tools.FileTools
	bashTools      *tools.BashTools
	workDir        string
	toolsConfig    *config.ToolsConfig
	sessionManager *session.SessionManager
	sessionConfig  *config.SessionConfig
}

// NewTinglyAgent creates a new TinglyAgent
func NewTinglyAgent(cfg *config.AgentConfig, workDir string) (*TinglyAgent, error) {
	return NewTinglyAgentWithToolsConfig(cfg, nil, workDir)
}

// NewTinglyAgentWithToolsConfig creates a new TinglyAgent with tool filtering
func NewTinglyAgentWithToolsConfig(cfg *config.AgentConfig, toolsConfig *config.ToolsConfig, workDir string) (*TinglyAgent, error) {
	return NewTinglyAgentWithToolsConfigAndSession(cfg, toolsConfig, nil, workDir)
}

// NewTinglyAgentWithToolsConfigAndSession creates a new TinglyAgent with tool filtering and session config
func NewTinglyAgentWithToolsConfigAndSession(cfg *config.AgentConfig, toolsConfig *config.ToolsConfig, sessionConfig *config.SessionConfig, workDir string) (*TinglyAgent, error) {
	reactAgent, err := CreateTinglyAgent(cfg, toolsConfig, workDir)
	if err != nil {
		return nil, err
	}

	fileTools := tools.NewFileTools(workDir)
	bashSession := tools.GetGlobalBashSession()
	tools.ConfigureBash(cfg.Shell.InitCommands, cfg.Shell.VerboseInit)
	bashTools := tools.NewBashTools(bashSession)

	ta := &TinglyAgent{
		ReActAgent:     reactAgent,
		fileTools:      fileTools,
		bashTools:      bashTools,
		workDir:        workDir,
		toolsConfig:    toolsConfig,
		sessionConfig:  sessionConfig,
		sessionManager: nil,
	}

	// Initialize session manager if enabled
	if sessionConfig != nil && sessionConfig.Enabled {
		if err := ta.initSessionManager(sessionConfig); err != nil {
			return nil, fmt.Errorf("failed to initialize session manager: %w", err)
		}
	}

	return ta, nil
}

// NewTinglyAgentFromConfigFile creates a TinglyAgent from a config file
func NewTinglyAgentFromConfigFile(configPath, workDir string) (*TinglyAgent, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewTinglyAgentWithToolsConfigAndSession(&cfg.Agent, &cfg.Tools, &cfg.Session, workDir)
}

// NewTinglyAgentFromDefaultConfig creates a TinglyAgent from default config locations
func NewTinglyAgentFromDefaultConfig(workDir string) (*TinglyAgent, error) {
	cfg, err := config.LoadConfigFromDefaultLocations()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewTinglyAgentWithToolsConfigAndSession(&cfg.Agent, &cfg.Tools, &cfg.Session, workDir)
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

// initSessionManager initializes the session manager for state persistence
func (ta *TinglyAgent) initSessionManager(cfg *config.SessionConfig) error {
	// Determine save directory
	saveDir := cfg.SaveDir
	if saveDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		saveDir = filepath.Join(homeDir, ".tingly", "sessions")
	}

	// Create JSON session
	sess := session.NewJSONSession(saveDir)
	ta.sessionManager = session.NewSessionManager(sess)

	// Register ReActAgent as a state module (it implements the correct interface)
	ta.sessionManager.RegisterModule("agent", ta.ReActAgent)

	// Register memory as a state module
	if mem := ta.ReActAgent.GetMemory(); mem != nil {
		if stateMem, ok := mem.(module.StateModule); ok {
			ta.sessionManager.RegisterModule("memory", stateMem)
		}
	}

	return nil
}

// SaveSession saves the current agent state to a session file
func (ta *TinglyAgent) SaveSession(ctx context.Context, sessionID string) error {
	if ta.sessionManager == nil {
		return fmt.Errorf("session manager is not initialized. enable session in config to use this feature")
	}
	return ta.sessionManager.Save(ctx, sessionID)
}

// LoadSession loads agent state from a session file
func (ta *TinglyAgent) LoadSession(ctx context.Context, sessionID string, allowNotExist bool) error {
	if ta.sessionManager == nil {
		return fmt.Errorf("session manager is not initialized. enable session in config to use this feature")
	}
	return ta.sessionManager.Load(ctx, sessionID, allowNotExist)
}

// DeleteSession deletes a session file
func (ta *TinglyAgent) DeleteSession(ctx context.Context, sessionID string) error {
	if ta.sessionManager == nil {
		return fmt.Errorf("session manager is not initialized. enable session in config to use this feature")
	}
	return ta.sessionManager.Delete(ctx, sessionID)
}

// ListSessions returns all available session IDs
func (ta *TinglyAgent) ListSessions(ctx context.Context) ([]string, error) {
	if ta.sessionManager == nil {
		return nil, fmt.Errorf("session manager is not initialized. enable session in config to use this feature")
	}
	return ta.sessionManager.List(ctx)
}

// SessionExists checks if a session exists
func (ta *TinglyAgent) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	if ta.sessionManager == nil {
		return false, fmt.Errorf("session manager is not initialized. enable session in config to use this feature")
	}
	return ta.sessionManager.Exists(ctx, sessionID)
}

// IsSessionEnabled returns true if session persistence is enabled
func (ta *TinglyAgent) IsSessionEnabled() bool {
	return ta.sessionManager != nil
}

// GetDefaultSessionID returns the default session ID from config, or generates a timestamp-based one
func (ta *TinglyAgent) GetDefaultSessionID() string {
	if ta.sessionConfig != nil && ta.sessionConfig.SessionID != "" {
		return ta.sessionConfig.SessionID
	}
	// Generate timestamp-based session ID
	return fmt.Sprintf("session_%s", time.Now().Format("20060102_150405"))
}

// ShouldAutoSave returns true if auto-save is enabled
func (ta *TinglyAgent) ShouldAutoSave() bool {
	return ta.sessionConfig != nil && ta.sessionConfig.AutoSave
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
