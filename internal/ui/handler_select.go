package ui

import (
	"strconv"

	tea "charm.land/bubbletea/v2"
)

// handleSelectMode processes key events in select mode.
// Navigation keys extend the selection.
func (m *Model) handleSelectMode(msg tea.KeyPressMsg) {
	keyStr := msg.String()

	// Handle 'g' prefix commands
	if m.PendingOp == "g" {
		m.PendingOp = ""
		switch keyStr {
		case "h":
			m.Editor.MoveToStartOfLine()
			m.Editor.ExtendSelection()
			m.KeyPreview = "gh"
		case "l":
			m.Editor.MoveToEndOfLine()
			m.Editor.ExtendSelection()
			m.KeyPreview = "gl"
		default:
			m.KeyPreview = keyStr
		}
		return
	}

	// Count prefix accumulation: digits 1-9 start a count, 0 continues if already started
	if keyStr >= "1" && keyStr <= "9" || (keyStr == "0" && m.CountPrefix != "") {
		m.CountPrefix += keyStr
		m.KeyPreview = m.CountPrefix
		return
	}

	// Parse count and reset
	count := 1
	if m.CountPrefix != "" {
		if n, err := strconv.Atoi(m.CountPrefix); err == nil && n > 0 {
			count = n
		}
		m.CountPrefix = ""
	}

	switch keyStr {
	case "esc":
		m.mode = Normal
		m.Editor.ClearSelection()
		m.KeyPreview = ""
	case "y":
		m.handleYank()
		m.mode = Normal
		m.Editor.ClearSelection()
		m.KeyPreview = "y"
	case "d":
		m.Undo.Save(&m.Editor)
		deleted, _ := m.Editor.DeleteSelection()
		m.YankBuffer = deleted
		m.YankWasLineWise = false
		m.mode = Normal
		m.Dirty = true
		m.StatusMessage = "Deleted"
		m.KeyPreview = "d"
	case "h", "left":
		for range count {
			m.Editor.MoveCursor(0, -1)
		}
		m.Editor.ExtendSelection()
		m.KeyPreview = keyStr
	case "j", "down":
		for range count {
			m.Editor.MoveCursor(1, 0)
		}
		m.Editor.ExtendSelection()
		m.KeyPreview = keyStr
	case "k", "up":
		for range count {
			m.Editor.MoveCursor(-1, 0)
		}
		m.Editor.ExtendSelection()
		m.KeyPreview = keyStr
	case "l", "right":
		for range count {
			m.Editor.MoveCursor(0, 1)
		}
		m.Editor.ExtendSelection()
		m.KeyPreview = keyStr
	case "w":
		for range count {
			m.Editor.MoveWordForward()
		}
		m.Editor.ExtendSelection()
		m.KeyPreview = "w"
	case "b":
		for range count {
			m.Editor.MoveWordBackward()
		}
		m.Editor.ExtendSelection()
		m.KeyPreview = "b"
	case "e":
		for range count {
			m.Editor.MoveToEndOfWord()
		}
		m.Editor.ExtendSelection()
		m.KeyPreview = "e"
	case "g":
		m.PendingOp = "g"
		m.KeyPreview = "g"
	case "x":
		// Extend selection to cover the entire current line
		m.Editor.Selection.Start.LineIdx = m.Editor.Cursor.LineIdx
		m.Editor.Selection.Start.Col = 0
		m.Editor.Selection.End.LineIdx = m.Editor.Cursor.LineIdx
		line := m.Editor.Blocks[m.Editor.Cursor.BlockIdx].Lines[m.Editor.Cursor.LineIdx]
		m.Editor.Selection.End.Col = len([]rune(line))
		m.Editor.Cursor.Col = m.Editor.Selection.End.Col
		m.KeyPreview = "x"
	default:
		m.KeyPreview = keyStr
	}
}
