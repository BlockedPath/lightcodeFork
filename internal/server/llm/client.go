package llm

import (
	"context"
	"fmt"

	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/prompt"
	"github.com/Kartik-2239/lightcode/internal/server/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

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

func ApiCall(ctx context.Context, input string, chats []Chat, mode string) (Response, error) {
	var toolCalls []ToolCall
	client := openai.NewClient(option.WithAPIKey(config.GetCurrentModel().ApiKey), option.WithBaseURL(config.GetCurrentModel().BaseUrl))

	var messages []openai.ChatCompletionMessageParamUnion
	if mode == "plan" {
		messages = append(messages, openai.SystemMessage(prompt.Plan_prompt()+prompt.AvailableSkills()))
	}
	if mode == "chat" {
		messages = append(messages, openai.SystemMessage(prompt.SystemPrompt()+" Available skills: "+" "+prompt.AvailableSkills()))
	}
	if mode == "assistant" {
		messages = append(messages, openai.SystemMessage(prompt.Assistant_prompt()+prompt.ExplorePrompt()))
	}

	for _, c := range chats {
		switch c.Role {
		case "user":
			messages = append(messages, openai.UserMessage(c.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(c.Content))
		case "tool":
			messages = append(messages, openai.ToolMessage(c.Content, c.ToolCallID))
		}
	}
	if input != "" {
		messages = append(messages, openai.UserMessage(input))
	}
	m := config.GetCurrentModel().Model

	// if mode == "plan" {
	// 	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
	// 		Messages: messages,
	// 		Tools:    tools.GetToolsForPlan(),
	// 		Model:    m,
	// 	})
	// }else{
	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		Tools:    tools.GetToolsForChat(),
		Model:    m,
	})
	// }

	if err != nil {
		fmt.Println("Error", err)
		return Response{
			Text:             "Ran into an error while calling the LLM",
			ToolCalls:        []ToolCall{},
			CompleteResponse: nil,
		}, err
	}
	if len(resp.Choices) == 0 {
		return Response{
			Text:             "Ran into an error while calling the LLM",
			ToolCalls:        []ToolCall{},
			CompleteResponse: nil,
		}, err
	}

	for _, item := range resp.Choices[0].Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:        item.ID,
			Name:      item.Function.Name,
			Arguments: item.Function.Arguments,
		})
	}

	return Response{
		Text:             resp.Choices[0].Message.Content,
		ToolCalls:        toolCalls,
		CompleteResponse: resp,
	}, nil
}

func ExecuteToolCall(tc ToolCall, workingDirectory string, sessionID string) (tools.ToolResponse, error) {
	args, err := tools.ParseArgs(tc.Arguments)
	if err != nil {
		return tools.ToolResponse{Content: "Error: " + err.Error()}, err
	}
	return tools.Execute(tc.Name, tools.ToolContext{WorkingDirectory: workingDirectory, SessionID: sessionID}, args)
}
