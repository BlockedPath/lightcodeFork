package views

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/config"
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
	modes             []string
	mode              string
	modelsList        []config.ResModel
	isModelsListWin   bool
	modelsListIndex   int
	queue             []string
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.SetVirtualCursor(false)
	ta.Focus()

	ta.Prompt = "┃ "

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

	modelsList, err := config.GetModels()
	if len(modelsList) == 0 {
		fmt.Println("No models found, add models in ~/.lightcode/config.json")
		os.Exit(1)
	}
	if err != nil {
		modelsList = []config.ResModel{}
	}

	currentModel := config.GetCurrentModel()
	currentModelIndex := 0
	for i, model := range modelsList {
		if model.Model == currentModel.Model {
			currentModelIndex = i
			break
		}
	}

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
		modes:             []string{"chat", "plan", "assistant"},
		modelsList:        modelsList,
		isModelsListWin:   false,
		modelsListIndex:   currentModelIndex,
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
	if len(m.mode) > 0 {
		reservedHeight++
	}
	if len(m.queue) > 0 {
		reservedHeight += len(m.queue) + 1
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
	if m.isModelsListWin {
		reservedHeight += m.modelsListHeight()
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
		if m.isModelsListWin {
			return m.handleModelsListInput(msg)
		}
		switch msg.String() {
		case "ctrl+c":
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case "esc":
			if m.streamCh != nil && m.cancelStream != nil && time.Since(m.lastEsc) < 500*time.Millisecond {
				m.cancelActiveGeneration()
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
			m.ensureCurrentSession(m.textarea.Value())
			if m.isGenerating == true {
				m.queue = append(m.queue, createPrompt(strings.Trim(m.textarea.Value(), "\n"), &m))
				m.textarea.SetValue("")
				m.syncLayout()
				return m, nil
			}
			return m, m.beginGeneration(m.textarea.Value())
		case "up", "down":
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		case "tab":
			for i, v := range m.modes {
				if v == m.mode {
					m.mode = m.modes[(i+1)%len(m.modes)]
					break
				}
			}
			return m, nil

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
		if msg.Role == "error" {
			return m, nil
		}
		m.messages = append(m.messages, models.Message{
			SessionID: m.currentSession.ID,
			// ID:        fmt.Sprintf("%s-assistant-%d", m.currentSession.ID, len(m.messages)),
			Data: models.EncodeMessageData(models.StoredMessageData(msg)),
		})

		m.refreshMessagesView()
		m.syncLayout()
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
			// temp will turn this into a function
			if len(m.queue) > 0 {
				return m, m.runNextQueuedPrompt()
			}

		}
		m.refreshMessagesView()
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
	sections = append(sections, m.viewport.View())

	if m.questionMode {
		sections = append(sections, m.renderQuestionUI())
	}
	if m.isModelsListWin {
		sections = append(sections, m.renderModelsList())
	}
	if len(m.queue) > 0 {
		sections = append(sections, m.renderQueueList())
	}

	textareaSectionIndex := len(sections)
	sections = append(sections, m.textarea.View())
	if !m.islistCommandsWin {
		s := strings.ToUpper(m.mode[:1]) + m.mode[1:]
		sections = append(sections, lipgloss.NewStyle().Foreground(lipgloss.BrightMagenta).Bold(true).Render(s)+" "+lipgloss.NewStyle().Foreground(lipgloss.Color("43")).Bold(false).Render(m.modelsList[m.modelsListIndex].Model))
	}

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
	if m.isModelsListWin {
		c = nil
	} else if c != nil {
		if textareaSectionIndex > 0 {
			c.Y += lipgloss.Height(strings.Join(sections[:textareaSectionIndex], "\n"))
		}
	}
	v.Cursor = c
	v.AltScreen = true
	return v
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

func (m model) handleModelsListInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "enter":
		m.isModelsListWin = false
		m.textarea.Placeholder = "Send a message..."
		config.SetCurrentModel(m.modelsList[m.modelsListIndex])
		m.textarea.Focus()
		(&m).syncLayout()
		return m, nil
	case "up":
		if len(m.modelsList) == 0 {
			return m, nil
		}
		m.modelsListIndex--
		if m.modelsListIndex < 0 {
			m.modelsListIndex = len(m.modelsList) - 1
		}
		return m, nil
	case "down":
		if len(m.modelsList) == 0 {
			return m, nil
		}
		m.modelsListIndex++
		if m.modelsListIndex >= len(m.modelsList) {
			m.modelsListIndex = 0
		}
		return m, nil
	default:
		return m, nil
	}
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

func (m model) renderQueueList() string {
	header := styleTodoTitle.Render("Queue")
	lines := []string{header}
	if len(m.queue) == 0 {
		lines = append(lines, styleOptionNormal.Render(" no models configured"))
		return strings.Join(lines, "\n")
	}
	for i, ent := range m.queue {
		label := ent
		lines = append(lines, styleOptionNormal.Render(" "+strconv.Itoa(i+1)+".) "+label))
	}
	return strings.Join(lines, "\n")
}

func (m model) renderModelsList() string {
	header := styleQuestionHeader.Render("Models")
	lines := []string{header}
	if len(m.modelsList) == 0 {
		lines = append(lines, styleOptionNormal.Render(" no models configured"))
		return strings.Join(lines, "\n")
	}
	for i, ent := range m.modelsList {
		label := ent.Model
		if i == m.modelsListIndex {
			lines = append(lines, styleOptionSelected.Render("  ▸ "+label))
		} else {
			lines = append(lines, styleOptionNormal.Render("    "+label))
		}
	}
	return strings.Join(lines, "\n")
}

func (m model) modelsListHeight() int {
	if len(m.modelsList) == 0 {
		return 2
	}
	return 1 + len(m.modelsList)
}

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
	return m, m.beginGeneration(answer)
}
