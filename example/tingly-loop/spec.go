package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Spec represents a parsed spec document
type Spec struct {
	Title      string
	Status     string
	Problem    string
	Solution   string
	Questions  []string
	Decisions  []string
	Tasks      *Tasks
	Discussion string
	SpecPath   string
}

// ParseSpec parses a spec markdown file
func ParseSpec(path string) (*Spec, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	spec := &Spec{
		SpecPath: path,
	}

	text := string(content)

	// Extract title (first # Spec: heading)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# Spec:") {
			spec.Title = strings.TrimSpace(strings.TrimPrefix(line, "# Spec:"))
			break
		}
	}

	// Extract status
	if strings.Contains(text, "[x] Ready for Implementation") {
		spec.Status = "Ready for Implementation"
	} else if strings.Contains(text, "[x] In Discussion") {
		spec.Status = "In Discussion"
	} else if strings.Contains(text, "[x] Draft") {
		spec.Status = "Draft"
	}

	// Extract Tasks section (JSON block)
	spec.Tasks, err = extractTasksFromSpec(text)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tasks: %w", err)
	}

	return spec, nil
}

// extractTasksFromSpec extracts the Tasks JSON from spec markdown
func extractTasksFromSpec(text string) (*Tasks, error) {
	// Find the Tasks section
	tasksIdx := strings.Index(text, "## Tasks")
	if tasksIdx == -1 {
		return nil, fmt.Errorf("no Tasks section found in spec")
	}

	// Find JSON block after Tasks section
	remaining := text[tasksIdx:]
	jsonStart := strings.Index(remaining, "```json")
	if jsonStart == -1 {
		return nil, fmt.Errorf("no JSON block found in Tasks section")
	}

	// Extract content between ```json and ```
	jsonContent := remaining[jsonStart+7:] // Skip "```json"
	jsonEnd := strings.Index(jsonContent, "```")
	if jsonEnd == -1 {
		return nil, fmt.Errorf("malformed JSON block in Tasks section")
	}

	jsonStr := strings.TrimSpace(jsonContent[:jsonEnd])
	var tasks Tasks
	if err := json.Unmarshal([]byte(jsonStr), &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks JSON: %w", err)
	}

	return &tasks, nil
}

// GenerateTasksFromSpec generates tasks.json from a spec file
func GenerateTasksFromSpec(specPath, tasksPath, workDir string) error {
	spec, err := ParseSpec(specPath)
	if err != nil {
		return fmt.Errorf("failed to parse spec: %w", err)
	}

	if spec.Tasks == nil {
		return fmt.Errorf("spec has no tasks defined")
	}

	// Check if tasks.json already exists
	if _, err := os.Stat(tasksPath); err == nil {
		// Archive existing tasks.json
		if err := archiveTasks(tasksPath); err != nil {
			return fmt.Errorf("failed to archive existing tasks: %w", err)
		}
	}

	// Ensure directory exists
	tasksDir := filepath.Dir(tasksPath)
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Write tasks.json
	data, err := json.MarshalIndent(spec.Tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	if err := os.WriteFile(tasksPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}

	fmt.Printf("Generated tasks: %s\n", tasksPath)
	return nil
}

// archiveTasks moves existing tasks.json to archive
func archiveTasks(tasksPath string) error {
	archiveDir := filepath.Join(filepath.Dir(tasksPath), "..", "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	archiveName := fmt.Sprintf("tasks-%s.json", timestamp)
	archivePath := filepath.Join(archiveDir, archiveName)

	// Read existing file
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to read existing tasks: %w", err)
	}

	// Write to archive
	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write archive: %w", err)
	}

	fmt.Printf("Archived existing tasks: %s\n", archivePath)
	return nil
}

// FindSpecFile finds the most recent spec file in docs/spec/
func FindSpecFile(workDir string) (string, error) {
	specDir := filepath.Join(workDir, "docs", "spec")
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return "", fmt.Errorf("failed to read spec directory: %w", err)
	}

	// Find most recent .md file
	var newest string
	var newestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newest = filepath.Join(specDir, entry.Name())
		}
	}

	if newest == "" {
		return "", fmt.Errorf("no spec files found in %s", specDir)
	}

	return newest, nil
}

// IsSpecReady checks if a spec is ready for implementation
func IsSpecReady(specPath string) (bool, error) {
	content, err := os.ReadFile(specPath)
	if err != nil {
		return false, fmt.Errorf("failed to read spec: %w", err)
	}
	return strings.Contains(string(content), "[x] Ready for Implementation"), nil
}
