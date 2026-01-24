package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileTools holds state for file operations
type FileTools struct {
	workDir string
}

// NewFileTools creates a new FileTools instance
func NewFileTools(workDir string) *FileTools {
	return &FileTools{
		workDir: workDir,
	}
}

// SetWorkDir sets the working directory for file operations
func (ft *FileTools) SetWorkDir(dir string) {
	ft.workDir = dir
}

// GetWorkDir returns the current working directory
func (ft *FileTools) GetWorkDir() string {
	if ft.workDir == "" {
		if dir, err := os.Getwd(); err == nil {
			ft.workDir = dir
		}
	}
	return ft.workDir
}

// ViewFileResponse is the response from ViewFile
type ViewFileResponse struct {
	Content string `json:"content"`
}

// ViewFile reads file contents with line numbers
func (ft *FileTools) ViewFile(ctx context.Context, kwargs map[string]any) (string, error) {
	path, ok := kwargs["path"].(string)
	if !ok {
		return "Error: path is required", nil
	}

	limit := 0
	if l, ok := kwargs["limit"].(float64); ok {
		limit = int(l)
	}

	offset := 0
	if o, ok := kwargs["offset"].(float64); ok {
		offset = int(o)
	}

	fullPath := filepath.Join(ft.GetWorkDir(), path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to read file: %v", err), nil
	}

	lines := strings.Split(string(data), "\n")

	// Apply offset and limit
	start := offset
	if start < 0 {
		start = 0
	}
	if start >= len(lines) {
		return "Error: offset beyond file length", nil
	}

	end := len(lines)
	if limit > 0 && start+limit < end {
		end = start + limit
	}

	var result strings.Builder
	for i := start; i < end; i++ {
		result.WriteString(fmt.Sprintf("%6d: %s\n", i+1, lines[i]))
	}

	if offset > 0 {
		return fmt.Sprintf("[%d lines above omitted]\n%s", offset, result.String()), nil
	}
	if end < len(lines) {
		result.WriteString(fmt.Sprintf("[%d lines below omitted]", len(lines)-end))
	}

	return result.String(), nil
}

// ReplaceFile creates or overwrites a file with content
func (ft *FileTools) ReplaceFile(ctx context.Context, kwargs map[string]any) (string, error) {
	path, ok := kwargs["path"].(string)
	if !ok {
		return "Error: path is required", nil
	}

	content, ok := kwargs["content"].(string)
	if !ok {
		return "Error: content is required", nil
	}

	fullPath := filepath.Join(ft.GetWorkDir(), path)

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error: failed to create directory: %v", err), nil
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error: failed to write file: %v", err), nil
	}

	return fmt.Sprintf("Successfully wrote file: %s", path), nil
}

// EditFile replaces a specific text in a file
func (ft *FileTools) EditFile(ctx context.Context, kwargs map[string]any) (string, error) {
	path, ok := kwargs["path"].(string)
	if !ok {
		return "Error: path is required", nil
	}

	oldText, ok := kwargs["old_text"].(string)
	if !ok {
		return "Error: old_text is required", nil
	}

	newText, ok := kwargs["new_text"].(string)
	if !ok {
		return "Error: new_text is required", nil
	}

	fullPath := filepath.Join(ft.GetWorkDir(), path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to read file: %v", err), nil
	}

	content := string(data)
	if !strings.Contains(content, oldText) {
		return "Error: old_text not found in file. The text must match exactly.", nil
	}

	newContent := strings.Replace(content, oldText, newText, 1)

	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error: failed to write file: %v", err), nil
	}

	return fmt.Sprintf("Successfully edited file: %s", path), nil
}

// GlobFiles finds files by name pattern
func (ft *FileTools) GlobFiles(ctx context.Context, kwargs map[string]any) (string, error) {
	pattern, ok := kwargs["pattern"].(string)
	if !ok {
		return "Error: pattern is required", nil
	}

	fullPattern := filepath.Join(ft.GetWorkDir(), pattern)

	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return fmt.Sprintf("Error: invalid pattern: %v", err), nil
	}

	// Make paths relative to workDir
	var result []string
	for _, match := range matches {
		relPath, err := filepath.Rel(ft.GetWorkDir(), match)
		if err == nil {
			result = append(result, relPath)
		}
	}

	return strings.Join(result, "\n"), nil
}

// GrepFiles searches file contents using regex pattern
func (ft *FileTools) GrepFiles(ctx context.Context, kwargs map[string]any) (string, error) {
	pattern, ok := kwargs["pattern"].(string)
	if !ok {
		return "Error: pattern is required", nil
	}

	glob := "**/*.go"
	if g, ok := kwargs["glob"].(string); ok {
		glob = g
	}

	// Get matching files
	fullPattern := filepath.Join(ft.GetWorkDir(), glob)
	matches, err := filepath.Glob(fullPattern)
	if err != nil && len(matches) == 0 {
		return "No matches found", nil
	}

	var result strings.Builder
	for _, match := range matches {
		relPath, err := filepath.Rel(ft.GetWorkDir(), match)
		if err != nil {
			continue
		}

		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.Contains(line, pattern) {
				result.WriteString(fmt.Sprintf("%s:%d: %s\n", relPath, i+1, line))
			}
		}
	}

	output := result.String()
	if output == "" {
		return "No matches found", nil
	}

	return output, nil
}

// ListDirectory lists files and directories
func (ft *FileTools) ListDirectory(ctx context.Context, kwargs map[string]any) (string, error) {
	path := "."
	if p, ok := kwargs["path"].(string); ok {
		path = p
	}

	fullPath := filepath.Join(ft.GetWorkDir(), path)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to read directory: %v", err), nil
	}

	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("DIR\t%s\n", entry.Name()))
		} else {
			result.WriteString(fmt.Sprintf("FILE\t%s\n", entry.Name()))
		}
	}

	return result.String(), nil
}
