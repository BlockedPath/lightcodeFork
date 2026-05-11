package views

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/tui/client"
)

func exportCurrentSessionMarkdown(m *model, session models.Session) (string, error) {
	if session.ID == "" {
		return "", fmt.Errorf("no active session selected")
	}

	exportedAt := time.Now()
	exportSession, err := loadSessionForExport(session)
	if err != nil {
		m.isError = true
		m.errorMessage = "Failed to export session"
	}
	sessionData, err := client.GetSessionData(session.ID)
	if err != nil {
		m.isError = true
		m.errorMessage = "Failed to export session"
	}
	messages := sessionData
	sort.SliceStable(messages, func(i, j int) bool {
		if messages[i].TimeCreated == messages[j].TimeCreated {
			return messages[i].ID < messages[j].ID
		}
		return messages[i].TimeCreated < messages[j].TimeCreated
	})

	exportDir := filepath.Join(config.Dir(), "exports")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("session-%s.md", exportedAt.Format("2006-01-02-15-04-05"))
	path := filepath.Join(exportDir, filename)
	content := buildSessionExportMarkdown(exportSession, messages, exportedAt)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}

	return path, nil
}

func loadSessionForExport(session models.Session) (models.Session, error) {
	sessions, err := client.ListSession()
	if err != nil {
		return models.Session{}, err
	}
	for _, item := range sessions {
		if item.ID == session.ID {
			return item, nil
		}
	}
	return session, nil
}

func buildSessionExportMarkdown(session models.Session, messages []models.Message, exportedAt time.Time) string {
	title := strings.TrimSpace(session.Title)
	if title == "" {
		title = "Untitled Session"
	}

	var sb strings.Builder
	sb.WriteString("# Session Export\n\n")
	sb.WriteString(fmt.Sprintf("- Title: %s\n", title))
	sb.WriteString(fmt.Sprintf("- Session ID: %s\n", session.ID))
	sb.WriteString(fmt.Sprintf("- Session Created: %s\n", formatExportTimestamp(session.TimeCreated)))
	sb.WriteString(fmt.Sprintf("- Session Updated: %s\n", formatExportTimestamp(session.TimeUpdated)))
	sb.WriteString(fmt.Sprintf("- Exported At: %s\n\n", exportedAt.Local().Format("2006-01-02 15:04:05 MST")))
	sb.WriteString("## Messages\n\n")

	if len(messages) == 0 {
		sb.WriteString("_No messages in this session._\n")
		return sb.String()
	}

	for _, msg := range messages {
		data := models.DecodeMessageData(msg.Data)
		if shouldSkipExportMessage(data) {
			continue
		}

		label := exportRoleLabel(data.Role)
		sb.WriteString(fmt.Sprintf("### %s (%s)\n\n", label, formatExportTimestamp(msg.TimeCreated)))

		content := strings.TrimSpace(data.Content)
		if content == "" {
			sb.WriteString("_No content._\n\n")
		} else {
			sb.WriteString(content)
			sb.WriteString("\n\n")
		}

		if len(data.ToolCalls) > 0 {
			sb.WriteString("#### Tool Calls\n\n")
			for _, tc := range data.ToolCalls {
				sb.WriteString(fmt.Sprintf("- `%s`\n", tc.Name))
				if strings.TrimSpace(tc.Arguments) != "" {
					sb.WriteString("```json\n")
					sb.WriteString(tc.Arguments)
					sb.WriteString("\n```\n")
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func shouldSkipExportMessage(data models.StoredMessageData) bool {
	if data.Role == "" || data.Role == "todo_status" {
		return true
	}
	return strings.HasPrefix(data.Content, "<memory>") && strings.HasSuffix(data.Content, "</memory>")
}

func exportRoleLabel(role string) string {
	switch role {
	case "user":
		return "User"
	case "assistant":
		return "Assistant"
	case "tool_call":
		return "Tool Call"
	case "question":
		return "Question"
	default:
		return strings.Title(strings.ReplaceAll(role, "_", " "))
	}
}

func formatExportTimestamp(ts int64) string {
	if ts <= 0 {
		return "unknown"
	}

	if ts > 1_000_000_000_000 {
		return time.Unix(0, ts).Local().Format("2006-01-02 15:04:05 MST")
	}
	return time.Unix(ts, 0).Local().Format("2006-01-02 15:04:05 MST")
}

func appendCommandStatusMessage(m *model, content string) {
	m.messages = append(m.messages, models.Message{
		SessionID: m.currentSession.ID,
		Data:      models.EncodeMessageData(models.StoredMessageData{Role: "assistant", Content: content}),
	})
	m.viewport.SetContent(renderMessages(m.messages, m.width))
	m.viewport.GotoBottom()
	m.syncLayout()
}
