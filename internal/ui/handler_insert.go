package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/RNAV2019/quasar/internal/editor"
)

// handleInsertMode processes key events in insert mode.
// Returns tea.Quit if the app should exit, otherwise nil.
func (m *Model) handleInsertMode(msg tea.KeyPressMsg) (cmds []tea.Cmd) {
	switch msg.String() {
	case "left":
		m.Editor.MoveCursor(0, -1)
		m.Autocomplete.Close()
	case "down":
		m.Editor.MoveCursor(1, 0)
		m.Autocomplete.Close()
	case "up":
		m.Editor.MoveCursor(-1, 0)
		m.Autocomplete.Close()
	case "right":
		m.Editor.MoveCursor(0, 1)
		m.Autocomplete.Close()
	case "backspace", "delete":
		m.Editor.Backspace()
		m.Dirty = true
		if m.Autocomplete.IsActive() {
			query := m.getSlashQuery()
			if query == "" {
				m.Autocomplete.Close()
			} else {
				m.Autocomplete.UpdateQuery(query)
			}
		}
	case "enter":
		if m.Autocomplete.IsActive() {
			m.confirmAutocomplete()
		} else {
			m.Editor.InsertNewLine()
			m.Dirty = true
		}
	case "tab":
		if m.Autocomplete.IsActive() {
			m.Autocomplete.MoveDown()
		} else {
			m.Editor.InsertChar('\t')
		}
	case "shift+tab":
		if m.Autocomplete.IsActive() {
			m.Autocomplete.MoveUp()
		}
	case "space":
		m.Editor.InsertChar(' ')
		m.Autocomplete.Close()
	case "esc":
		m.mode = Normal
		m.Autocomplete.Close()
		cmds = append(cmds, m.processDirtyBlocks())
	default:
		if msg.Text != "" {
			for _, r := range msg.Text {
				m.Editor.InsertChar(r)
			}
			m.Dirty = true
			if msg.Text == "/" {
				m.slashStartCol = m.Editor.Cursor.Col - 1
				inMath := m.Editor.Cursor.BlockIdx < len(m.Editor.Blocks) &&
					m.Editor.Blocks[m.Editor.Cursor.BlockIdx].Type == editor.MathBlock
				m.Autocomplete.SetMathMode(inMath)
				m.Autocomplete.Start("/")
			} else if m.Autocomplete.IsActive() {
				query := m.getSlashQuery()
				if query == "" {
					m.Autocomplete.Close()
				} else {
					m.Autocomplete.UpdateQuery(query)
				}
			}
		}
	}
	return cmds
}
