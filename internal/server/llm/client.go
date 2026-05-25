package llm

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/server/llm/llmModel"
	"github.com/Kartik-2239/lightcode/internal/server/oauth"
	"github.com/Kartik-2239/lightcode/internal/server/prompt"
	"github.com/Kartik-2239/lightcode/internal/server/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func ApiCall(ctx context.Context, m config.ResModel, input string, chats []llmModel.Chat, originalMessages []models.Message, mode string, img_bytes [][]byte) (llmModel.Response, error) {
	trimmedMessages := originalMessages[len(originalMessages)-len(chats):]
	var toolCalls []llmModel.ToolCall
	// cur_model, err := config.GetCurrentModel()
	// if err != nil {
	// 	return Response{
	// 		Text:             "Ran into an error while getting the model",
	// 		ToolCalls:        []ToolCall{},
	// 		CompleteResponse: nil,
	// 	}, err
	// }

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
		if c.Content != "" {
			switch c.Role {
			case "user":
				messages = append(messages, openai.UserMessage(c.Content))
			case "assistant":
				messages = append(messages, openai.AssistantMessage(c.Content))
			case "tool":
				messages = append(messages, openai.ToolMessage(c.Content, c.ToolCallID))
			}
		}

	}
	parts := []openai.ChatCompletionContentPartUnionParam{}

	if input != "" {
		messages = append(messages, openai.UserMessage(input))
	}
	for _, img := range img_bytes {
		b64 := base64.StdEncoding.EncodeToString(img)
		// fmt.Println("======b64======")
		// fmt.Println(b64)
		// fmt.Println("======b64======")
		parts = append(parts, openai.ImageContentPart(
			openai.ChatCompletionContentPartImageImageURLParam{
				URL:    "data:image/png;base64," + b64,
				Detail: "auto",
			},
		))
	}

	if len(parts) > 0 {
		messages = append(messages, openai.UserMessage(parts))
	}
	// m := cur_model.Model
	var resp *openai.ChatCompletion
	var err error
	if strings.HasPrefix(m.BaseUrl, "https://") {
		client := openai.NewClient(option.WithAPIKey(m.ApiKey), option.WithBaseURL(m.BaseUrl))

		resp, err = client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Messages: messages,
			Tools:    tools.GetToolsForChat(),
			Model:    m.Model,
		})
	} else {
		resp, err = oauth.MakeOauthRequest(m.BaseUrl, m.Model, trimmedMessages, "WRITE CODE DON'T KEEP SAYING HI AGAIN AND AGAIN AFTER USER ASKS YOU TO DO SOMETHING.\n"+" Available skills: "+" "+prompt.AvailableSkills(), tools.GetAllTools())
	}

	if err != nil {
		fmt.Println("Error", err.Error())
		var apierr *openai.Error
		if errors.As(err, &apierr) {
			return llmModel.Response{
				Text:             apierr.Message,
				ToolCalls:        []llmModel.ToolCall{},
				CompleteResponse: nil,
			}, err
		}
		return llmModel.Response{Text: "Internal Error: " + err.Error()}, err

	}
	if len(resp.Choices) == 0 {
		return llmModel.Response{
			Text:             "Ran into an error while calling the LLM",
			ToolCalls:        []llmModel.ToolCall{},
			CompleteResponse: nil,
		}, err
	}

	for _, item := range resp.Choices[0].Message.ToolCalls {
		toolCalls = append(toolCalls, llmModel.ToolCall{
			ID:        item.ID,
			Name:      item.Function.Name,
			Arguments: item.Function.Arguments,
		})
	}
	fmt.Println("===========CACHED==============")
	fmt.Println("cached token", resp.Usage.PromptTokensDetails.CachedTokens)
	fmt.Println("usage: ", resp.Usage)
	fmt.Println("===========CACHED==============")

	return llmModel.Response{
		Text:             resp.Choices[0].Message.Content,
		ToolCalls:        toolCalls,
		CompleteResponse: resp,
	}, nil
}

func ExecuteToolCall(tc llmModel.ToolCall, workingDirectory string, sessionID string) (tools.ToolResponse, error) {
	args, err := tools.ParseArgs(tc.Arguments)
	if err != nil {
		return tools.ToolResponse{Content: "Error: " + err.Error()}, err
	}
	return tools.Execute(tc.Name, tools.ToolContext{WorkingDirectory: workingDirectory, SessionID: sessionID}, args)
}
