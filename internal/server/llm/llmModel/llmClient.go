package llmModel

import "github.com/openai/openai-go/v3"

type Response struct {
	Text             string
	ToolCalls        []ToolCall
	CompleteResponse *openai.ChatCompletion
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type Chat struct {
	Role       string
	Content    string
	ToolCallID string
}
