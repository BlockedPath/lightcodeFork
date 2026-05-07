package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/tui/client"
)

type refreshSessionsMsg struct{}

func CmdHandler(cmd string, m *model) tea.Cmd {
	switch cmd {
	case "/sessions":
		m.sessions = client.ListSession()
		m.listSession.Refresh(m.sessions)
		m.islistSessionWin = true
		m.textarea.Reset()
	case "/new_session":
		resetCurrentSession(m)
		return func() tea.Msg { return refreshSessionsMsg{} }

	case "/delete_session":
		deleteCurrentSession(m)
		return func() tea.Msg { return refreshSessionsMsg{} }

	case "/models":
		openModelsList(m)
		return nil

	case "/usage":
		appendUsageMessage(m)
		return nil
	case "/skills":
		appendSkillsMessage(m)
	case "/dir":
		appendDirMessage(m)
	default:
		return nil
	}
	if m.isGenerating {
		return waitForMessages(m.streamCh)
	}
	return nil
}

func resetCurrentSession(m *model) {
	m.currentSession = models.Session{ID: "", Title: "", Directory: "."}
	m.messages = []models.Message{}
	m.viewport.SetContent(renderMessages(m.messages, m.width))
	m.textarea.Reset()
	m.viewport.GotoBottom()
}

func deleteCurrentSession(m *model) {
	sessionID := m.currentSession.ID
	client.DeleteSession(sessionID)
	var newSessions []models.Session
	for _, session := range m.sessions {
		if session.ID != sessionID {
			newSessions = append(newSessions, session)
		}
	}
	m.sessions = newSessions
	resetCurrentSession(m)
}

func openModelsList(m *model) {
	m.textarea.Reset()
	// list, err := config.GetModels()
	list, err := client.GetModels()
	if err != nil {
		list = []config.ResModel{}
	}
	m.modelsList = list
	m.modelsListIndex = 0
	m.isModelsListWin = true
	m.textarea.Placeholder = "↑↓ select · Enter/Esc close"
	m.textarea.Blur()
	m.syncLayout()
}

func appendUsageMessage(m *model) {
	usageContent := buildUsageContent(m.currentSession)
	m.messages = append(m.messages, models.Message{
		SessionID: m.currentSession.ID,
		Data:      models.EncodeMessageData(models.StoredMessageData{Role: "assistant", Content: usageContent}),
	})
	m.viewport.SetContent(renderMessages(m.messages, m.width))
	m.viewport.GotoBottom()
	m.syncLayout()
}
func appendSkillsMessage(m *model) {
	available_skills := client.GetAvailableSkills(m.currentSession.ID)
	formatted_skills_list := lipgloss.NewStyle().Render("Available Skills: \n" + strings.Join(available_skills, ", "))
	m.messages = append(m.messages, models.Message{
		SessionID: m.currentSession.ID,
		Data:      models.EncodeMessageData(models.StoredMessageData{Role: "assistant", Content: formatted_skills_list}),
	})
	m.viewport.SetContent(renderMessages(m.messages, m.width))
	m.viewport.GotoBottom()
	m.syncLayout()
}
func appendDirMessage(m *model) {
	var dir string
	if m.currentSession.Directory == "" {
		dir = "start a session"
	} else {
		dir = shortenDir(m.currentSession.Directory)
	}
	m.messages = append(m.messages, models.Message{
		SessionID: m.currentSession.ID,
		Data:      models.EncodeMessageData(models.StoredMessageData{Role: "assistant", Content: dir}),
	})
	m.viewport.SetContent(renderMessages(m.messages, m.width))
	m.viewport.GotoBottom()
	m.syncLayout()
}

func buildUsageContent(session models.Session) string {
	if session.ID == "" {
		return "## Session Usage\n\nNo active session selected."
	}

	sessionMessages := client.GetSessionData(session.ID)
	var promptTokens int64
	var completionTokens int64
	var totalTokens int64
	messageCount := 0
	for _, msg := range sessionMessages {
		data := models.DecodeMessageData(msg.Data)
		if data.Usage == nil {
			continue
		}
		promptTokens += data.Usage.PromptTokens
		completionTokens += data.Usage.CompletionTokens
		totalTokens += data.Usage.TotalTokens
		messageCount++
	}

	if messageCount == 0 {
		return "## Session Usage\n\nNo token usage has been recorded for this session yet."
	}

	title := strings.TrimSpace(session.Title)
	if title == "" {
		title = "Untitled Session"
	}

	return fmt.Sprintf(
		"## Session Usage\n\n- Session: %s\n- Prompt tokens: %d\n- Completion tokens: %d\n- Total tokens: %d\n",
		title,
		promptTokens,
		completionTokens,
		totalTokens,
	)
}
