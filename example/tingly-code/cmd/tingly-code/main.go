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
	"github.com/tingly-io/agentscope-go/pkg/agentscope/message"
	"github.com/tingly-io/agentscope-go/pkg/agentscope/types"
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
	case "diff":
		runDiff()
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
	fmt.Println("  diff         Create patch file from git changes")
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
