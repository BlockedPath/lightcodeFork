package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteFile(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	path, ok := args["path"].(string)
	if !ok {
		return ToolResponse{Content: "Error: path is required and must be a string"}, nil
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(ctx.WorkingDirectory, path)
	}
	cleanPath := filepath.Clean(path)
	allowedDir := filepath.Clean(ctx.WorkingDirectory)
	if !strings.HasPrefix(cleanPath, allowedDir+string(filepath.Separator)) && cleanPath != allowedDir {
		return ToolResponse{Content: fmt.Sprintf("Error: access denied: path %q is outside the allowed working directory %q", path, ctx.WorkingDirectory)}, nil
	}
	content, ok := args["content"].(string)
	if !ok {
		return ToolResponse{Content: "Error: content is required and must be a string"}, nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ToolResponse{Content: "Error: " + err.Error()}, err
	}

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error()}, err
	}
	return ToolResponse{Content: "File written successfully", CodeChanges: []string{"", content}}, nil
}

func init() {
	Prompt := `"Write content to a file, creating it if it doesn't exist", Do not guess paths before writing a new file`
	Register("write_file", ToolDef{
		Name:        "write_file",
		Description: Prompt,
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]string{
					"type":        "string",
					"description": "The path to the file to write",
				},
				"content": map[string]string{
					"type":        "string",
					"description": "The content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
	}, WriteFile)
}
