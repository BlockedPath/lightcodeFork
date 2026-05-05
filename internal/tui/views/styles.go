package views

import "charm.land/lipgloss/v2"

var (
	styleDot        = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	styleToolName   = lipgloss.NewStyle().Bold(true)
	styleTree       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleUser       = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true).Background(lipgloss.Color("236")).Margin(1, 0).AlignVertical(lipgloss.Center)
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
	styleTodoDone  = lipgloss.NewStyle().Foreground(lipgloss.Color("247")).Strikethrough(true)
	styleTodoOpen  = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	styleTodoBox   = lipgloss.NewStyle().
			Padding(0, 1).
			MarginLeft(2)
	styleQuestionHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("43")).Bold(true)
	styleOptionNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleOptionSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("43")).Bold(true)
)
