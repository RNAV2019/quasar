package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/RNAV2019/quasar/internal/editor"
)

var (
	// Mode-specific styles for the statusline left side
	normalModeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#181825")).
			Background(lipgloss.Color("#85aff3")).
			Bold(true)

	insertModeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#181825")).
			Background(lipgloss.Color("#a1dc9c")).
			Bold(true)

	selectModeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#181825")).
			Background(lipgloss.Color("#c5a1f0")).
			Bold(true)

	commandModeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#181825")).
				Background(lipgloss.Color("#f3ae83")).
				Bold(true)

	clearStyle = lipgloss.NewStyle()
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

var modeName = map[Mode]string{
	Normal:  "NORMAL",
	Insert:  "INSERT",
	Select:  "SELECT",
	Command: "COMMAND",
	NewNote: "NEW NOTE",
	Help:    "HELP",
}

// getModeStyle returns the appropriate style for the current mode
func (m Model) getModeStyle() lipgloss.Style {
	switch m.mode {
	case Insert:
		return insertModeStyle
	case Select:
		return selectModeStyle
	case Command:
		return commandModeStyle
	case NewNote:
		return normalModeStyle
	default:
		return normalModeStyle
	}
}

// getTimeStyle returns the time section style for the current mode
func (m Model) getTimeStyle() lipgloss.Style {
	modeColor := "#85aff3" // normal
	switch m.mode {
	case Insert:
		modeColor = "#a1dc9c"
	case Select:
		modeColor = "#c5a1f0"
	case Command:
		modeColor = "#f3ae83"
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#181825")).
		Background(lipgloss.Color(modeColor)).
		Bold(true)
}

// getLineNumberStyle returns the time section style for the current mode
func (m Model) getLineNumberStyle() lipgloss.Style {
	modeColor := "#85aff3" // normal
	switch m.mode {
	case Insert:
		modeColor = "#a1dc9c"
	case Select:
		modeColor = "#c5a1f0"
	case Command:
		modeColor = "#f3ae83"
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(modeColor)).
		Background(lipgloss.Color("#313244")).
		Bold(true)
}

// getFilename extracts the filename from the document's front matter or title
func (m Model) getFilename() string {
	// Try to extract from front matter first
	for _, block := range m.Editor.Blocks {
		if block.Type == editor.TextBlock && len(block.Lines) > 0 {
			// Check for front matter
			if block.Lines[0] == "---" {
				for i := 1; i < len(block.Lines); i++ {
					if block.Lines[i] == "---" {
						break
					}
					// Look for title: field
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

	// Fall back to extracting title from content
	title := m.Editor.ExtractTitle()
	if title != "" {
		return editor.SanitizeFilename(title) + ".md"
	}

	return "untitled.md"
}

// RenderStatusline renders the statusline with mode indicator, filename, and position info.
// Returns the rendered statusline string.
func (m Model) RenderStatusline() string {
	modeStyle := m.getModeStyle()
	renderedLeft := modeStyle.Render(" " + modeName[m.mode] + " ")

	// Get filename for center
	var renderedCenter string
	if m.CurrentFile == "" {
		// No file open - show notebook name or empty
		if m.NotebookName != "" {
			renderedCenter = clearStyle.Render(m.NotebookName)
		}
	} else if m.StatusMessage != "" {
		renderedCenter = clearStyle.Render(m.StatusMessage)
	} else {
		filename := m.getFilename()
		renderedCenter = clearStyle.Render(filename)
	}

	// Get cursor position for right side
	cursorLine := 1
	cursorCol := 1
	if m.CurrentFile != "" && m.Editor.Cursor.BlockIdx < len(m.Editor.Blocks) {
		// Calculate global line number
		for i := 0; i < m.Editor.Cursor.BlockIdx; i++ {
			cursorLine += len(m.Editor.Blocks[i].Lines)
		}
		cursorLine += m.Editor.Cursor.LineIdx
		cursorCol = m.Editor.Cursor.Col + 1 // 1-indexed
	}

	// Check for error first
	var statusError string
	for _, block := range m.Editor.Blocks {
		if block.HasError && block.ErrorMessage != "" {
			statusError = block.ErrorMessage
			break
		}
	}

	var renderedRight string
	if statusError != "" {
		renderedRight = errorStyle.Render(" " + statusError + " ")
	} else {
		// Beginning Seperator
		sep := ""
		sepStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#313244"))
		renderedSep := sepStyle.Render(sep)

		// Build the right side: position (dark gray bg, blue text) + separator + time (mode color)
		var posStr string
		if m.CurrentFile == "" {
			posStr = " ~ "
		} else {
			posStr = fmt.Sprintf(" %d:%d ", cursorLine, cursorCol)
		}
		lineNumberStyle := m.getLineNumberStyle()
		renderedPos := lineNumberStyle.Render(posStr)

		// Time with clock icon in mode color
		timeStr := fmt.Sprintf("  %s ", m.Time.Format("15:04"))
		timeStyle := m.getTimeStyle()
		renderedTime := timeStyle.Render(timeStr)

		renderedRight = lipgloss.JoinHorizontal(lipgloss.Top, renderedSep, renderedPos, renderedTime)
	}

	wLeft := lipgloss.Width(renderedLeft)
	wCenter := lipgloss.Width(renderedCenter)
	wRight := lipgloss.Width(renderedRight)

	// Calculate gaps to truly center the filename
	// Center of screen minus half of center element width = where center element should start
	centerStart := m.width/2 - wCenter/2

	// Gap from left element to center element
	gap1Width := max(centerStart-wLeft, 0)
	// Gap from center element to right edge
	gap2Width := max(m.width-wLeft-gap1Width-wCenter-wRight, 0)

	gap1 := clearStyle.Width(gap1Width).Render(strings.Repeat(" ", gap1Width))
	gap2 := clearStyle.Width(gap2Width).Render(strings.Repeat(" ", gap2Width))

	statusLine := lipgloss.JoinHorizontal(lipgloss.Top,
		renderedLeft,
		gap1,
		renderedCenter,
		gap2,
		renderedRight,
	)

	return statusLine
}
