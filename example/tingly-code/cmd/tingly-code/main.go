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
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "tingly-code",
		Version: "v0.1.0",
		Usage:   "AI Programming Assistant",
		Commands: []*cli.Command{
			chatCommand,
			autoCommand,
			dualCommand,
			diffCommand,
			swebenchCommand,
			initConfigCommand,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var chatCommand = &cli.Command{
	Name:  "chat",
	Usage: "Interactive chat mode",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"TINGLY_CONFIG"},
		},
	},
	Action: func(c *cli.Context) error {
		workDir, _ := os.Getwd()
		cfg, err := loadConfig(c.String("config"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			fmt.Fprintf(os.Stderr, "Using default configuration...\n")
			cfg = config.GetDefaultConfig()
		}

		tinglyAgent, err := agent.NewTinglyAgent(&cfg.Agent, workDir)
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
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

			if handleCommand(input, tinglyAgent) {
				continue
			}

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

			printResponse(resp)

			if tinglyAgent.IsJobDone(resp) {
				fmt.Println("\n✓ Task completed")
			}
		}
		return nil
	},
}

var autoCommand = &cli.Command{
	Name:  "auto",
	Usage: "Automated task resolution",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"TINGLY_CONFIG"},
		},
	},
	ArgsUsage: "<task>",
	Action: func(c *cli.Context) error {
		if c.Args().Len() < 1 {
			return fmt.Errorf("usage: tingly-code auto <task>")
		}

		task := c.Args().First()
		workDir, _ := os.Getwd()

		cfg, err := loadConfig(c.String("config"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			cfg = config.GetDefaultConfig()
		}

		tinglyAgent, err := agent.NewTinglyAgent(&cfg.Agent, workDir)
		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		fmt.Printf("🤖 Running task: %s\n\n", task)

		ctx := context.Background()
		response, err := tinglyAgent.RunSinglePrompt(ctx, task)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}

		fmt.Println(response)
		return nil
	},
}

var dualCommand = &cli.Command{
	Name:  "dual",
	Usage: "Dual mode with planner and executor agents",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"TINGLY_CONFIG"},
		},
	},
	ArgsUsage: "<task>",
	Action: func(c *cli.Context) error {
		if c.Args().Len() < 1 {
			return fmt.Errorf("usage: tingly-code dual <task>")
		}

		task := c.Args().First()
		workDir, _ := os.Getwd()

		cfg, err := loadConfig(c.String("config"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			cfg = config.GetDefaultConfig()
		}

		dualAgent, err := agent.CreateDualTinglyAgent(cfg, workDir)
		if err != nil {
			return fmt.Errorf("failed to create dual agent: %w", err)
		}

		fmt.Printf("🤖 Dual Act Mode - Planner + Executor\n")
		fmt.Printf("📋 Task: %s\n\n", task)

		if !agent.IsDualModeEnabled(cfg) {
			fmt.Println("⚠️  Dual mode is not enabled in config.")
			fmt.Println("Enable it by setting [dual.enabled] = true in your config.")
			fmt.Println("\nFalling back to single agent mode...\n")
		}

		ctx := context.Background()
		userMsg := message.NewMsg("user", task, types.RoleUser)

		response, err := dualAgent.Reply(ctx, userMsg)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}

		printResponse(response)
		fmt.Println("\n✓ Dual Act execution completed")
		return nil
	},
}

var diffCommand = &cli.Command{
	Name:  "diff",
	Usage: "Create patch file from git changes",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"TINGLY_CONFIG"},
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output patch file path",
			Value:   "changes.patch",
		},
	},
	Action: func(c *cli.Context) error {
		outputFile := c.String("output")

		cfg, err := loadConfig(c.String("config"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			cfg = config.GetDefaultConfig()
		}

		diffAgent, err := agent.NewDiffAgent(&cfg.Agent)
		if err != nil {
			return fmt.Errorf("failed to create diff agent: %w", err)
		}

		fmt.Println("📦 Creating patch file from git changes...")
		fmt.Printf("Output file: %s\n\n", outputFile)

		ctx := context.Background()
		if err := diffAgent.CreatePatch(ctx, outputFile); err != nil {
			return fmt.Errorf("error: %w", err)
		}

		fmt.Printf("\n✓ Patch file created: %s\n", outputFile)
		return nil
	},
}

var swebenchCommand = &cli.Command{
	Name:  "swebench",
	Usage: "SWEbench integration",
	Subcommands: []*cli.Command{
		swebenchDownloadCommand,
		swebenchListCommand,
		swebenchRunCommand,
	},
}

var swebenchDownloadCommand = &cli.Command{
	Name:  "download",
	Usage: "Download SWEbench dataset",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "dataset",
			Aliases: []string{"d"},
			Usage:   "Dataset variant (lite|verified|full)",
			Value:   "lite",
		},
	},
	Action: func(c *cli.Context) error {
		dataset := c.String("dataset")
		// Also check positional argument if flag is not provided
		if c.Args().First() != "" {
			dataset = c.Args().First()
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
			return fmt.Errorf("error: %w", err)
		}

		fmt.Println("\n✓ Download complete!")
		return nil
	},
}

var swebenchListCommand = &cli.Command{
	Name:  "list",
	Usage: "List available SWEbench tasks",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "dataset",
			Aliases: []string{"d"},
			Usage:   "Dataset variant (lite|verified|full)",
			Value:   "lite",
		},
	},
	Action: func(c *cli.Context) error {
		dataset := c.String("dataset")
		// Also check positional argument if flag is not provided
		if c.Args().First() != "" {
			dataset = c.Args().First()
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
			return fmt.Errorf("error: %w\nMake sure to download the dataset first: tingly-code swebench download", err)
		}

		fmt.Printf("Found %d tasks in %s dataset:\n\n", len(tasks), dataset)
		for _, taskID := range tasks {
			fmt.Printf("  - %s\n", taskID)
		}
		return nil
	},
}

var swebenchRunCommand = &cli.Command{
	Name:  "run",
	Usage: "Run a single SWEbench task",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
			EnvVars: []string{"TINGLY_CONFIG"},
		},
		&cli.StringFlag{
			Name:    "agent-binary",
			Aliases: []string{"b"},
			Usage:   "Path to tingly-code binary (linux/amd64)",
		},
		&cli.StringFlag{
			Name:    "dataset",
			Aliases: []string{"d"},
			Usage:   "Dataset variant (lite|verified|full)",
			Value:   "lite",
		},
		&cli.BoolFlag{
			Name:    "keep-container",
			Aliases: []string{"k"},
			Usage:   "Keep container after execution",
		},
	},
	ArgsUsage: "<task-id>",
	Action: func(c *cli.Context) error {
		if c.Args().Len() < 1 {
			return fmt.Errorf("usage: tingly-code swebench run <task-id>")
		}

		taskID := c.Args().First()
		configPath := c.String("config")
		agentBinary := c.String("agent-binary")
		dataset := c.String("dataset")
		keepContainer := c.Bool("keep-container")

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
		task, err := fetcher.GetTask(taskID, dt)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}

		fmt.Printf("Running task: %s\n", taskID)
		fmt.Printf("Repository: %s\n", task.Repo)
		fmt.Printf("Base commit: %s\n", task.BaseCommit)
		fmt.Printf("Image: swebench/sweb.eval.x86_64.%s:latest\n\n", strings.ReplaceAll(taskID, "__", "_1776_"))

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

		cfg := &swebench.Config{
			CacheDir:      cacheDir,
			WorkDir:       filepath.Join(homeDir, ".tingly", "swebench", "work"),
			KeepContainer: keepContainer,
		}

		cm, err := swebench.NewContainerManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to create container manager: %w", err)
		}
		defer cm.Close()

		ctx := context.Background()

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
			return fmt.Errorf("error: %w", err)
		}

		fmt.Printf("\n✓ Task completed\n")
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Passed: %v\n", result.Passed)
		fmt.Printf("Duration: %v\n", result.Duration)
		return nil
	},
}

var initConfigCommand = &cli.Command{
	Name:  "init-config",
	Usage: "Create default config file",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output config file path",
			Value:   "tingly-config.toml",
		},
	},
	Action: func(c *cli.Context) error {
		configPath := c.String("output")

		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config file already exists: %s\n", configPath)
			fmt.Print("Overwrite? [y/N]: ")

			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() || strings.ToLower(scanner.Text()) != "y" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		cfg := config.GetDefaultConfig()

		if err := config.SaveConfig(cfg, configPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Config file created: %s\n", configPath)
		fmt.Println("\nEdit the file to configure your model and API keys.")
		return nil
	},
}

func loadConfig(explicitPath string) (*config.Config, error) {
	if explicitPath != "" {
		return config.LoadConfig(explicitPath)
	}

	if _, err := os.Stat("tingly-config.toml"); err == nil {
		return config.LoadConfig("tingly-config.toml")
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".tingly", "config.toml")
		if _, err := os.Stat(configPath); err == nil {
			return config.LoadConfig(configPath)
		}
	}

	return nil, fmt.Errorf("no config file found")
}

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
