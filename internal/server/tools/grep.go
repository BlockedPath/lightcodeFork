package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Grep(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return ToolResponse{Content: "Error: pattern is required and must be a string"}, nil
	}

	path, ok := args["path"].(string)
	if !ok {
		return ToolResponse{Content: "Error: path is required and must be a string"}, nil
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(ctx.WorkingDirectory, path)
	}

	include, ok := args["include"].(string)
	if !ok {
		return ToolResponse{Content: "Error: include is required and must be a string"}, nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ToolResponse{Content: "Error: path does not exist: " + path}, nil
	}

	cmd := exec.Command("grep", "-r", "-l", "--include="+include, pattern, path)
	cmd.Dir = ctx.WorkingDirectory
	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				return ToolResponse{Content: "Error: No matches found"}, nil
			}
			return ToolResponse{Content: "Error: grep error: " + string(output)}, nil
		}
		return ToolResponse{Content: "Error: failed to execute grep: " + err.Error()}, err
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return ToolResponse{Content: "Error: No matches found"}, nil
	}

	return ToolResponse{Content: result}, nil
}

func init() {
	Register("grep", ToolDef{
		Name:        "grep",
		Description: "Search for a pattern in a file or directory",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]string{
					"type":        "string",
					"description": "The pattern to search for",
				},
				"path": map[string]string{
					"type":        "string",
					"description": "The path to search in",
				},
				"include": map[string]string{
					"type":        "string",
					"description": "The file extension to include",
					"default":     "*.go",
				},
			},
			"required": []string{"pattern", "path"},
		},
	}, Grep)
}
