package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

// Config holds the tingly-loop configuration
type Config struct {
	// Paths
	PRDPath      string
	ProgressPath string
	WorkDir      string
	ConfigPath   string // Optional config file for worker

	// Loop settings
	MaxIterations int

	// Worker settings
	WorkerType   string   // "claude", "tingly-code", "subprocess"
	WorkerBinary string   // Path to worker binary (for subprocess/tingly-code)
	WorkerArgs   []string // Additional args for subprocess worker

	// Model settings (for reference, actual model config may be in worker config)
	ModelName   string
	BaseURL     string
	APIKey      string
	MaxTokens   int
	Temperature float64

	// Instructions (for claude worker)
	Instructions string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		PRDPath:       "prd.json",
		ProgressPath:  "progress.txt",
		MaxIterations: 10,
		WorkerType:    "claude", // Default to claude CLI like ralph
		ModelName:     "claude-sonnet-4-20250514",
		MaxTokens:     8000,
		Temperature:   0.3,
		Instructions:  defaultInstructions,
	}
}

// LoadConfigFromCLI creates a config from CLI flags
func LoadConfigFromCLI(c *cli.Context) (*Config, error) {
	cfg := DefaultConfig()

	// Get working directory
	workDir := c.String("workdir")
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}
	cfg.WorkDir = workDir

	// Override paths if provided
	if c.String("prd") != "" {
		cfg.PRDPath = c.String("prd")
	}
	if c.String("progress") != "" {
		cfg.ProgressPath = c.String("progress")
	}
	if c.String("config") != "" {
		cfg.ConfigPath = c.String("config")
	}

	// Make paths absolute if they're relative
	if !filepath.IsAbs(cfg.PRDPath) {
		cfg.PRDPath = filepath.Join(workDir, cfg.PRDPath)
	}
	if !filepath.IsAbs(cfg.ProgressPath) {
		cfg.ProgressPath = filepath.Join(workDir, cfg.ProgressPath)
	}

	// Override loop settings
	if c.Int("max-iterations") > 0 {
		cfg.MaxIterations = c.Int("max-iterations")
	}

	// Worker settings
	if c.String("worker") != "" {
		cfg.WorkerType = c.String("worker")
	}
	if c.String("worker-binary") != "" {
		cfg.WorkerBinary = c.String("worker-binary")
	}
	if c.StringSlice("worker-arg") != nil {
		cfg.WorkerArgs = c.StringSlice("worker-arg")
	}

	// Load custom instructions if provided
	if c.String("instructions") != "" {
		data, err := os.ReadFile(c.String("instructions"))
		if err != nil {
			return nil, fmt.Errorf("failed to read instructions file: %w", err)
		}
		cfg.Instructions = string(data)
	}

	return cfg, nil
}

// Validate checks if the config is valid
func (c *Config) Validate() error {
	// Validate worker type
	validWorkers := map[string]bool{
		"claude":      true,
		"tingly-code": true,
		"subprocess":  true,
	}
	if !validWorkers[c.WorkerType] {
		return fmt.Errorf("invalid worker type: %s (valid: claude, tingly-code, subprocess)", c.WorkerType)
	}

	// Subprocess worker requires binary path
	if c.WorkerType == "subprocess" && c.WorkerBinary == "" {
		return fmt.Errorf("--worker-binary is required for subprocess worker type")
	}

	return nil
}
