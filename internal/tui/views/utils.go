package views

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

func FormatK(n int64) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return strconv.FormatInt(n, 10)
}

func shortenDir(curDir string) string {
	home, _ := os.UserHomeDir()
	return strings.Replace(curDir, home, "~", 1)
}

func countWrappedLines(text string, width int, m *model) int {
	return len(wrapTextLines(text, width))
}

func wrapTextLines(text string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	var lines []string
	for _, logicalLine := range strings.Split(text, "\n") {
		if logicalLine == "" {
			lines = append(lines, "")
			continue
		}

		var current strings.Builder
		currentWidth := 0
		for _, r := range logicalLine {
			runeWidth := lipgloss.Width(string(r))
			if runeWidth == 0 {
				runeWidth = 1
			}
			if currentWidth+runeWidth > width {
				lines = append(lines, current.String())
				current.Reset()
				currentWidth = runeWidth
				current.WriteRune(r)
				continue
			}
			current.WriteRune(r)
			currentWidth += runeWidth
		}
		lines = append(lines, current.String())
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func wrappedCursorPosition(text string, logicalLine int, column int, width int) (int, int) {
	if width <= 0 {
		return 0, 0
	}
	logicalLines := strings.Split(text, "\n")
	if logicalLine < 0 {
		logicalLine = 0
	}
	if logicalLine >= len(logicalLines) {
		logicalLine = len(logicalLines) - 1
	}

	y := 0
	for i := 0; i < logicalLine; i++ {
		y += len(wrapTextLines(logicalLines[i], width))
	}

	x := 0
	currentColumn := 0
	for _, r := range logicalLines[logicalLine] {
		if currentColumn >= column {
			break
		}
		runeWidth := lipgloss.Width(string(r))
		if runeWidth == 0 {
			runeWidth = 1
		}
		if x+runeWidth > width {
			y++
			x = 0
		}
		x += runeWidth
		currentColumn++
	}

	return x, y
}
