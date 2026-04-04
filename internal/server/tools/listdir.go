package tools

import (
	"os"
	"path/filepath"
)

func ListDir(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	path, ok := args["path"].(string)
	if !ok {
		return ToolResponse{Content: "Error: path is required and must be a string"}, nil
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(ctx.WorkingDirectory, path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error()}, err
	}

	var result string
	for _, e := range entries {
		if e.IsDir() {
			result += e.Name() + "/\n"
		} else {
			result += e.Name() + "\n"
		}
	}
	return ToolResponse{Content: result}, nil
}

func init() {
	Register("list_dir", ToolDef{
		Name:        "list_dir",
		Description: "List files and directories in a given path",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]string{
					"type":        "string",
					"description": "The directory path to list",
				},
			},
			"required": []string{"path"},
		},
	}, ListDir)
}
