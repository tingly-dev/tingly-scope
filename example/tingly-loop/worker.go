package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worker defines the interface for an agent that executes tasks
type Worker interface {
	// Execute runs the agent with the given prompt and returns the output
	Execute(ctx context.Context, prompt string) (string, error)
	// Name returns the worker name
	Name() string
}

// SubprocessWorker calls an external agent binary (like tingly-code or claude CLI)
type SubprocessWorker struct {
	binaryPath string
	args       []string
	workDir    string
	env        []string
}

// NewSubprocessWorker creates a worker that calls an external binary
func NewSubprocessWorker(binaryPath string, args []string, workDir string) *SubprocessWorker {
	return &SubprocessWorker{
		binaryPath: binaryPath,
		args:       args,
		workDir:    workDir,
		env:        os.Environ(),
	}
}

// Execute runs the external agent with the prompt via stdin
func (w *SubprocessWorker) Execute(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, w.binaryPath, w.args...)
	cmd.Dir = w.workDir
	cmd.Env = w.env

	// Pass prompt via stdin
	cmd.Stdin = strings.NewReader(prompt)

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("worker failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// Name returns the worker name
func (w *SubprocessWorker) Name() string {
	return filepath.Base(w.binaryPath)
}

// ClaudeCLIWorker calls the claude CLI directly (like ralph does)
type ClaudeCLIWorker struct {
	workDir      string
	skipPerms    bool
	printMode    bool
	instructions string
}

// NewClaudeCLIWorker creates a worker that calls claude CLI
func NewClaudeCLIWorker(workDir string, instructions string) *ClaudeCLIWorker {
	return &ClaudeCLIWorker{
		workDir:      workDir,
		skipPerms:    true,
		printMode:    true,
		instructions: instructions,
	}
}

// Execute runs claude CLI with the instructions
func (w *ClaudeCLIWorker) Execute(ctx context.Context, prompt string) (string, error) {
	// Combine instructions with the iteration prompt
	fullPrompt := w.instructions + "\n\n" + prompt

	args := []string{}
	if w.skipPerms {
		args = append(args, "--dangerously-skip-permissions")
	}
	if w.printMode {
		args = append(args, "--print")
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = w.workDir
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(fullPrompt)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("claude failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// Name returns the worker name
func (w *ClaudeCLIWorker) Name() string {
	return "claude"
}

// TinglyCodeWorker calls tingly-code as a subprocess
type TinglyCodeWorker struct {
	binaryPath string
	workDir    string
	configPath string
}

// NewTinglyCodeWorker creates a worker that calls tingly-code
func NewTinglyCodeWorker(binaryPath string, workDir string, configPath string) *TinglyCodeWorker {
	return &TinglyCodeWorker{
		binaryPath: binaryPath,
		workDir:    workDir,
		configPath: configPath,
	}
}

// Execute runs tingly-code auto mode with the prompt
func (w *TinglyCodeWorker) Execute(ctx context.Context, prompt string) (string, error) {
	args := []string{"auto", prompt}
	if w.configPath != "" {
		args = append([]string{"--config", w.configPath}, args...)
	}

	cmd := exec.CommandContext(ctx, w.binaryPath, args...)
	cmd.Dir = w.workDir
	cmd.Env = os.Environ()

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("tingly-code failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// Name returns the worker name
func (w *TinglyCodeWorker) Name() string {
	return "tingly-code"
}

// CreateWorker creates a worker based on configuration
func CreateWorker(cfg *Config) (Worker, error) {
	switch cfg.WorkerType {
	case "claude":
		return NewClaudeCLIWorker(cfg.WorkDir, cfg.Instructions), nil
	case "tingly-code":
		binaryPath := cfg.WorkerBinary
		if binaryPath == "" {
			// Try to find tingly-code in PATH or common locations
			if path, err := exec.LookPath("tingly-code"); err == nil {
				binaryPath = path
			} else {
				// Try relative path
				candidates := []string{
					"../tingly-code/tingly-code",
					"./tingly-code",
				}
				for _, c := range candidates {
					absPath := filepath.Join(cfg.WorkDir, c)
					if _, err := os.Stat(absPath); err == nil {
						binaryPath = absPath
						break
					}
				}
			}
		}
		if binaryPath == "" {
			return nil, fmt.Errorf("tingly-code binary not found, specify with --worker-binary")
		}
		return NewTinglyCodeWorker(binaryPath, cfg.WorkDir, cfg.ConfigPath), nil
	case "subprocess":
		if cfg.WorkerBinary == "" {
			return nil, fmt.Errorf("--worker-binary is required for subprocess worker")
		}
		return NewSubprocessWorker(cfg.WorkerBinary, cfg.WorkerArgs, cfg.WorkDir), nil
	default:
		return nil, fmt.Errorf("unknown worker type: %s", cfg.WorkerType)
	}
}
