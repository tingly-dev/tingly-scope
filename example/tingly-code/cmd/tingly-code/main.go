package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"example/tingly-code/agent"
	"example/tingly-code/config"
	"example/tingly-code/swebench"
	"github.com/tingly-dev/tingly-scope/pkg/message"
	"github.com/tingly-dev/tingly-scope/pkg/types"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "chat":
		runChat()
	case "auto":
		runAuto()
	case "dual":
		runDual()
	case "diff":
		runDiff()
	case "swebench":
		runSwebench()
	case "init-config":
		runInitConfig()
	case "version", "--version", "-v":
		printVersion()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Tingly Code - AI Programming Assistant")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  tingly-code <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  chat         Interactive chat mode")
	fmt.Println("  auto <task>  Automated task resolution")
	fmt.Println("  dual <task>  Dual mode with planner and executor agents")
	fmt.Println("  diff         Create patch file from git changes")
	fmt.Println("  swebench     SWEbench integration")
	fmt.Println("  init-config  Create default config file")
	fmt.Println("  version      Show version information")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  TINGLY_CONFIG  Path to config file (default: ./tingly-config.toml or ~/.tingly/config.toml)")
}

func printVersion() {
	fmt.Println("Tingly Code v0.1.0")
	fmt.Println("AgentScope Go Framework")
}

// runChat runs the interactive chat mode
func runChat() {
	// Get working directory
	workDir, _ := os.Getwd()

	// Load config
	cfg, err := loadConfigOrUseDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		fmt.Fprintf(os.Stderr, "Using default configuration...\n")
		cfg = config.GetDefaultConfig()
	}

	// Create agent
	tinglyAgent, err := agent.NewTinglyAgent(&cfg.Agent, workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create agent: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🤖 Tingly Code - AI Programming Assistant")
	fmt.Println("Type /quit to exit, /help for commands")

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Print("\033[32m➜\033[0m ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle built-in commands
		if handleCommand(input, tinglyAgent) {
			continue
		}

		// Process user input
		msg := message.NewMsg(
			"user",
			[]message.ContentBlock{message.Text(input)},
			types.RoleUser,
		)

		resp, err := tinglyAgent.Reply(ctx, msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
			continue
		}

		// Print response
		printResponse(resp)

		// Check if task is done
		if tinglyAgent.IsJobDone(resp) {
			fmt.Println("\n✓ Task completed")
		}
	}
}

// runAuto runs the agent in automated mode
func runAuto() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: tingly-code auto <task>")
		os.Exit(1)
	}

	task := strings.Join(os.Args[2:], " ")

	// Get working directory
	workDir, _ := os.Getwd()

	// Load config
	cfg, err := loadConfigOrUseDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		cfg = config.GetDefaultConfig()
	}

	// Create agent
	tinglyAgent, err := agent.NewTinglyAgent(&cfg.Agent, workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create agent: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🤖 Running task: %s\n\n", task)

	ctx := context.Background()
	response, err := tinglyAgent.RunSinglePrompt(ctx, task)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(response)
}

// runDual runs the dual act agent mode with planner and executor
func runDual() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: tingly-code dual <task>")
		os.Exit(1)
	}

	task := strings.Join(os.Args[2:], " ")

	// Get working directory
	workDir, _ := os.Getwd()

	// Load config
	cfg, err := loadConfigOrUseDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		cfg = config.GetDefaultConfig()
	}

	// Create dual act agent
	dualAgent, err := agent.CreateDualTinglyAgent(cfg, workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create dual agent: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🤖 Dual Act Mode - Planner + Executor\n")
	fmt.Printf("📋 Task: %s\n\n", task)

	// Check if dual mode is actually enabled
	if !agent.IsDualModeEnabled(cfg) {
		fmt.Println("⚠️  Dual mode is not enabled in config.")
		fmt.Println("Enable it by setting [dual.enabled] = true in your config.")
		fmt.Println("\nFalling back to single agent mode...\n")
	}

	ctx := context.Background()
	userMsg := message.NewMsg(
		"user",
		task,
		types.RoleUser,
	)

	response, err := dualAgent.Reply(ctx, userMsg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print response
	printResponse(response)

	fmt.Println("\n✓ Dual Act execution completed")
}

// runDiff creates a patch file from git changes
func runDiff() {
	outputFile := "changes.patch"
	if len(os.Args) >= 3 {
		outputFile = os.Args[2]
	}

	// Load config
	cfg, err := loadConfigOrUseDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		cfg = config.GetDefaultConfig()
	}

	// Create diff agent
	diffAgent, err := agent.NewDiffAgent(&cfg.Agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create diff agent: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("📦 Creating patch file from git changes...")
	fmt.Printf("Output file: %s\n\n", outputFile)

	ctx := context.Background()
	if err := diffAgent.CreatePatch(ctx, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Patch file created: %s\n", outputFile)
}

// runInitConfig creates a default config file
func runInitConfig() {
	configPath := "tingly-config.toml"
	if len(os.Args) >= 3 {
		configPath = os.Args[2]
	}

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config file already exists: %s\n", configPath)
		fmt.Print("Overwrite? [y/N]: ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() || strings.ToLower(scanner.Text()) != "y" {
			fmt.Println("Cancelled")
			return
		}
	}

	// Create default config
	cfg := config.GetDefaultConfig()

	// Save config
	if err := config.SaveConfig(cfg, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Config file created: %s\n", configPath)
	fmt.Println("\nEdit the file to configure your model and API keys.")
}

// runSwebench runs the SWEbench integration
func runSwebench() {
	if len(os.Args) < 3 {
		runSwebenchHelp()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "download":
		runSwebenchDownload()
	case "list":
		runSwebenchList()
	case "run":
		runSwebenchRun()
	case "help", "-h", "--help":
		runSwebenchHelp()
	default:
		fmt.Printf("Unknown swebench command: %s\n\n", subcommand)
		runSwebenchHelp()
		os.Exit(1)
	}
}

func runSwebenchHelp() {
	fmt.Println("Tingly Code - SWEbench Integration")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  tingly-code swebench <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  download [lite|verified|full]  Download SWEbench dataset (default: lite)")
	fmt.Println("  list [lite|verified|full]       List available tasks")
	fmt.Println("  run <task-id> [options]        Run a single task")
	fmt.Println("  help                             Show this help")
	fmt.Println()
	fmt.Println("Run Options:")
	fmt.Println("  --agent-binary, -b <path>       Path to tingly-code binary (linux/amd64)")
	fmt.Println("  --config, -c <path>              Path to config file")
	fmt.Println("  --dataset, -d <name>             Dataset variant (lite|verified|full)")
	fmt.Println("  --keep-container, -k            Keep container after execution")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Download dataset")
	fmt.Println("  tingly-code swebench download lite")
	fmt.Println()
	fmt.Println("  # List tasks")
	fmt.Println("  tingly-code swebench list")
	fmt.Println()
	fmt.Println("  # Run task (auto-detects tingly-code-linux-amd64)")
	fmt.Println("  tingly-code swebench run django__django-11019")
	fmt.Println()
	fmt.Println("  # Build linux/amd64 binary first")
	fmt.Println("  GOOS=linux GOARCH=amd64 go build -o tingly-code-linux-amd64 ./cmd/tingly-code")
	fmt.Println()
	fmt.Println("  # Run with explicit binary and config")
	fmt.Println("  tingly-code swebench run django__django-11019 -b ./tingly-code-linux-amd64 -c ./tingly-config.toml")
}

// runSwebenchDownload downloads the SWEbench dataset
func runSwebenchDownload() {
	dataset := "lite"
	if len(os.Args) >= 4 {
		dataset = os.Args[3]
	}

	var dt swebench.DatasetType
	switch dataset {
	case "full":
		dt = swebench.DatasetTypeFull
	case "verified":
		dt = swebench.DatasetTypeVerified
	default:
		dt = swebench.DatasetTypeLite
	}

	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".tingly", "swebench")

	fetcher := swebench.NewFetcher(cacheDir)

	fmt.Printf("Downloading SWEbench %s dataset...\n", dataset)
	_, err := fetcher.Fetch(swebench.FetchOptions{
		Dataset: dt,
		Progress: func(msg string) {
			fmt.Println(msg)
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Download complete!")
}

// runSwebenchList lists available tasks
func runSwebenchList() {
	dataset := "lite"
	if len(os.Args) >= 4 {
		dataset = os.Args[3]
	}

	var dt swebench.DatasetType
	switch dataset {
	case "full":
		dt = swebench.DatasetTypeFull
	case "verified":
		dt = swebench.DatasetTypeVerified
	default:
		dt = swebench.DatasetTypeLite
	}

	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".tingly", "swebench")

	fetcher := swebench.NewFetcher(cacheDir)

	tasks, err := fetcher.ListTasks(dt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure to download the dataset first: tingly-code swebench download\n")
		os.Exit(1)
	}

	fmt.Printf("Found %d tasks in %s dataset:\n\n", len(tasks), dataset)
	for _, taskID := range tasks {
		fmt.Printf("  - %s\n", taskID)
	}
}

// runSwebenchRun runs a single SWEbench task
func runSwebenchRun() {
	if len(os.Args) < 4 {
		printSwebenchRunUsage()
		os.Exit(1)
	}

	taskID := os.Args[3]

	// Parse optional flags
	var dataset, agentBinary, configPath string
	var keepContainer bool

	args := os.Args[4:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agent-binary", "-b":
			if i+1 < len(args) {
				agentBinary = args[i+1]
				i++
			}
		case "--config", "-c":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		case "--dataset", "-d":
			if i+1 < len(args) {
				dataset = args[i+1]
				i++
			}
		case "--keep-container", "-k":
			keepContainer = true
		default:
			if args[i] == "lite" || args[i] == "verified" || args[i] == "full" {
				dataset = args[i]
			}
		}
	}

	if dataset == "" {
		dataset = "lite"
	}

	var dt swebench.DatasetType
	switch dataset {
	case "full":
		dt = swebench.DatasetTypeFull
	case "verified":
		dt = swebench.DatasetTypeVerified
	default:
		dt = swebench.DatasetTypeLite
	}

	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".tingly", "swebench")

	// Fetch task
	fetcher := swebench.NewFetcher(cacheDir)
	task, err := fetcher.GetTask(taskID, dt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Running task: %s\n", taskID)
	fmt.Printf("Repository: %s\n", task.Repo)
	fmt.Printf("Base commit: %s\n", task.BaseCommit)
	fmt.Printf("Image: swebench/sweb.eval.x86_64.%s:latest\n\n", strings.ReplaceAll(taskID, "__", "_1776_"))

	// Auto-detect agent binary if not specified
	if agentBinary == "" {
		candidates := []string{
			"./tingly-code-linux-amd64",
			"./cmd/tingly-code/tingly-code-linux-amd64",
			"tingly-code-linux-amd64",
		}
		for _, cand := range candidates {
			if _, err := os.Stat(cand); err == nil {
				agentBinary = cand
				break
			}
		}
		if agentBinary == "" {
			fmt.Fprintf(os.Stderr, "Warning: tingly-code-linux-amd64 binary not found.\n")
			fmt.Fprintf(os.Stderr, "Build with: GOOS=linux GOARCH=amd64 go build -o tingly-code-linux-amd64 ./cmd/tingly-code\n")
			fmt.Fprintf(os.Stderr, "Or specify with --agent-binary flag\n")
		}
	}

	if agentBinary != "" {
		fmt.Printf("Agent binary: %s\n", agentBinary)
	}
	if configPath != "" {
		fmt.Printf("Config: %s\n", configPath)
	}
	fmt.Println()

	// Create container manager
	cfg := &swebench.Config{
		CacheDir:      cacheDir,
		WorkDir:       filepath.Join(homeDir, ".tingly", "swebench", "work"),
		KeepContainer: keepContainer,
	}

	cm, err := swebench.NewContainerManager(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create container manager: %v\n", err)
		os.Exit(1)
	}
	defer cm.Close()

	ctx := context.Background()

	// Run in container
	result, err := cm.RunTaskInContainer(ctx, swebench.ContainerRunOptions{
		Task:          task,
		AgentBinary:   agentBinary,
		ConfigPath:    configPath,
		KeepContainer: keepContainer,
		OutputWriter:  os.Stdout,
		Progress: func(msg string) {
			fmt.Printf("[Progress] %s\n", msg)
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Task completed\n")
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Passed: %v\n", result.Passed)
	fmt.Printf("Duration: %v\n", result.Duration)
}

func printSwebenchRunUsage() {
	fmt.Println("Usage: tingly-code swebench run <task-id> [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --agent-binary, -b <path>   Path to tingly-code binary (linux/amd64)")
	fmt.Println("                               Default: ./tingly-code-linux-amd64")
	fmt.Println("  --config, -c <path>         Path to config file")
	fmt.Println("  --dataset, -d <name>        Dataset variant (lite|verified|full)")
	fmt.Println("  --keep-container, -k       Keep container after execution")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ./tingly-code swebench run django__django-11019")
	fmt.Println("  ./tingly-code swebench run django__django-11019 -b ./tingly-code-linux-amd64 -c ./tingly-config.toml")
}

// loadConfigOrUseDefault tries to load config from default locations
func loadConfigOrUseDefault() (*config.Config, error) {
	// Check environment variable
	if envPath := os.Getenv("TINGLY_CONFIG"); envPath != "" {
		return config.LoadConfig(envPath)
	}

	// Check current directory
	if _, err := os.Stat("tingly-config.toml"); err == nil {
		return config.LoadConfig("tingly-config.toml")
	}

	// Check home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".tingly", "config.toml")
		if _, err := os.Stat(configPath); err == nil {
			return config.LoadConfig(configPath)
		}
	}

	return nil, fmt.Errorf("no config file found")
}

// handleCommand handles built-in commands
func handleCommand(input string, ag *agent.TinglyAgent) bool {
	switch input {
	case "/quit", "/exit", "/q":
		fmt.Println("👋 Goodbye!")
		os.Exit(0)
		return true
	case "/help", "/h", "/?":
		printHelp()
		return true
	case "/clear", "/c":
		fmt.Print("\033[2J\033[H")
		return true
	default:
		return false
	}
}

func printHelp() {
	fmt.Println("\nCommands:")
	fmt.Println("  /quit, /exit, /q    - Exit")
	fmt.Println("  /help, /h, /?       - Show this help")
	fmt.Println("  /clear, /c          - Clear screen")
	fmt.Println()
	fmt.Println("Just type your question or task to interact with the agent!")
}

// printResponse prints the agent's response
func printResponse(msg *message.Msg) {
	blocks, ok := msg.Content.([]message.ContentBlock)
	if !ok {
		return
	}

	for _, block := range blocks {
		switch b := block.(type) {
		case *message.TextBlock:
			fmt.Print(b.Text)
		case *message.ToolUseBlock:
			fmt.Printf("\033[33m[Tool: %s]\033[0m\n", b.Name)
		case *message.ToolResultBlock:
			// Format output blocks
			var output strings.Builder
			for _, ob := range b.Output {
				if tb, ok := ob.(*message.TextBlock); ok {
					output.WriteString(tb.Text)
				}
			}
			fmt.Printf("\033[36m→ %s\033[0m\n", truncateString(output.String(), 200))
		}
	}
	fmt.Println()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
