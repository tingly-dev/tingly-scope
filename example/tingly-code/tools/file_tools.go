package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar"
	"github.com/tingly-dev/tingly-scope/pkg/tool"
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

// Constraint returns the output constraint for file tools
// Implements the ConstrainedTool interface
func (ft *FileTools) Constraint() tool.OutputConstraint {
	// File tools can produce large output, especially:
	// - view_file: reading entire files
	// - grep_files: many matches
	// - glob_files: many file paths
	return tool.NewDefaultConstraint(10*1024, 2000, 100, 30) // 10KB, 2000 lines, 100 items, 30s timeout
}

// Tool description for view_file
const ToolDescViewFile = "Read file contents with line numbers"

// ViewFileParams holds the parameters for ViewFile
type ViewFileParams struct {
	Path   string `json:"path" required:"true" description:"Path to the file to read"`
	Limit  int    `json:"limit,omitempty" description:"Maximum number of lines to return (0 = all lines)"`
	Offset int    `json:"offset,omitempty" description:"Line number to start reading from (0-based)"`
}

// ViewFile reads file contents with line numbers (optimized for large files)
func (ft *FileTools) ViewFile(ctx context.Context, params ViewFileParams) (string, error) {
	var fullPath string
	if filepath.IsAbs(params.Path) {
		fullPath = params.Path
	} else {
		fullPath = filepath.Join(ft.GetWorkDir(), params.Path)
	}

	f, err := os.Open(fullPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to open file: %v", err), nil
	}
	defer f.Close()

	// Use bufio.Scanner for line-by-line reading (memory efficient)
	scanner := bufio.NewScanner(f)
	var result strings.Builder

	// Apply offset
	start := params.Offset
	if start < 0 {
		start = 0
	}

	lineNum := 0
	// Skip to offset
	for lineNum < start && scanner.Scan() {
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("Error: failed to read file: %v", err), nil
	}

	if lineNum < start && !scanner.Scan() {
		return "Error: offset beyond file length", nil
	}

	// Read lines with limit
	remaining := params.Limit
	if remaining <= 0 {
		remaining = -1 // No limit
	}

	for scanner.Scan() {
		if remaining == 0 {
			break
		}
		result.WriteString(fmt.Sprintf("%5d: %s\n", lineNum+1, scanner.Text()))
		lineNum++
		if remaining > 0 {
			remaining--
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Sprintf("Error: failed to read file: %v", err), nil
	}

	return result.String(), nil
}

// Tool description for replace_file
const ToolDescReplaceFile = "Create or overwrite a file with content"

// ReplaceFileParams holds the parameters for ReplaceFile
type ReplaceFileParams struct {
	Path    string `json:"path" required:"true" description:"Path to the file to create/overwrite"`
	Content string `json:"content" required:"true" description:"Content to write to the file"`
}

// ReplaceFile creates or overwrites a file with content
func (ft *FileTools) ReplaceFile(ctx context.Context, params ReplaceFileParams) (string, error) {
	var fullPath string
	if filepath.IsAbs(params.Path) {
		// Path is already absolute, use it directly
		fullPath = params.Path
	} else {
		// Relative path, join with workDir
		fullPath = filepath.Join(ft.GetWorkDir(), params.Path)
	}

	if err := os.WriteFile(fullPath, []byte(params.Content), 0644); err != nil {
		return fmt.Sprintf("Error: failed to write file: %v", err), nil
	}

	return fmt.Sprintf("File '%s' has been updated.", params.Path), nil
}

// Tool description for edit_file
const ToolDescEditFile = "Replace a specific text in a file (requires exact match)"

// EditFileParams holds the parameters for EditFile
type EditFileParams struct {
	Path    string `json:"path" required:"true" description:"Path to the file to edit"`
	OldText string `json:"old_text" required:"true" description:"Exact text to replace (must match exactly, consider context)"`
	NewText string `json:"new_text" required:"true" description:"New text to insert in place of old_text"`
}

// EditFile replaces a specific text in a file
func (ft *FileTools) EditFile(ctx context.Context, params EditFileParams) (string, error) {
	var fullPath string
	if filepath.IsAbs(params.Path) {
		// Path is already absolute, use it directly
		fullPath = params.Path
	} else {
		// Relative path, join with workDir
		fullPath = filepath.Join(ft.GetWorkDir(), params.Path)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to read file: %v", err), nil
	}

	content := string(data)
	if !strings.Contains(content, params.OldText) {
		return fmt.Sprintf("Error: old_text not found in file"), nil
	}

	newContent := strings.Replace(content, params.OldText, params.NewText, 1)

	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error: failed to write file: %v", err), nil
	}

	return fmt.Sprintf("File '%s' has been edited.", params.Path), nil
}

// Tool description for glob_files
const ToolDescGlobFiles = "Find files by name pattern (supports ** for recursive matching, e.g., **/*.go)"

// GlobFilesParams holds the parameters for GlobFiles
type GlobFilesParams struct {
	Pattern string `json:"pattern" required:"true" description:"Glob pattern to match files (supports **, e.g., **/*.go, *.txt)"`
}

// GlobFiles finds files by name pattern (supports ** recursive matching)
func (ft *FileTools) GlobFiles(ctx context.Context, params GlobFilesParams) (string, error) {
	// Use doublestar for ** support and better pattern matching
	// Join base directory with pattern for the full pattern
	pattern := filepath.Join(ft.GetWorkDir(), params.Pattern)
	matches, err := doublestar.Glob(pattern)
	if err != nil {
		return fmt.Sprintf("Error: failed to glob files: %v", err), nil
	}

	if len(matches) == 0 {
		return "No files found.", nil
	}

	return strings.Join(matches, "\n"), nil
}

// Tool description for grep_files
const ToolDescGrepFiles = "Search file contents using a text pattern (supports regex, concurrent search, and ripgrep fallback)"

// GrepFilesParams holds the parameters for GrepFiles
type GrepFilesParams struct {
	Pattern    string `json:"pattern" required:"true" description:"Text pattern or regex to search for in files"`
	Glob       string `json:"glob,omitempty" description:"Glob pattern to filter files (default: **/*.go)"`
	Regex      bool   `json:"regex,omitempty" description:"Treat pattern as regular expression"`
	IgnoreCase bool   `json:"ignore_case,omitempty" description:"Case-insensitive search"`
	UseRipgrep bool   `json:"use_ripgrep,omitempty" description:"Use ripgrep if available (default: true)"`
}

// grepWithRipgrep uses ripgrep (rg) command for fastest search
func (ft *FileTools) grepWithRipgrep(params GrepFilesParams) (string, error) {
	args := []string{}
	if params.Regex {
		args = append(args, "--regexp")
	} else {
		args = append(args, "--fixed-strings")
	}
	if params.IgnoreCase {
		args = append(args, "--ignore-case")
	}
	args = append(args, params.Pattern)

	globPattern := params.Glob
	if globPattern == "" {
		globPattern = "**/*.go"
	}
	args = append(args, "--glob", globPattern)

	// Add line number and no heading for cleaner output
	args = append(args, "--line-number", "--no-heading")

	cmd := exec.Command("rg", args...)
	cmd.Dir = ft.GetWorkDir()
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return "No matches found.", nil
	}
	return result, nil
}

// grepWithGo implements concurrent search using Go
func (ft *FileTools) grepWithGo(ctx context.Context, params GrepFilesParams) (string, error) {
	globPattern := params.Glob
	if globPattern == "" {
		globPattern = "**/*.go"
	}

	// Use doublestar for ** support
	pattern := filepath.Join(ft.GetWorkDir(), globPattern)
	files, err := doublestar.Glob(pattern)
	if err != nil {
		return fmt.Sprintf("Error: failed to glob files: %v", err), nil
	}

	// Compile regex if needed
	var regex *regexp.Regexp
	if params.Regex {
		patternRegex := params.Pattern
		if params.IgnoreCase {
			patternRegex = "(?i)" + patternRegex
		}
		regex, err = regexp.Compile(patternRegex)
		if err != nil {
			return fmt.Sprintf("Invalid regex: %v", err), nil
		}
	}

	var results []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrent goroutines to CPU count
	sem := make(chan struct{}, runtime.NumCPU())
	var hasMatches bool

	for _, file := range files {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fileHandle, err := os.Open(f)
			if err != nil {
				return
			}
			defer fileHandle.Close()

			scanner := bufio.NewScanner(fileHandle)
			lineNum := 0
			var fileMatches []string

			for scanner.Scan() {
				line := scanner.Text()
				match := false

				if regex != nil {
					match = regex.MatchString(line)
				} else {
					if params.IgnoreCase {
						match = strings.Contains(strings.ToLower(line), strings.ToLower(params.Pattern))
					} else {
						match = strings.Contains(line, params.Pattern)
					}
				}

				if match {
					fileMatches = append(fileMatches, fmt.Sprintf("%s:%d: %s", f, lineNum+1, line))
				}
				lineNum++
			}

			if len(fileMatches) > 0 {
				mu.Lock()
				results = append(results, fileMatches...)
				hasMatches = true
				mu.Unlock()
			}
		}(file)
	}
	wg.Wait()

	if !hasMatches {
		return "No matches found.", nil
	}

	return strings.Join(results, "\n"), nil
}

// GrepFiles searches file contents using a text pattern
// Tries ripgrep first (if available), falls back to concurrent Go implementation
func (ft *FileTools) GrepFiles(ctx context.Context, params GrepFilesParams) (string, error) {
	// Default: try ripgrep first if not explicitly disabled
	if params.UseRipgrep || (!params.UseRipgrep && params.UseRipgrep == false) {
		if _, err := exec.LookPath("rg"); err == nil {
			return ft.grepWithRipgrep(params)
		}
	}

	// Fallback to Go implementation
	return ft.grepWithGo(ctx, params)
}

// Tool description for list_directory
const ToolDescListDirectory = "List files and directories in a path"

// ListDirectoryParams holds the parameters for ListDirectory
type ListDirectoryParams struct {
	Path string `json:"path,omitempty" description:"Relative path to list (default: current directory)"`
}

// ListDirectory lists files and directories in a path
func (ft *FileTools) ListDirectory(ctx context.Context, params ListDirectoryParams) (string, error) {
	var targetPath string
	if params.Path != "" {
		if filepath.IsAbs(params.Path) {
			// Path is already absolute, use it directly
			targetPath = params.Path
		} else {
			// Relative path, join with workDir
			targetPath = filepath.Join(ft.GetWorkDir(), params.Path)
		}
	} else {
		targetPath = ft.GetWorkDir()
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return fmt.Sprintf("Error: failed to list directory: %v", err), nil
	}

	var dirs []string
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name()+"/")
		} else {
			files = append(files, entry.Name())
		}
	}

	var result strings.Builder
	if len(dirs) > 0 {
		result.WriteString("Directories:\n")
		for _, d := range dirs {
			result.WriteString("  " + d + "\n")
		}
	}
	if len(files) > 0 {
		result.WriteString("Files:\n")
		for _, f := range files {
			result.WriteString("  " + f + "\n")
		}
	}

	return result.String(), nil
}

func init() {
	// Register file tools in the global registry
	RegisterTool("view_file", ToolDescViewFile, "File Operations", true)
	RegisterTool("replace_file", ToolDescReplaceFile, "File Operations", true)
	RegisterTool("edit_file", ToolDescEditFile, "File Operations", true)
	RegisterTool("glob_files", ToolDescGlobFiles, "File Operations", true)
	RegisterTool("grep_files", ToolDescGrepFiles, "File Operations", true)
	RegisterTool("list_directory", ToolDescListDirectory, "File Operations", true)
}
