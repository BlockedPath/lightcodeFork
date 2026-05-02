package tools

import (
	"os"
	"strings"
)

func Edit(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	path, ok := args["filePath"].(string)
	if !ok {
		return ToolResponse{Content: "Error: filePath is required and must be a string"}, nil
	}
	resolved, err := ValidatePath(ctx, path)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error()}, nil
	}
	path = resolved
	oldString, ok := args["oldString"].(string)
	if !ok {
		return ToolResponse{Content: "Error: oldString is required and must be a string"}, nil
	}
	newString, ok := args["newString"].(string)
	if !ok {
		return ToolResponse{Content: "Error: newString is required and must be a string"}, nil
	}
	copyNewString := newString

	var n int
	// JSON numbers are decoded as float64 into map[string]any
	if val, ok := args["replaceAll"].(float64); ok {
		if val == 1 {
			n = -1 // All occurrences
		} else {
			n = 1 // First occurrence
		}
	} else if val, ok := args["replaceAll"].(int); ok {
		if val == 1 {
			n = -1
		} else {
			n = 1
		}
	} else {
		return ToolResponse{Content: "Error: replaceAll is required and must be an integer"}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error()}, err
	}
	content := string(data)

	// Check if oldString exists
	if !strings.Contains(content, oldString) {
		return ToolResponse{Content: "Old string not found in file"}, nil
	}

	newContent := strings.Replace(content, oldString, newString, n)
	err = os.WriteFile(path, []byte(newContent), 0644)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error()}, err
	}

	return ToolResponse{Content: strings.Join([]string{"old_string: ", oldString, "========", "new_string: ", copyNewString}, "\n"), CodeChanges: []string{"---" + oldString, "+++" + copyNewString}}, nil
}

func init() {
	Prompt := `Perform exact string replacements in existing files
- Read the file before using edit tool and provide the exact old string and filepath
- DO NOT GUESS THE OLD STRING AND FILE PATH
`
	Register("edit", ToolDef{
		Name:        "edit",
		Description: Prompt,
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]any{
					"type":        "string",
					"description": "The path to the file to edit",
				},
				"oldString": map[string]any{
					"type":        "string",
					"description": "The string to find and replace",
				},
				"newString": map[string]any{
					"type":        "string",
					"description": "The replacement string",
				},
				"replaceAll": map[string]any{
					"type":        "integer",
					"description": "Replace all occurrences of the old string. 0 for first occurrence, 1 for all occurrences",
				},
			},
			"required": []string{"filePath", "oldString", "newString", "replaceAll"},
		},
	}, Edit)
}
