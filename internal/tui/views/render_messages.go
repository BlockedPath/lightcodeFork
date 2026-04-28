package views

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kartik-2239/lightcode/internal/server/db/models"
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
	title := styleTodoTitle.Render("Todo list")
	boxStyle := styleTodoBox
	if width > 8 {
		boxStyle = boxStyle.MaxWidth(width - 4)
	}
	var inner strings.Builder
	if len(todos) == 0 {
		inner.WriteString(styleTodoEmpty.Render("No tasks in this session."))
	} else {
		for _, t := range todos {
			mark := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[ ] ")
			line := styleTodoOpen.Render(t.Description)
			if t.Completed {
				mark = lipgloss.NewStyle().Foreground(lipgloss.Color("247")).Render("[x] ")
				line = styleTodoDone.Render(t.Description)
			}
			inner.WriteString(mark + line + "\n")
		}
	}
	boxed := boxStyle.Render(strings.TrimSuffix(inner.String(), "\n"))
	return dot + " " + title + "\n" + boxed
}

func formatToolCall(tc models.StoredToolCall) string {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(tc.Arguments), &args)
	values := ""
	for _, value := range args {
		cur := fmt.Sprintf("%v", value)
		if filepath.IsAbs(cur) {
			home, _ := os.UserHomeDir()
			cur = strings.Replace(cur, home, "~", 1)
		}
		values = values + strings.TrimSpace(cur) + " "
	}
	if err != nil {
		return styleToolName.Render(tc.Name) + "()"
	}
	if tc.Name == "write_file" || tc.Name == "edit" || tc.Name == "skill" {
		return styleToolName.Render(tc.Name)
	}
	return styleToolName.Render(tc.Name) + "(" + styleTree.Render(values) + ")"
}

func formatToolResult(content string, codeChanges []string, width int, tc models.StoredToolCall) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return styleResultText.Render("(no output)")
	}
	if strings.Contains(tc.Name, "todo") {
		return renderTodoStatusBlock(styleDot.Render("●"), models.DecodeToDoList(content), width)
	}
	if len(codeChanges) == 0 {
		lines := strings.Split(content, "\n")
		if len(lines) <= 4 {
			home, _ := os.UserHomeDir()
			content = strings.Replace(content, home, "~", 1)
			return styleResultText.Render(content)
		}
		return styleTree.Render(lines[0]+"\n") + styleTree.PaddingLeft(3).Render(strings.Join(lines[1:4], "\n")+"\n...")
	}

	var sb strings.Builder
	oldlines := strings.Split(codeChanges[0], "\n")
	newlines := strings.Split(codeChanges[1], "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Red).Render(fmt.Sprintf("-%d", len(oldlines))) + " " + lipgloss.NewStyle().Foreground(lipgloss.Green).Render(fmt.Sprintf("+%d", len(newlines))) + "\n")

	if len(newlines) > 4 {
		newlines = newlines[:4]
		newlines = append(newlines, "...")
	}
	if len(oldlines) > 4 {
		oldlines = oldlines[:4]

		oldlines = append(oldlines, "...")
	}
	for _, line := range oldlines {
		sb.WriteString(styleRemoved.Width(width).Render("- " + line))
		sb.WriteString("\n")
	}
	for _, line := range newlines {
		sb.WriteString(styleAdded.Width(width).Render("+ " + line))
		sb.WriteString("\n")
	}
	return sb.String()
}

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
	lines = append(lines, mascot())
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
					if len(matchedContent) > width {
						matchedContent = "Thinking: " + matchedContent
						formattedData := styleThink.PaddingLeft(1).Render(matchedContent[:width]) + styleThink.Width(width).PaddingLeft(2).Width(width).Render(matchedContent[width:])
						lines = append(lines, dot+formattedData)
					} else {
						lines = append(lines, styleThink.Width(width).Render(matchedContent))
					}
				}
				if content != "" {
					if len(content) > width {
						formattedData := styleThink.Width(width).PaddingLeft(1).Render(content[:width]) + styleThink.PaddingLeft(2).Width(width).Render(content[width:])
						lines = append(lines, dot+formattedData)
					} else {
						lines = append(lines, dot+styleThink.Width(width).Render(content))
					}
					lines = append(lines, dot+" "+content)
				}

				for _, tc := range d.ToolCalls {
					lines = append(lines, dot+" "+formatToolCall(tc))
				}

			case "tool_call":
				for _, toolcall := range d.ToolCalls {
					lines = append(lines, dot+" "+formatToolCall(toolcall))
					resultSummary := formatToolResult(content, d.CodeChanges, width, toolcall)
					lines = append(lines, tree+" "+resultSummary)
				}

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
			for i, tc := range d.ToolCalls {
				if !hasResult[callKey{fmt.Sprintf("%d", msg.ID), i}] {
					lines = append(lines, dot+" "+formatToolCall(tc))
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

func mascot() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#41f7fa")).Render(`
  ▐█████▌
  █  █  █
 ▘▜█████▛▘▘
   ▘▘ ▝▝ 
`)
}
