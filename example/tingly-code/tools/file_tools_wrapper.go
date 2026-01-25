package tools

import (
	"context"
	"fmt"
)

// ViewFileTool is a type-safe wrapper for ViewFile
type ViewFileTool struct {
	ft *FileTools
}

// NewViewFileTool creates a new ViewFileTool
func NewViewFileTool(ft *FileTools) *ViewFileTool {
	return &ViewFileTool{ft: ft}
}

func (t *ViewFileTool) Name() string {
	return "view_file"
}

func (t *ViewFileTool) Description() string {
	return "Read file contents with line numbers"
}

func (t *ViewFileTool) ParameterSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to read",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of lines to show (optional)",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "Starting line number (1-indexed, optional)",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ViewFileTool) Call(ctx context.Context, params any) (string, error) {
	var p ViewFileParams
	if err := MapToStruct(params.(map[string]any), &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	return t.ft.ViewFile(ctx, p)
}

// ReplaceFileTool is a type-safe wrapper for ReplaceFile
type ReplaceFileTool struct {
	ft *FileTools
}

// NewReplaceFileTool creates a new ReplaceFileTool
func NewReplaceFileTool(ft *FileTools) *ReplaceFileTool {
	return &ReplaceFileTool{ft: ft}
}

func (t *ReplaceFileTool) Name() string {
	return "replace_file"
}

func (t *ReplaceFileTool) Description() string {
	return "Create or overwrite a file with content"
}

func (t *ReplaceFileTool) ParameterSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to create or overwrite",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *ReplaceFileTool) Call(ctx context.Context, params any) (string, error) {
	var p ReplaceFileParams
	if err := MapToStruct(params.(map[string]any), &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	return t.ft.ReplaceFile(ctx, p)
}

// EditFileTool is a type-safe wrapper for EditFile
type EditFileTool struct {
	ft *FileTools
}

// NewEditFileTool creates a new EditFileTool
func NewEditFileTool(ft *FileTools) *EditFileTool {
	return &EditFileTool{ft: ft}
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "Replace a specific text in a file (requires exact match)"
}

func (t *EditFileTool) ParameterSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to edit",
			},
			"old_text": map[string]any{
				"type":        "string",
				"description": "Text to replace (must match exactly)",
			},
			"new_text": map[string]any{
				"type":        "string",
				"description": "Replacement text",
			},
		},
		"required": []string{"path", "old_text", "new_text"},
	}
}

func (t *EditFileTool) Call(ctx context.Context, params any) (string, error) {
	var p EditFileParams
	if err := MapToStruct(params.(map[string]any), &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	return t.ft.EditFile(ctx, p)
}

// GlobFilesTool is a type-safe wrapper for GlobFiles
type GlobFilesTool struct {
	ft *FileTools
}

// NewGlobFilesTool creates a new GlobFilesTool
func NewGlobFilesTool(ft *FileTools) *GlobFilesTool {
	return &GlobFilesTool{ft: ft}
}

func (t *GlobFilesTool) Name() string {
	return "glob_files"
}

func (t *GlobFilesTool) Description() string {
	return "Find files by name pattern (e.g., **/*.go, src/**/*.ts)"
}

func (t *GlobFilesTool) ParameterSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern to match files",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GlobFilesTool) Call(ctx context.Context, params any) (string, error) {
	var p GlobFilesParams
	if err := MapToStruct(params.(map[string]any), &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	return t.ft.GlobFiles(ctx, p)
}

// GrepFilesTool is a type-safe wrapper for GrepFiles
type GrepFilesTool struct {
	ft *FileTools
}

// NewGrepFilesTool creates a new GrepFilesTool
func NewGrepFilesTool(ft *FileTools) *GrepFilesTool {
	return &GrepFilesTool{ft: ft}
}

func (t *GrepFilesTool) Name() string {
	return "grep_files"
}

func (t *GrepFilesTool) Description() string {
	return "Search file contents using a text pattern"
}

func (t *GrepFilesTool) ParameterSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Text pattern to search for in files",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "Glob pattern to filter files (default: **/*.go)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepFilesTool) Call(ctx context.Context, params any) (string, error) {
	var p GrepFilesParams
	if err := MapToStruct(params.(map[string]any), &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	return t.ft.GrepFiles(ctx, p)
}

// ListDirectoryTool is a type-safe wrapper for ListDirectory
type ListDirectoryTool struct {
	ft *FileTools
}

// NewListDirectoryTool creates a new ListDirectoryTool
func NewListDirectoryTool(ft *FileTools) *ListDirectoryTool {
	return &ListDirectoryTool{ft: ft}
}

func (t *ListDirectoryTool) Name() string {
	return "list_directory"
}

func (t *ListDirectoryTool) Description() string {
	return "List files and directories in a path"
}

func (t *ListDirectoryTool) ParameterSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to list (default: current directory)",
			},
		},
		"required": []string{},
	}
}

func (t *ListDirectoryTool) Call(ctx context.Context, params any) (string, error) {
	var p ListDirectoryParams
	if err := MapToStruct(params.(map[string]any), &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	return t.ft.ListDirectory(ctx, p)
}
