package tools

import (
	"context"
	"fmt"
)

// ReadNotebookParams holds the parameters for ReadNotebook
type ReadNotebookParams struct {
	NotebookPath string `json:"notebook_path" required:"true"`
}

// ReadNotebookTool is a type-safe wrapper for ReadNotebook
type ReadNotebookTool struct {
	nt *NotebookTools
}

func NewReadNotebookTool(nt *NotebookTools) *ReadNotebookTool {
	return &ReadNotebookTool{nt: nt}
}

func (t *ReadNotebookTool) Name() string {
	return "read_notebook"
}

func (t *ReadNotebookTool) Description() string {
	return "Read a Jupyter notebook (.ipynb file) and return all cells with their outputs"
}

func (t *ReadNotebookTool) ParameterSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"notebook_path": map[string]any{
				"type":        "string",
				"description": "Path to the Jupyter notebook file",
			},
		},
		"required": []string{"notebook_path"},
	}
}

func (t *ReadNotebookTool) Call(ctx context.Context, params any) (string, error) {
	var p ReadNotebookParams
	if err := MapToStruct(params.(map[string]any), &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}

	// Convert to old-style kwargs
	kwargs := map[string]any{
		"notebook_path": p.NotebookPath,
	}
	return t.nt.ReadNotebook(ctx, kwargs)
}

// NotebookEditCellParams holds the parameters for NotebookEditCell
type NotebookEditCellParams struct {
	NotebookPath string `json:"notebook_path" required:"true"`
	CellNumber   int    `json:"cell_number" required:"true"`
	NewSource    string `json:"new_source" required:"true"`
	EditMode     string `json:"edit_mode,omitempty"` // "replace", "insert", "delete"
	CellType     string `json:"cell_type,omitempty"`  // "code", "markdown" (for insert)
}

// NotebookEditCellTool is a type-safe wrapper for NotebookEditCell
type NotebookEditCellTool struct {
	nt *NotebookTools
}

func NewNotebookEditCellTool(nt *NotebookTools) *NotebookEditCellTool {
	return &NotebookEditCellTool{nt: nt}
}

func (t *NotebookEditCellTool) Name() string {
	return "notebook_edit_cell"
}

func (t *NotebookEditCellTool) Description() string {
	return "Edit a cell in a Jupyter notebook (supports replace, insert, delete modes)"
}

func (t *NotebookEditCellTool) ParameterSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"notebook_path": map[string]any{
				"type":        "string",
				"description": "Path to the Jupyter notebook file",
			},
			"cell_number": map[string]any{
				"type":        "integer",
				"description": "The index of the cell to edit (0-based)",
			},
			"new_source": map[string]any{
				"type":        "string",
				"description": "The new source for the cell",
			},
			"edit_mode": map[string]any{
				"type":        "string",
				"description": "The type of edit (replace, insert, delete)",
				"enum":         []string{"replace", "insert", "delete"},
			},
			"cell_type": map[string]any{
				"type":        "string",
				"description": "The type of cell (required for insert mode)",
				"enum":         []string{"code", "markdown"},
			},
		},
		"required": []string{"notebook_path", "cell_number", "new_source"},
	}
}

func (t *NotebookEditCellTool) Call(ctx context.Context, params any) (string, error) {
	var p NotebookEditCellParams
	if err := MapToStruct(params.(map[string]any), &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}

	// Convert to old-style kwargs
	kwargs := map[string]any{
		"notebook_path": p.NotebookPath,
		"cell_number":   p.CellNumber,
		"new_source":    p.NewSource,
		"edit_mode":     p.EditMode,
		"cell_type":     p.CellType,
	}
	return t.nt.NotebookEditCell(ctx, kwargs)
}
