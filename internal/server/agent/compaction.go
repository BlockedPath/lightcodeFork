package agent

import (
	"context"
	"fmt"
	"slices"

	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/server/llm/llmModel"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func predictTokenCount(messages []models.Message, i int) int64 {
	messages_before_current := messages[:i]
	slices.Reverse(messages_before_current)
	// messages_after_current := messages[i:]

	var last_assistant_tokens int64
	// var next_assistant_tokens int64

	for _, m := range messages_before_current {
		if models.DecodeMessageData(m.Data).Role == "assistant" {
			last_assistant_tokens = models.DecodeMessageData(m.Data).Usage.PromptTokens
			break
		}
	}

	// for _, m := range messages_after_current {
	// 	if models.DecodeMessageData(m.Data).Role == "assistant" {
	// 		next_assistant_tokens = models.DecodeMessageData(m.Data).Usage.PromptTokens
	// 		break
	// 	}
	// }

	return last_assistant_tokens + int64(len(models.DecodeMessageData(messages[i].Data).Content)/4)
}

func CompactMemory(chats []llmModel.Chat) (models.StoredMessageData, error) {
	summary, err := apiCall(chats)
	if err != nil {
		return models.StoredMessageData{}, err
	}
	compactedMemory := models.StoredMessageData{
		Role:        "user",
		Content:     "<memory>" + summary + "</memory>",
		CodeChanges: []string{},
		Usage:       &models.StoredUsage{PromptTokens: 0, CompletionTokens: 0, TotalTokens: 0},
		ToolCalls:   []models.StoredToolCall{},
	}
	return compactedMemory, nil
}

func apiCall(chats []llmModel.Chat) (string, error) {
	ctx := context.Background()
	cur_model, err := config.GetCurrentModel()
	if err != nil {
		return "", err
	}
	client := openai.NewClient(option.WithAPIKey(cur_model.ApiKey), option.WithBaseURL(cur_model.BaseUrl))

	var messages []openai.ChatCompletionMessageParamUnion
	// messages = append(messages, openai.SystemMessage(COMPACTION_PROMPT))

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
	messages = append(messages, openai.UserMessage(COMPACTION_PROMPT))
	m := cur_model.Model

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    m,
	})

	if err != nil {
		fmt.Println("Error", err)
		return "Ran into an error while calling the LLM", err
	}
	if len(resp.Choices) == 0 {
		return "Ran into an error while calling the LLM", err
	}
	fmt.Println("predicted compacted context", int(len(resp.Choices[0].Message.Content)/4))
	return resp.Choices[0].Message.Content, nil
}

var COMPACTION_PROMPT = `The messages above are a conversation to summarize. Create a structured context checkpoint summary that another LLM will use to continue the work.

Use this EXACT format:

## Goal
[What is the user trying to accomplish? Can be multiple items if the session covers different tasks.]

## Constraints & Preferences
- [Any constraints, preferences, or requirements mentioned by user]
- [Or "(none)" if none were mentioned]

## Progress
### Done
- [x] [Completed tasks/changes]

### In Progress
- [ ] [Current work]

### Blocked
- [Issues preventing progress, if any]

## Key Decisions
- **[Decision]**: [Brief rationale]

## Next Steps
1. [Ordered list of what should happen next]

## Critical Context
- [Any data, examples, or references needed to continue]
- [Or "(none)" if not applicable]

Keep each section concise. Preserve exact file paths, function names, and error messages.`

const UPDATE_SUMMARIZATION_PROMPT = `The messages above are NEW conversation messages to incorporate into the existing summary provided in <previous-summary> tags.

Update the existing structured summary with new information. RULES:
- PRESERVE all existing information from the previous summary
- ADD new progress, decisions, and context from the new messages
- UPDATE the Progress section: move items from "In Progress" to "Done" when completed
- UPDATE "Next Steps" based on what was accomplished
- PRESERVE exact file paths, function names, and error messages
- If something is no longer relevant, you may remove it

Use this EXACT format:

## Goal
[Preserve existing goals, add new ones if the task expanded]

## Constraints & Preferences
- [Preserve existing, add new ones discovered]

## Progress
### Done
- [x] [Include previously done items AND newly completed items]

### In Progress
- [ ] [Current work - update based on progress]

### Blocked
- [Current blockers - remove if resolved]

## Key Decisions
- **[Decision]**: [Brief rationale] (preserve all previous, add new)

## Next Steps
1. [Update based on current state]

## Critical Context
- [Preserve important context, add new if needed]

Keep each section concise. Preserve exact file paths, function names, and error messages.
`
