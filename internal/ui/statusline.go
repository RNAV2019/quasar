package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/RNAV2019/quasar/internal/editor"
	"github.com/RNAV2019/quasar/internal/errors"
	"github.com/RNAV2019/quasar/internal/styles"
)

var modeName = map[Mode]string{
	Normal:  "NORMAL",
	Insert:  "INSERT",
	Select:  "SELECT",
	Command: "COMMAND",
	NewNote: "NEW NOTE",
	Help:    "HELP",
	Error:   "ERROR",
}

func (m Model) getModeStyle() lipgloss.Style {
	return styles.GetModeStyle(int(m.mode))
}

func (m Model) getTimeStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(styles.ColorBackground).
		Background(styles.GetModeColor(int(m.mode))).
		Bold(true)
}

func (m Model) getLineNumberStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(styles.GetModeColor(int(m.mode))).
		Background(styles.ColorOverlay).
		Bold(true)
}

func (m Model) getFilename() string {
	for _, block := range m.Editor.Blocks {
		if block.Type == editor.TextBlock && len(block.Lines) > 0 {
			if block.Lines[0] == "---" {
				for i := 1; i < len(block.Lines); i++ {
					if block.Lines[i] == "---" {
						break
					}
					line := strings.TrimSpace(block.Lines[i])
					if after, ok := strings.CutPrefix(line, "title:"); ok {
						title := strings.TrimSpace(after)
						title = strings.Trim(title, `"'`)
						if title != "" {
							return editor.SanitizeFilename(title) + ".md"
						}
					}
				}
			}
			break
		}
	}

	title := m.Editor.ExtractTitle()
	if title != "" {
		return editor.SanitizeFilename(title) + ".md"
	}

	return "untitled.md"
}

// RenderStatusline renders the mode indicator, filename, position, and clock.
func (m Model) RenderStatusline() string {
	modeStyle := m.getModeStyle()
	renderedLeft := modeStyle.Render(" " + modeName[m.mode] + " ")

	var renderedCenter string
	if m.CurrentFile == "" {
		if m.NotebookName != "" {
			renderedCenter = styles.ClearStyle.Render(m.NotebookName)
		}
	} else if m.StatusMessage != "" {
		renderedCenter = styles.ClearStyle.Render(m.StatusMessage)
	} else {
		filename := m.getFilename()
		prefix := ""
		if m.Dirty {
			prefix = "[+] "
		}
		renderedCenter = styles.ClearStyle.Render(prefix + filename)
	}

	cursorLine := 1
	cursorCol := 1
	if m.CurrentFile != "" && m.Editor.Cursor.BlockIdx < len(m.Editor.Blocks) {
		for i := 0; i < m.Editor.Cursor.BlockIdx; i++ {
			cursorLine += len(m.Editor.Blocks[i].Lines)
		}
		cursorLine += m.Editor.Cursor.LineIdx
		cursorCol = m.Editor.Cursor.Col + 1
	}

	var statusError string
	for _, block := range m.Editor.Blocks {
		if block.HasError && block.ErrorMessage != "" {
			statusError = block.ErrorMessage
			break
		}
	}

	// Key preview (shown in non-insert modes when there's a pending key)
	var renderedKeyPreview string
	if m.KeyPreview != "" && m.mode != Insert {
		keyPreviewStyle := lipgloss.NewStyle().
			Foreground(styles.GetModeColor(int(m.mode))).
			Background(styles.ColorOverlay).
			Padding(0, 1)
		renderedKeyPreview = keyPreviewStyle.Render(m.KeyPreview)
	}

	// Error count indicator
	errorCount := errors.ErrorCount()
	var renderedErrorCount string
	if errorCount > 0 {
		errorStyle := lipgloss.NewStyle().
			Foreground(styles.ColorBackground).
			Background(lipgloss.Color("#ff6b6b")).
			Bold(true)
		renderedErrorCount = errorStyle.Render(fmt.Sprintf(" %d err ", errorCount))
	}

	var renderedRight string
	if statusError != "" {
		renderedRight = styles.ErrorStyle.Render(" " + statusError + " ")
	} else {
		sep := "\ue0b2"
		sepStyle := lipgloss.NewStyle().Foreground(styles.ColorOverlay)
		renderedSep := sepStyle.Render(sep)

		var posStr string
		if m.CurrentFile == "" {
			posStr = " ~ "
		} else {
			posStr = fmt.Sprintf(" %d:%d \ue0b2", cursorLine, cursorCol)
		}
		lineNumberStyle := m.getLineNumberStyle()
		renderedPos := lineNumberStyle.Render(posStr)

		timeStr := fmt.Sprintf(" \uf017 %s ", m.Time.Format("15:04"))
		timeStyle := m.getTimeStyle()
		renderedTime := timeStyle.Render(timeStr)

		rightParts := []string{renderedSep, renderedPos, renderedTime}
		if renderedKeyPreview != "" {
			rightParts = append([]string{renderedSep, renderedKeyPreview}, rightParts[1:]...)
		}
		if renderedErrorCount != "" {
			rightParts = append([]string{renderedErrorCount}, rightParts...)
		}
		renderedRight = lipgloss.JoinHorizontal(lipgloss.Top, rightParts...)
	}

	wLeft := lipgloss.Width(renderedLeft)
	wCenter := lipgloss.Width(renderedCenter)
	wRight := lipgloss.Width(renderedRight)

	centerStart := m.width / 2
	if wCenter > 0 {
		centerStart = m.width/2 - wCenter/2
	}

	gap1Width := max(centerStart-wLeft, 0)
	gap2Width := max(m.width-wLeft-gap1Width-wCenter-wRight, 0)

	gap1 := styles.ClearStyle.Width(gap1Width).Render(strings.Repeat(" ", gap1Width))
	gap2 := styles.ClearStyle.Width(gap2Width).Render(strings.Repeat(" ", gap2Width))

	statusLine := lipgloss.JoinHorizontal(lipgloss.Top,
		renderedLeft,
		gap1,
		renderedCenter,
		gap2,
		renderedRight,
	)

	return statusLine
}
