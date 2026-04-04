package tools

import (
	"fmt"
	"os/exec"
)

func Glob(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return ToolResponse{Content: "Error: pattern is required and must be a string"}, nil
	}
	path, ok := args["path"].(string)
	if !ok {
		return ToolResponse{Content: "Error: path is required and must be a string"}, nil
	}
	resolved, err := ValidatePath(ctx, path)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error()}, nil
	}
	path = resolved
	cmd := exec.Command("find", path, "-name", fmt.Sprintf("%s", pattern))
	cmd.Dir = ctx.WorkingDirectory
	output, err := cmd.Output()
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error()}, err
	}
	return ToolResponse{Content: string(output)}, nil
}

func init() {
	Register("glob", ToolDef{
		Name:        "glob",
		Description: "Fast file pattern matching tool that works with any codebase size",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]string{
					"type":        "string",
					"description": "The pattern to match",
				},
				"path": map[string]string{
					"type":        "string",
					"description": "The path to search in",
				},
			},
			"required": []string{"pattern", "path"},
		},
	}, Glob)
}
