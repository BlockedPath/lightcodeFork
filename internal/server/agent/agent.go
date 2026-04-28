package agent

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Kartik-2239/lightcode/internal/server/db"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/server/llm"
	"gorm.io/gorm"
)

const MaxIterations = 25
const DEBUG = false

type Agent struct{}

func New() *Agent {
	return &Agent{}
}

func (a *Agent) Run(ctx context.Context, prompt string, session_id string, mode string) <-chan models.StoredMessageData {
	ch := make(chan models.StoredMessageData)
	// currentPrompt := prompt
	database, err := db.Connect()
	if err != nil {
		ch <- models.StoredMessageData{Role: "error", Content: "Ran into error: " + err.Error()}
		close(ch)
		return ch
	}
	var session models.Session
	result := database.Where("id = ?", session_id).First(&session)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newSession := models.Session{
				ID:        session_id,
				Title:     prompt,
				Directory: ".",
			}
			database.Create(&newSession)
		}
	}

	go func() {
		defer close(ch)

		for i := 0; i < MaxIterations; i++ {

			select {
			case <-ctx.Done():
				return
			default:
			}
			if DEBUG {
				fmt.Println("Iteration:", i)
			}
			var messages []models.Message
			database.Where("session_id = ?", session_id).Find(&messages)
			chats := make([]llm.Chat, 0, len(messages)+2) // +2 for agents.md and todo list
			for _, message := range messages {
				d := models.DecodeMessageData(message.Data)
				switch d.Role {
				case "tool_call":
					name, id := "tool", ""
					if len(d.ToolCalls) > 0 {
						name = d.ToolCalls[0].Name
						id = d.ToolCalls[0].ID
					}
					chats = append(chats, llm.Chat{
						Role:    "user",
						Content: fmt.Sprintf("Tool %q (call_id=%s) output:\n%s", name, id, d.Content),
					})
				case "assistant":
					content := d.Content
					chats = append(chats, llm.Chat{Role: "assistant", Content: content})
				default:
					chats = append(chats, llm.Chat{Role: d.Role, Content: d.Content})
				}
			}
			var session models.Session
			database.Where("id = ?", session_id).First(&session)
			cur_list := session.ToDoList
			chats = append(chats, llm.Chat{Role: "user", Content: cur_list})
			agents_md, err := ReadAgentsMd(session.Directory)
			if err != nil {
				slices.Reverse(chats)
				chats = append(chats, llm.Chat{Role: "user", Content: fmt.Sprintf("<agents_md>%s<agents_md>", agents_md)})
				slices.Reverse(chats)
			}

			resp, err := llm.ApiCall(ctx, "", chats, mode)
			if err != nil {
				errorMessage := models.StoredMessageData{Role: "error", Content: resp.Text, Usage: &models.StoredUsage{}}
				ch <- errorMessage
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
			if DEBUG {
				fmt.Println("================================================")
				fmt.Println("Tool calls:", resp.ToolCalls)
				fmt.Println("Number of tool calls:", len(resp.ToolCalls))
				fmt.Println("================================================")
			}
			if len(resp.ToolCalls) == 0 {
				select {
				case <-ctx.Done():
					return
				default:
				}
				assistantMessage := models.StoredMessageData{Role: "assistant", Content: resp.Text, Usage: &models.StoredUsage{PromptTokens: resp.CompleteResponse.Usage.PromptTokens, CompletionTokens: resp.CompleteResponse.Usage.CompletionTokens, TotalTokens: resp.CompleteResponse.Usage.TotalTokens}}
				newMessage := models.Message{
					SessionID: session_id,
					// ID:        fmt.Sprintf("%s-%d", session_id, len(messages)),
					Data: models.EncodeMessageData(assistantMessage),
				}
				if DEBUG {
					fmt.Println("Creating message:", newMessage)
				}
				if err := database.Create(&newMessage).Error; err != nil {
					if DEBUG {
						fmt.Println("Error creating message:", err)
					}
					return
				} else {
					if DEBUG {
						fmt.Println("Message created successfully! LAST!")
					}
				}
				ch <- assistantMessage
				return
			}

			var storedToolCalls []models.StoredToolCall
			for _, tc := range resp.ToolCalls {
				storedToolCalls = append(storedToolCalls, models.StoredToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
			}
			assistantMessage := models.StoredMessageData{Role: "assistant", Content: resp.Text, ToolCalls: storedToolCalls, Usage: &models.StoredUsage{PromptTokens: resp.CompleteResponse.Usage.PromptTokens, CompletionTokens: resp.CompleteResponse.Usage.CompletionTokens, TotalTokens: resp.CompleteResponse.Usage.TotalTokens}}
			assistantMsg := models.Message{
				SessionID: session_id,
				// ID:        fmt.Sprintf("%s-%d", session_id, len(messages)),
				Data: models.EncodeMessageData(assistantMessage),
			}
			ch <- assistantMessage
			if DEBUG {
				fmt.Println("Creating message:", assistantMsg)
			}
			if err := database.Create(&assistantMsg).Error; err != nil {
				if DEBUG {
					fmt.Println("Error creating message:", err)
				}
			} else {
				if DEBUG {
					fmt.Println("Message created successfully!")
				}
			}
			for _, tc := range resp.ToolCalls {
				if DEBUG {
					fmt.Println("Executing tool call:", tc.Name)
				}
				if tc.Name == "question" {
					ch <- models.StoredMessageData{Role: "question", Content: tc.Arguments}
					questionMsg := models.Message{
						SessionID: session_id,
						Data:      models.EncodeMessageData(models.StoredMessageData{Role: "question", Content: tc.Arguments, ToolCalls: []models.StoredToolCall{{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}}}),
					}
					database.Create(&questionMsg)
					return
				}
				result, err := llm.ExecuteToolCall(tc, session.Directory, session_id)
				if err != nil {
					if DEBUG {
						fmt.Println("Error executing tool call:", err)
					}
					ch <- models.StoredMessageData{Role: "error", Content: fmt.Sprintf("Tool '%s' failed: %v", tc.Name, err)}
					continue
				}
				ch <- models.StoredMessageData{Role: "tool_call", Content: result.Content, CodeChanges: result.CodeChanges, ToolCalls: []models.StoredToolCall{{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}}}
				toolMsg := models.Message{
					SessionID: session_id,
					Data:      models.EncodeMessageData(models.StoredMessageData{Role: "tool_call", Content: result.Content, CodeChanges: result.CodeChanges, ToolCalls: []models.StoredToolCall{{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}}}),
				}
				database.Create(&toolMsg)
				if DEBUG {
					fmt.Println("Result of tool call:", result)
				}
			}
		}
	}()
	return ch
}

// func (a *Agent) TextSkill(skill_name string) (string, error) {
// 	result, err := llm.ExecuteToolCall(llm.ToolCall{Name: "skill", Arguments: fmt.Sprintf("{\"skill_name\": \"%s\"}", skill_name)})
// 	if err != nil {
// 		return "", err
// 	}
// 	return result, nil
// }
