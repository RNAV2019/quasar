package ui

import (
	"fmt"
	"strings"

	"github.com/RNAV2019/quasar/internal/errors"
	"github.com/atotto/clipboard"
)

// executeCommand handles command mode commands.
// Returns true if the application should quit.
func (m *Model) executeCommand() bool {
	cmd := strings.TrimSpace(m.CmdInput.Value())
	cmd = strings.TrimPrefix(cmd, ":")
	cmd = strings.TrimSpace(cmd)

	switch cmd {
	case "w", "write":
		if err := m.saveFile(); err != nil {
			m.StatusMessage = fmt.Sprintf("Error: %v", err)
			errors.AddError(err.Error(), "file")
			return false
		}
		m.StatusMessage = "File saved successfully"
		m.Dirty = false
		return false
	case "wq":
		if err := m.saveFile(); err != nil {
			m.StatusMessage = fmt.Sprintf("Error: %v", err)
			errors.AddError(err.Error(), "file")
			return false
		}
		return true
	case "q", "quit":
		if m.Dirty {
			m.mode = QuitConfirm
			m.QuitConfirmDialog.Activate()
			m.CmdInput.SetValue("")
			m.CmdInput.Blur()
			return false
		}
		return true
	case "q!":
		return true
	case "new":
		if m.NotebookPath == "" {
			m.StatusMessage = "No notebook open"
			return false
		}
		m.mode = NewNote
		m.NewNoteDialog.ActivateEmpty()
		return false
	case "h", "help":
		m.mode = Help
		m.HelpDialog.Activate()
		m.CmdInput.SetValue("")
		m.CmdInput.Blur()
		return false
	case "delete", "del":
		if m.CurrentFile == "" {
			m.StatusMessage = "No file open to delete"
			return false
		}
		m.mode = DeleteConfirm
		m.DeleteConfirmDialog.Activate()
		m.CmdInput.SetValue("")
		m.CmdInput.Blur()
		return false
	default:
		m.StatusMessage = fmt.Sprintf("unknown command: %s", cmd)
		return false
	}
}

// getSlashQuery extracts the current slash command query from the editor.
func (m *Model) getSlashQuery() string {
	if m.Editor.Cursor.BlockIdx >= len(m.Editor.Blocks) {
		return ""
	}
	block := m.Editor.Blocks[m.Editor.Cursor.BlockIdx]
	if m.Editor.Cursor.LineIdx >= len(block.Lines) {
		return ""
	}
	line := block.Lines[m.Editor.Cursor.LineIdx]
	col := m.Editor.Cursor.Col

	if m.slashStartCol < 0 || col <= m.slashStartCol {
		return ""
	}

	runes := []rune(line)
	if m.slashStartCol >= len(runes) {
		return ""
	}
	if runes[m.slashStartCol] != '/' {
		return ""
	}

	return string(runes[m.slashStartCol:col])
}

// confirmAutocomplete inserts the selected slash command.
func (m *Model) confirmAutocomplete() {
	cmd := m.Autocomplete.GetSelected()
	if cmd == nil {
		m.Autocomplete.Close()
		return
	}

	block := &m.Editor.Blocks[m.Editor.Cursor.BlockIdx]
	line := block.Lines[m.Editor.Cursor.LineIdx]
	runes := []rune(line)

	newLine := string(runes[:m.slashStartCol]) + string(runes[m.Editor.Cursor.Col:])
	block.Lines[m.Editor.Cursor.LineIdx] = newLine
	m.Editor.Cursor.Col = m.slashStartCol

	if cmd.Trigger == "math" {
		m.insertMathBlock()
		m.Autocomplete.Close()
		block.IsDirty = true
		return
	}

	var cursorLine, cursorCol int
	for i, r := range cmd.Snippet {
		if r == '\n' {
			m.Editor.InsertNewLine()
		} else {
			m.Editor.InsertChar(r)
		}
		if i == cmd.CursorPos-1 {
			cursorLine = m.Editor.Cursor.LineIdx
			cursorCol = m.Editor.Cursor.Col
		}
	}

	if cmd.CursorPos > 0 {
		m.Editor.Cursor.LineIdx = cursorLine
		m.Editor.Cursor.Col = cursorCol
	}

	m.Autocomplete.Close()
	block.IsDirty = true
}

// handleYank handles the yank command (copy).
func (m *Model) handleYank() {
	if m.Editor.HasSelection() {
		text, isLineWise, ok := m.Editor.YankSelection()
		if ok {
			m.YankBuffer = text
			m.YankWasLineWise = isLineWise
			clipboard.WriteAll(text)
			m.StatusMessage = "Yanked selection"
			m.Dirty = true
		}
	} else {
		line, ok := m.Editor.YankLine()
		if ok {
			m.YankBuffer = line
			m.YankWasLineWise = true
			clipboard.WriteAll(line)
			m.StatusMessage = "Yanked line"
			m.Dirty = true
		}
	}
}

// handlePaste handles the paste command.
func (m *Model) handlePaste() {
	sysClipboard, err := clipboard.ReadAll()
	if err == nil && sysClipboard != "" {
		if sysClipboard != m.YankBuffer && m.YankBuffer == "" {
			m.YankBuffer = sysClipboard
			m.YankWasLineWise = strings.Contains(sysClipboard, "\n")
		}
	}

	if m.YankBuffer == "" {
		m.StatusMessage = "Nothing to paste"
		return
	}

	m.Editor.PasteText(m.YankBuffer)
	m.StatusMessage = "Pasted"
	m.Dirty = true
}
