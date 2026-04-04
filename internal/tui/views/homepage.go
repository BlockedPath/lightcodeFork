package views

// A simple program demonstrating the text area component from the Bubbles
// component library.

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"
	"time"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/tui/client"
	"github.com/Kartik-2239/lightcode/internal/tui/components"
	"github.com/charmbracelet/x/term"
	"golang.design/x/clipboard"
)

func LauchHomePage() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Oof: %v\n", err)
	}
}

type streamMessageMsg models.StoredMessageData
type streamDoneMsg struct{}

type questionItem struct {
	question string
	options  []string
}

type model struct {
	viewport          viewport.Model
	islistSessionWin  bool
	islistCommandsWin bool
	listSession       components.Model
	listCommands      components.ModelCmdList
	sessions          []models.Session
	currentSession    models.Session
	messages          []models.Message
	textarea          textarea.Model
	pasteCounter      int
	pastedTexts       map[int]string
	senderStyle       lipgloss.Style
	err               error
	cache             map[int]string
	cacheIndex        int
	streamCh          chan models.StoredMessageData
	width             int
	height            int
	bashMode          bool
	streamch          chan models.StoredMessageData
	cancelStream      context.CancelFunc
	spinner           spinner.Model
	isGenerating      bool
	lastEsc           time.Time
	showEscMsg        bool
	questionMode      bool
	questions         []questionItem
	questionIdx       int
	questionAnswers   []string
	questionSelected  int
	todoList          []models.ToDo
	mode              string
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.SetVirtualCursor(false)
	ta.Focus()

	ta.Prompt = "┃ "
	// ta.SetPromptFunc(2, func(info textarea.PromptInfo) string {
	// 	if info.LineNumber == 0 {
	// 		return "❯ "
	// 	}
	// 	return " "
	// })

	ta.CharLimit = 32000

	s := ta.Styles()
	s.Focused.CursorLine = lipgloss.NewStyle()

	s.Focused.Base = lipgloss.NewStyle()

	width, height := 80, 24
	if w, h, err := term.GetSize(os.Stdout.Fd()); err == nil {
		width, height = w, h
	}

	ta.SetWidth(width)
	ta.SetHeight(3)

	ta.SetStyles(s)

	ta.ShowLineNumbers = false

	vp := viewport.New(viewport.WithWidth(width), viewport.WithHeight(height-ta.Height()))
	vp.KeyMap.Left.SetEnabled(false)
	vp.KeyMap.Right.SetEnabled(false)

	ta.KeyMap.InsertNewline.SetEnabled(false)
	sessions := client.ListSession()
	sessionItems := make([]list.Item, len(sessions))
	for i, s := range sessions {
		sessionItems[i] = components.NewItem(s.Title, s.Directory)
	}

	spin := spinner.New()
	spin.Spinner = spinner.MiniDot
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := model{
		textarea:          ta,
		pasteCounter:      0,
		pastedTexts:       make(map[int]string),
		messages:          []models.Message{},
		cacheIndex:        0,
		cache:             make(map[int]string),
		viewport:          vp,
		islistSessionWin:  false,
		islistCommandsWin: false,
		bashMode:          false,
		listSession:       components.LaunchSessionList(sessionItems),
		listCommands:      components.LaunchCommandList(),
		sessions:          sessions,
		senderStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:               nil,
		width:             width,
		spinner:           spin,
		isGenerating:      false,
		lastEsc:           time.Now(),
		mode:              "chat",
	}
	m.syncLayout()
	return m
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m *model) syncLayout() {
	m.textarea.SetWidth(m.width)
	m.viewport.SetWidth(m.width)

	reservedHeight := m.textarea.Height()
	if m.isGenerating {
		reservedHeight++
	}
	if m.mode == "chat" || m.mode == "plan" {
		reservedHeight++
	}
	if m.islistCommandsWin {
		reservedHeight += m.listCommands.Height()
	}
	if m.bashMode {
		reservedHeight++
	}
	if m.questionMode {
		reservedHeight += m.questionUIHeight()
	}

	viewportHeight := m.height - reservedHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	m.viewport.SetHeight(viewportHeight)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.islistSessionWin {
		var cmd tea.Cmd
		updatedModel, cmd := m.listSession.Update(msg)
		m.listSession = updatedModel.(components.Model)
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "enter":
				cur_idx := m.listSession.Current()
				m.currentSession = m.sessions[cur_idx]
				m.messages = client.GetSessionData(m.currentSession.ID)
				m.todoList = client.GetCurrentTodoList(m.currentSession.ID)
				m.islistSessionWin = false
				m.syncLayout()
				m.viewport.SetContent(renderMessages(m.messages, m.width))
				m.viewport.GotoBottom()
			}
		}
		return m, cmd
	}
	if m.islistCommandsWin {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "esc":
				m.islistCommandsWin = false
				m.syncLayout()
				return m, nil
			case "up", "down":
				var cmd tea.Cmd
				updatedModel, cmd := m.listCommands.Update(msg)
				m.listCommands = updatedModel.(components.ModelCmdList)
				return m, cmd
			case "enter":
				m.cacheIndex++
				cur_command := m.listCommands.Current()
				m.islistCommandsWin = false
				m.syncLayout()
				cmd := CmdHandler("/"+cur_command, &m)
				return m, cmd
			default:
				m.cache[m.cacheIndex] = m.textarea.Value()
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				val := m.textarea.Value()

				if !strings.HasPrefix(val, "/") {
					m.islistCommandsWin = false
					m.syncLayout()
				} else {
					m.listCommands.Filter(strings.TrimPrefix(val, "/"))
				}
				return m, cmd
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncLayout()

		if len(m.messages) > 0 {
			m.viewport.SetContent(renderMessages(m.messages, m.width))
		}
		m.viewport.GotoBottom()
	case tea.KeyPressMsg:
		if m.questionMode {
			return m.handleQuestionInput(msg)
		}
		switch msg.String() {
		case "ctrl+c":
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case "esc":
			if m.streamCh != nil && m.cancelStream != nil && time.Since(m.lastEsc) < 500*time.Millisecond {
				m.isGenerating = false
				m.cancelStream()
				m.cancelStream = nil
				m.streamCh = nil
				m.syncLayout()
				m.messages = append(m.messages, models.Message{
					SessionID: m.currentSession.ID,
					Data: models.EncodeMessageData(models.StoredMessageData{
						Role: "assistant", Content: "*Generation stopped.*",
					}),
				})
				m.viewport.SetContent(renderMessages(m.messages, m.width))
				m.viewport.GotoBottom()
				m.syncLayout()
				m.showEscMsg = false
				return m, nil
			}
			m.lastEsc = time.Now()
			m.showEscMsg = true
			return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
				return clearEscMsg()
			})
			// m.sessions = client.ListSession()
			// m.islistSessionWin = true

		case "ctrl+v", "super+v":
			curVal := m.textarea.Value()
			err := clipboard.Init()
			if err != nil {
				panic(err)
			}
			textBytes := clipboard.Read(clipboard.FmtText)
			pasteValue := string(textBytes)
			if strings.Count(pasteValue, "\n") > 1 {
				m.pasteCounter++
				m.pastedTexts[m.pasteCounter] = pasteValue
				placeholder := fmt.Sprintf("[pasted text #%d]", m.pasteCounter)
				m.textarea.SetValue(curVal + " " + placeholder)
			} else {
				m.textarea.SetValue(curVal + pasteValue)
			}
			return m, nil
		// case "shift+enter":
		// 	curVal := m.textarea.Value()
		// 	m.textarea.SetValue(curVal + "\n")
		// 	m.adjustTextareaHeight()
		// 	return m, nil
		case "enter":
			if strings.HasPrefix(m.textarea.Value(), "/") {
				cmd := CmdHandler(m.textarea.Value(), &m)
				return m, cmd
			}
			if m.currentSession.ID == "" {
				session_id := client.CreateSession((m.textarea.Value()))
				m.currentSession = models.Session{ID: session_id, Title: m.textarea.Value(), Directory: "."}
				m.sessions = append(m.sessions, m.currentSession)
				m.listSession.Refresh(m.sessions)
			}
			m.isGenerating = true
			m.syncLayout()
			textareaValue := createPrompt(strings.Trim(m.textarea.Value(), "\n"), &m)

			newMessage := client.SendMessage(m.currentSession.ID, textareaValue)
			m.messages = append(m.messages, newMessage)

			m.viewport.SetContent(renderMessages(m.messages, m.width))
			ctx, cancel := context.WithCancel(context.Background())
			ch := client.ChatCompletion(ctx, m.currentSession.ID, textareaValue, m.mode)
			m.cancelStream = cancel
			m.streamCh = ch
			m.textarea.SetValue("")
			m.textarea.Reset()
			m.viewport.GotoBottom()
			return m, tea.Batch(waitForMessages(ch), m.spinner.Tick)
		case "up", "down":
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		case "tab":
			if m.mode != "plan" {
				m.mode = "plan"
			} else {
				m.mode = "chat"
			}

		case "/":
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			wasListCommandsOpen := m.islistCommandsWin
			if len(m.textarea.Value()) == 1 {
				m.islistCommandsWin = true
			}
			if !wasListCommandsOpen && m.islistCommandsWin {
				m.syncLayout()
			}
			return m, cmd
		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			// m.adjustTextareaHeight()

			previousBashMode := m.bashMode
			if strings.HasPrefix(m.textarea.Value(), "!") {
				m.bashMode = true
				BashModeHandler(m.textarea.Value())
			} else {
				m.bashMode = false
				BashModeHandler(m.textarea.Value())
			}
			if previousBashMode != m.bashMode {
				m.syncLayout()
			}
			return m, cmd
		}

	case spinner.TickMsg:
		if !m.isGenerating {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case cursor.BlinkMsg:
		// Textarea should also process cursor blinks.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	case streamMessageMsg:
		m.messages = append(m.messages, models.Message{
			SessionID: m.currentSession.ID,
			// ID:        fmt.Sprintf("%s-assistant-%d", m.currentSession.ID, len(m.messages)),
			Data: models.EncodeMessageData(models.StoredMessageData(msg)),
		})

		m.viewport.SetContent(renderMessages(m.messages, m.width))
		m.syncLayout()
		m.viewport.GotoBottom()
		if msg.Role == "question" {
			questions := parseQuestions(msg.Content)
			if len(questions) > 0 {
				m.questionMode = true
				m.questions = questions
				m.questionIdx = 0
				m.questionAnswers = make([]string, len(questions))
				m.questionSelected = 0
				m.textarea.SetValue("")
				m.textarea.Reset()
				if len(questions[0].options) > 0 {
					m.textarea.Placeholder = "↑↓ select · Enter confirm"
					m.textarea.Blur()
				} else {
					m.textarea.Placeholder = "Type your answer..."
					m.textarea.Focus()
				}
				m.isGenerating = false
				m.syncLayout()
			}
		}
		return m, waitForMessages(m.streamCh)

	case streamDoneMsg:
		m.streamCh = nil
		m.isGenerating = false
		if m.currentSession.ID != "" {
			m.todoList = client.GetCurrentTodoList(m.currentSession.ID)
			m.messages = withoutEphemeralTodoStatus(m.messages)
			if len(m.todoList) != 0 {
				m.messages = append(m.messages, models.Message{
					SessionID: m.currentSession.ID,
					Data: models.EncodeMessageData(models.StoredMessageData{
						Role:    "todo_status",
						Content: models.EncodeToDoList(m.todoList),
					}),
				})
			}
		}
		m.viewport.SetContent(renderMessages(m.messages, m.width))
		m.viewport.GotoBottom()
		m.syncLayout()
		return m, nil

	case clearEscMsgMsg:
		m.showEscMsg = false
		return m, nil
	}

	return m, nil
}

func (m model) View() tea.View {
	if m.islistSessionWin {
		return m.listSession.View()
	}
	m.viewport.SetContent(
		// m.currentSession.ID +
		// "\n" +
		renderMessages(m.messages, m.width))

	sections := make([]string, 0, 5)
	if m.bashMode {
		sections = append(sections, "Bash Mode")
	}
	sections = append(sections, m.viewport.View())

	if m.questionMode {
		sections = append(sections, m.renderQuestionUI())
	}
	sections = append(sections, m.mode)
	textareaSectionIndex := len(sections)
	sections = append(sections, m.textarea.View())

	if m.isGenerating {
		if m.showEscMsg {
			sections = append(sections, m.spinner.View()+lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(" Press Esc again to cancel..."))
		} else {
			sections = append(sections, m.spinner.View()+" Generating...")
		}
	}

	if m.islistCommandsWin {
		sections = append(sections, m.listCommands.StringView())
	}

	v := tea.NewView(strings.Join(sections, "\n"))
	c := m.textarea.Cursor()
	if c != nil {
		if textareaSectionIndex > 0 {
			c.Y += lipgloss.Height(strings.Join(sections[:textareaSectionIndex], "\n"))
		}
	}
	v.Cursor = c
	v.AltScreen = true
	return v
}

var (
	styleDot        = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	styleToolName   = lipgloss.NewStyle().Bold(true)
	styleTree       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleUser       = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true).Background(lipgloss.Color("236")).Padding(0, 1)
	styleThink      = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Bold(false)
	styleResultText = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	styleAdded      = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#aff5b4")).
			Background(lipgloss.Color("#1a3a2a")).
			PaddingLeft(1)
	styleRemoved = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffdcd7")).
			Background(lipgloss.Color("#3d1a1f")).
			PaddingLeft(1)
	styleTodoTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	styleTodoEmpty = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	styleTodoDone  = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Strikethrough(true)
	styleTodoOpen  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleTodoBox   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1).
			MarginLeft(2)
)

func withoutEphemeralTodoStatus(msgs []models.Message) []models.Message {
	var out []models.Message
	for _, msg := range msgs {
		if models.DecodeMessageData(msg.Data).Role == "todo_status" {
			continue
		}
		out = append(out, msg)
	}
	return out
}

func renderTodoStatusBlock(dot string, todos []models.ToDo, width int) string {
	title := styleTodoTitle.Render("Task list")
	boxStyle := styleTodoBox
	if width > 8 {
		boxStyle = boxStyle.MaxWidth(width - 4)
	}
	var inner strings.Builder
	if len(todos) == 0 {
		inner.WriteString(styleTodoEmpty.Render("No tasks in this session."))
	} else {
		for i, t := range todos {
			prefix := "├ "
			if i == len(todos)-1 {
				prefix = "└ "
			}
			mark := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render("[ ] ")
			line := styleTodoOpen.Render(t.Description)
			if t.Completed {
				mark = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render("[✓] ")
				line = styleTodoDone.Render(t.Description)
			}
			inner.WriteString(styleTree.Render(prefix) + mark + line + "\n")
		}
	}
	boxed := boxStyle.Render(strings.TrimSuffix(inner.String(), "\n"))
	return dot + " " + title + "\n" + boxed
}

func formatToolCall(tc models.StoredToolCall) string {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(tc.Arguments), &args)
	values := []string{}
	for arg, value := range args {
		values = append(values, arg+": "+strings.TrimSpace(fmt.Sprintf("%v", value)))
	}
	if err != nil {
		return styleToolName.Render(tc.Name) + "()"
	}
	if len(values) > 7 {
		values = values[:7]
		values = append(values, values[7]+"...")
	}
	return styleToolName.Render(tc.Name) + "(" + styleTree.Render(strings.Join(values, ", ")) + ")"
}

func formatToolResult(content string, codeChanges []string, width int) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return styleResultText.Render("(no output)")
	}
	if len(codeChanges) == 0 {
		lines := strings.Split(content, "\n")
		if len(lines) <= 4 {
			return styleResultText.Render(content)
		}
		return styleTree.Render(strings.Join(lines[:4], "\n") + "...")
	}

	var sb strings.Builder
	sb.WriteString("\n")
	oldlines := strings.Split(codeChanges[0], "\n")
	if len(oldlines) > 4 {
		oldlines = oldlines[:4]
	}
	newlines := strings.Split(codeChanges[1], "\n")
	if len(newlines) > 4 {
		newlines = newlines[:4]
	}
	for _, line := range newlines {
		sb.WriteString(styleRemoved.Width(width).Render("- " + line))
		sb.WriteString("\n")
	}
	for _, line := range oldlines {
		sb.WriteString(styleAdded.Width(width).Render("+ " + line))
		sb.WriteString("\n")
	}
	return sb.String()
}

// lightcodeGlamourStyle is a custom glamour style tuned to match the app palette.
var lightcodeGlamourStyle = []byte(`{
  "document": {
    "block_prefix": "",
    "block_suffix": "",
    "color": "252",
    "margin": 0
  },
  "block_quote": {
    "indent": 1,
    "indent_token": "│ ",
    "color": "243",
    "italic": true
  },
  "paragraph": {},
  "list": {
    "level_indent": 2
  },
  "heading": {
    "block_suffix": "\n",
    "bold": true
  },
  "h1": {
    "prefix": " ",
    "suffix": " ",
    "color": "232",
    "background_color": "43",
    "bold": true
  },
  "h2": {
    "prefix": "▌ ",
    "color": "86",
    "bold": true
  },
  "h3": {
    "prefix": "◆ ",
    "color": "43",
    "bold": true
  },
  "h4": {
    "prefix": "◇ ",
    "color": "37",
    "bold": false
  },
  "h5": {
    "prefix": "· ",
    "color": "244"
  },
  "h6": {
    "prefix": "· ",
    "color": "241"
  },
  "text": {},
  "strikethrough": { "crossed_out": true },
  "emph": { "italic": true, "color": "245" },
  "strong": { "bold": true, "color": "255" },
  "hr": {
    "color": "237",
    "format": "\n──────────────────────────────────────\n"
  },
  "item": { "block_prefix": "• " },
  "enumeration": { "block_prefix": ". " },
  "task": { "ticked": "[✓] ", "unticked": "[ ] " },
  "link": { "color": "51", "underline": true },
  "link_text": { "color": "43", "bold": true },
  "image": { "color": "212", "underline": true },
  "image_text": { "color": "243", "format": "Image: {{.text}} →" },
  "code": {
    "prefix": " ",
    "suffix": " ",
    "color": "215",
    "background_color": "235"
  },
  "code_block": {
    "color": "252",
    "margin": 2,
    "chroma": {
      "text":                { "color": "#C4C4C4" },
      "error":               { "color": "#F1F1F1", "background_color": "#F05B5B" },
      "comment":             { "color": "#606060" },
      "comment_preproc":     { "color": "#FF875F" },
      "keyword":             { "color": "#41f7fa" },
      "keyword_reserved":    { "color": "#FF5FD2" },
      "keyword_namespace":   { "color": "#FF5F87" },
      "keyword_type":        { "color": "#86D0D0" },
      "operator":            { "color": "#8BE28B" },
      "punctuation":         { "color": "#C8C8A0" },
      "name":                { "color": "#C4C4C4" },
      "name_builtin":        { "color": "#FF8EC7" },
      "name_tag":            { "color": "#86D0D0" },
      "name_attribute":      { "color": "#7A7AE6" },
      "name_class":          { "color": "#F1F1F1", "underline": true, "bold": true },
      "name_constant":       {},
      "name_decorator":      { "color": "#FFFF87" },
      "name_exception":      {},
      "name_function":       { "color": "#41f7fa" },
      "name_other":          {},
      "literal_number":      { "color": "#6EEFC0" },
      "literal_string":      { "color": "#C69669" },
      "literal_string_escape": { "color": "#AFFFD7" },
      "generic_deleted":     { "color": "#FD5B5B" },
      "generic_emph":        { "italic": true },
      "generic_inserted":    { "color": "#41f7fa" },
      "generic_strong":      { "bold": true },
      "generic_subheading":  { "color": "#888888" },
      "background":          { "background_color": "#1e1e1e" }
    }
  },
  "table": {},
  "definition_list": {},
  "definition_term": {},
  "definition_description": { "block_prefix": "\n  → " },
  "html_block": {},
  "html_span": {}
}`)

func renderMessages(msgs []models.Message, width int) string {
	if width <= 0 {
		width = 80
	}
	r, _ := glamour.NewTermRenderer(glamour.WithWordWrap(width), glamour.WithStylesFromJSONBytes(lightcodeGlamourStyle))

	dot := styleDot.Render("●")
	tree := styleTree.Render("└─")

	// Pre-pass: Find which tool calls in which messages have corresponding result messages
	type callKey struct {
		msgID string
		idx   int
	}
	hasResult := make(map[callKey]bool)
	var lastAssistantMsgID string
	var callIdx int
	for _, msg := range msgs {
		d := models.DecodeMessageData(msg.Data)
		if d.Role == "assistant" {
			lastAssistantMsgID = fmt.Sprintf("%d", msg.ID)
			callIdx = 0
		} else if d.Role == "tool_call" && lastAssistantMsgID != "" {
			hasResult[callKey{lastAssistantMsgID, callIdx}] = true
			callIdx++
		}
	}

	var lines []string
	lines = append(lines, mascott())
	for _, msg := range msgs {
		d := models.DecodeMessageData(msg.Data)
		if d.Role == "" || d.Role == "error" {
			continue
		}

		if d.Content != "" {
			content := d.Content
			switch d.Role {
			case "assistant":
				content = html.UnescapeString(content)
				re := regexp.MustCompile(`(?s)<think>(.*?)</think>`)
				matches := re.FindAllString(content, -1)
				matchedContent := strings.Join(matches, " ")
				content = content[len(strings.Join(matches, " ")):]

				if out, err := r.Render(content); err == nil {
					content = strings.TrimSpace(out)
				}
				if matchedContent != "" {
					matchedContent = strings.TrimSpace(matchedContent)
					matchedContent = strings.ReplaceAll(matchedContent, "\n", "")
					matchedContent = strings.Replace(matchedContent, "<think>", "", 1)
					matchedContent = strings.Replace(matchedContent, "</think>", "", 1)
					lines = append(lines, dot+" Thinking: "+styleThink.Width(width).Render(matchedContent))
				}
				if content != "" {
					lines = append(lines, dot+" "+content)
				}

				// Only render tool calls that don't have results yet
				for i, tc := range d.ToolCalls {
					if !hasResult[callKey{fmt.Sprintf("%d", msg.ID), i}] {
						lines = append(lines, dot+" "+formatToolCall(tc))
					}
				}

			case "tool_call":
				// For tool results, render BOTH the call and the result
				if len(d.ToolCalls) > 0 {
					lines = append(lines, dot+" "+formatToolCall(d.ToolCalls[0]))
				}
				resultSummary := formatToolResult(content, d.CodeChanges, width)
				lines = append(lines, tree+" "+resultSummary)

			case "user":
				lines = append(lines, "")
				lines = append(lines, styleUser.Width(width).Render("> "+content))
				lines = append(lines, "")
			case "question":
				qs := parseQuestions(content)
				for _, q := range qs {
					line := styleQuestionHeader.Render("? " + q.question)
					if len(q.options) > 0 {
						line += styleTree.Render(" [" + strings.Join(q.options, ", ") + "]")
					}
					lines = append(lines, line)
				}
			case "todo_status":
				todos := models.DecodeToDoList(content)
				lines = append(lines, renderTodoStatusBlock(dot, todos, width))
			}
		} else if d.Role == "assistant" && len(d.ToolCalls) > 0 {
			// Assistant message with ONLY tool calls (no text content)
			for i, tc := range d.ToolCalls {
				if !hasResult[callKey{fmt.Sprintf("%d", msg.ID), i}] {
					lines = append(lines, dot+" "+formatToolCall(tc))
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

func waitForMessages(ch chan models.StoredMessageData) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return streamDoneMsg{}
		}
		return streamMessageMsg(msg)
	}
}

type clearEscMsgMsg struct {
}

func clearEscMsg() tea.Cmd {
	return func() tea.Msg {
		return clearEscMsgMsg{}
	}
}

func createPrompt(value string, m *model) string {
	re := regexp.MustCompile(`\[pasted text #(\d+)\]`)
	textareaValue := re.ReplaceAllStringFunc(value, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		var idx int
		fmt.Sscanf(sub[1], "%d", &idx)
		if real, ok := m.pastedTexts[idx]; ok {
			return real
		}
		return match
	})
	return textareaValue
}

type refreshSessionsMsg struct {
}

func CmdHandler(cmd string, m *model) tea.Cmd {
	switch cmd {
	case "/sessions":
		m.sessions = client.ListSession()
		m.islistSessionWin = true
		m.textarea.Reset()
	case "/new_session":
		m.currentSession = models.Session{ID: "", Title: "", Directory: "."}
		m.messages = []models.Message{}
		m.viewport.SetContent(renderMessages(m.messages, m.width))
		m.textarea.Reset()
		m.viewport.GotoBottom()
		return func() tea.Msg { return refreshSessionsMsg{} }

	case "/delete_session":
		session_id := m.currentSession.ID
		client.DeleteSession(session_id)
		var newSessions []models.Session
		for _, session := range m.sessions {
			if session.ID != session_id {
				newSessions = append(newSessions, session)
			}
		}
		m.sessions = newSessions
		m.currentSession = models.Session{ID: "", Title: "", Directory: "."}
		m.messages = []models.Message{}
		m.viewport.SetContent(renderMessages(m.messages, m.width))
		m.textarea.Reset()
		m.viewport.GotoBottom()
		return func() tea.Msg { return refreshSessionsMsg{} }
	}
	return nil
}

func BashModeHandler(cmd string) {

}

func parseQuestions(content string) []questionItem {
	var args map[string]any
	if err := json.Unmarshal([]byte(content), &args); err != nil {
		return nil
	}
	rawQuestions, ok := args["question"].([]any)
	if !ok {
		return nil
	}
	var questions []questionItem
	for _, item := range rawQuestions {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		q := questionItem{}
		q.question, _ = obj["question"].(string)
		if rawOpts, ok := obj["options"].([]any); ok {
			for _, o := range rawOpts {
				if s, ok := o.(string); ok {
					q.options = append(q.options, s)
				}
			}
		}
		if q.question != "" {
			questions = append(questions, q)
		}
	}
	return questions
}

var (
	styleQuestionHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("43")).Bold(true)
	styleOptionNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleOptionSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("43")).Bold(true)
)

func (m model) renderQuestionUI() string {
	if !m.questionMode || m.questionIdx >= len(m.questions) {
		return ""
	}
	q := m.questions[m.questionIdx]
	var lines []string
	header := fmt.Sprintf("? (%d/%d) %s", m.questionIdx+1, len(m.questions), q.question)
	lines = append(lines, styleQuestionHeader.Render(header))
	for i, opt := range q.options {
		if i == m.questionSelected {
			lines = append(lines, styleOptionSelected.Render("  ▸ "+opt))
		} else {
			lines = append(lines, styleOptionNormal.Render("    "+opt))
		}
	}
	return strings.Join(lines, "\n")
}

func (m model) questionUIHeight() int {
	if !m.questionMode || m.questionIdx >= len(m.questions) {
		return 0
	}
	h := 1
	h += len(m.questions[m.questionIdx].options)
	return h
}

func (m model) handleQuestionInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.questionMode = false
		m.textarea.Placeholder = "Send a message..."
		m.textarea.Focus()
		m.syncLayout()
		return m, nil
	case "enter":
		q := m.questions[m.questionIdx]
		if len(q.options) > 0 {
			m.questionAnswers[m.questionIdx] = q.options[m.questionSelected]
		} else {
			m.questionAnswers[m.questionIdx] = m.textarea.Value()
			m.textarea.SetValue("")
			m.textarea.Reset()
		}
		m.questionIdx++
		if m.questionIdx >= len(m.questions) {
			return m.submitQuestionAnswers()
		}
		m.questionSelected = 0
		next := m.questions[m.questionIdx]
		if len(next.options) > 0 {
			m.textarea.Placeholder = "↑↓ select · Enter confirm"
			m.textarea.Blur()
		} else {
			m.textarea.Placeholder = "Type your answer..."
			m.textarea.Focus()
		}
		m.syncLayout()
		return m, nil
	case "up":
		if len(m.questions[m.questionIdx].options) > 0 && m.questionSelected > 0 {
			m.questionSelected--
		}
		return m, nil
	case "down":
		q := m.questions[m.questionIdx]
		if len(q.options) > 0 && m.questionSelected < len(q.options)-1 {
			m.questionSelected++
		}
		return m, nil
	default:
		if len(m.questions[m.questionIdx].options) == 0 {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
		return m, nil
	}
}

func (m model) submitQuestionAnswers() (tea.Model, tea.Cmd) {
	m.questionMode = false
	m.textarea.Placeholder = "Send a message..."
	m.textarea.Focus()

	var parts []string
	for i, q := range m.questions {
		parts = append(parts, fmt.Sprintf("Q: %s\nA: %s", q.question, m.questionAnswers[i]))
	}
	answer := strings.Join(parts, "\n\n")

	newMessage := client.SendMessage(m.currentSession.ID, answer)
	m.messages = append(m.messages, newMessage)
	m.viewport.SetContent(renderMessages(m.messages, m.width))

	m.isGenerating = true
	m.syncLayout()

	ctx, cancel := context.WithCancel(context.Background())
	ch := client.ChatCompletion(ctx, m.currentSession.ID, answer, m.mode)
	m.cancelStream = cancel
	m.streamCh = ch
	m.viewport.GotoBottom()
	return m, tea.Batch(waitForMessages(ch), m.spinner.Tick)
}

func mascott() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#41f7fa")).Render(`
  ▐█████▌
  █  █  █
 ▘▜█████▛▘▘
   ▘▘ ▝▝ 
`)
}
