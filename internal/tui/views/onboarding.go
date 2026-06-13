package views

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/config"
)

type onbTickMsg time.Time

func onbTick() tea.Cmd {
	return tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
		return onbTickMsg(t)
	})
}

type onboardingStage int

const (
	stageSelect onboardingStage = iota
	stageKeys
)

type onboardingProvider struct {
	name        string
	description string
	selected    bool
}

type onboardingModel struct {
	stage     onboardingStage
	providers []onboardingProvider
	cursor    int
	selected  []string // provider names, in display order
	keyIndex  int
	keys      map[string]string
	input     textinput.Model
	showEmpty bool // "select at least one" warning
	frame     int  // animation frame for pending dots
	done      bool
	err       error
}

func newOnboardingModel() onboardingModel {
	ti := textinput.New()
	ti.Placeholder = "paste API key (or leave blank to add later)"
	ti.Prompt = ""
	ti.SetWidth(60)

	return onboardingModel{
		stage: stageSelect,
		providers: []onboardingProvider{
			{name: "openrouter", description: "OpenRouter   300+ models via one API key"},
			{name: "openai", description: "OpenAI       e.g. GPT-5"},
			{name: "anthropic", description: "Anthropic    e.g. Opus 4.8"},
		},
		keys:  map[string]string{},
		input: ti,
	}
}

func (m onboardingModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, onbTick())
}

func (m onboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case onbTickMsg:
		m.frame++
		return m, onbTick()
	case tea.KeyPressMsg:
		switch m.stage {
		case stageSelect:
			return m.updateSelect(msg)
		case stageKeys:
			return m.updateKeys(msg)
		}
	}
	return m, nil
}

func (m onboardingModel) updateSelect(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.providers)-1 {
			m.cursor++
		}
	case " ", "space":
		m.providers[m.cursor].selected = !m.providers[m.cursor].selected
		m.showEmpty = false
	case "enter":
		m.selected = nil
		for _, p := range m.providers {
			if p.selected {
				m.selected = append(m.selected, p.name)
			}
		}
		if len(m.selected) == 0 {
			m.showEmpty = true
			return m, nil
		}
		// move to API-key entry for the first selected provider
		m.stage = stageKeys
		m.keyIndex = 0
		m.input.SetValue("")
		m.input.Placeholder = keyPlaceholder(m.selected[0])
		m.input.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

// pendingDots returns a cycling "", ".", "..", "..." padded to a stable width.
func pendingDots(frame int) string {
	n := frame % 4
	return strings.Repeat(".", n) + strings.Repeat(" ", 3-n)
}

// keyPlaceholder spells out which provider's key to paste, with an example prefix.
func keyPlaceholder(provider string) string {
	switch provider {
	case "openrouter":
		return "paste your OpenRouter API key (sk-or-v1-...)"
	case "openai":
		return "paste your OpenAI API key (sk-...)"
	case "anthropic":
		return "paste your Anthropic API key (sk-ant-...)"
	default:
		return "paste your " + provider + " API key"
	}
}

func (m onboardingModel) updateKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		// go back to provider selection
		m.stage = stageSelect
		m.input.Blur()
		return m, nil
	case "enter":
		name := m.selected[m.keyIndex]
		m.keys[name] = strings.TrimSpace(m.input.Value())
		m.keyIndex++
		if m.keyIndex >= len(m.selected) {
			m.err = config.CreateConfig(m.selected, m.keys)
			m.done = true
			return m, tea.Quit
		}
		m.input.SetValue("")
		m.input.Placeholder = keyPlaceholder(m.selected[m.keyIndex])
		return m, nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

var (
	onbTitle    = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	onbSubtitle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	onbHint     = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	onbChecked  = lipgloss.NewStyle().Foreground(lipgloss.Color("43")).Bold(true)
	onbCursor   = lipgloss.NewStyle().Foreground(lipgloss.BrightMagenta).Bold(true)
	onbNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	onbErr      = lipgloss.NewStyle().Foreground(lipgloss.BrightRed)
	onbDone     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (m onboardingModel) View() tea.View {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(onbTitle.Render("Welcome to Lightcode!"))
	sb.WriteString("\n\n")

	switch m.stage {
	case stageSelect:
		sb.WriteString(m.viewSelect())
	case stageKeys:
		sb.WriteString(m.viewKeys())
	}

	return tea.NewView(sb.String())
}

func (m onboardingModel) viewSelect() string {
	var sb strings.Builder
	sb.WriteString(onbSubtitle.Render("Step 1/2 · Select the providers you plan to use:"))
	sb.WriteString("\n\n")

	for i, p := range m.providers {
		cursor := "  "
		if i == m.cursor {
			cursor = onbCursor.Render("▸ ")
		}
		checkbox := "[ ]"
		style := onbNormal
		if p.selected {
			checkbox = "[✓]"
			style = onbChecked
		}
		sb.WriteString(fmt.Sprintf("%s%s %s\n", cursor, checkbox, style.Render(p.description)))
	}

	sb.WriteString("\n")
	sb.WriteString(onbHint.Render("↑↓ navigate · Space toggle · Enter continue"))
	if m.showEmpty {
		sb.WriteString("\n")
		sb.WriteString(onbErr.Render("Select at least one provider"))
	}
	sb.WriteString("\n")
	return sb.String()
}

func (m onboardingModel) viewKeys() string {
	var sb strings.Builder
	sb.WriteString(onbSubtitle.Render(fmt.Sprintf("Step 2/2 · Enter API keys (%d/%d):", m.keyIndex+1, len(m.selected))))
	sb.WriteString("\n\n")

	for i, name := range m.selected {
		switch {
		case i < m.keyIndex:
			status := "saved"
			if m.keys[name] == "" {
				status = "skipped"
			}
			marker := onbDone.Render("✓ ")
			label := onbDone.Render(fmt.Sprintf("%-12s", name))
			sb.WriteString(fmt.Sprintf("%s%s %s\n", marker, label, onbDone.Render(status)))
		case i == m.keyIndex:
			marker := onbCursor.Render("❯ ")
			label := onbChecked.Render(fmt.Sprintf("%-12s", name))
			sb.WriteString(fmt.Sprintf("%s%s %s\n", marker, label, m.input.View()))
		default:
			marker := "  "
			label := onbNormal.Render(fmt.Sprintf("%-12s", name))
			sb.WriteString(fmt.Sprintf("%s%s %s\n", marker, label, onbDone.Render("pending"+pendingDots(m.frame))))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(onbHint.Render("Enter confirm · blank to skip · Esc back"))
	sb.WriteString("\n")
	return sb.String()
}

func RunOnboarding() error {
	p := tea.NewProgram(newOnboardingModel())
	result, err := p.Run()
	if err != nil {
		return err
	}
	m, ok := result.(onboardingModel)
	if !ok {
		return fmt.Errorf("unexpected onboarding model type")
	}
	if m.err != nil {
		return m.err
	}
	if !m.done {
		fmt.Fprintln(os.Stderr, "Onboarding cancelled.")
		os.Exit(0)
	}
	return nil
}
