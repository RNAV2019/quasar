package ui

import (
	tea "charm.land/bubbletea/v2"
)

// handleCommandMode processes key events in command mode.
// Returns tea.Quit if the app should exit, otherwise nil.
func (m *Model) handleCommandMode(msg tea.KeyPressMsg) (cmds []tea.Cmd, quit bool) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.mode = Normal
		m.CmdInput.SetValue("")
		m.CmdInput.Blur()
		m.StatusMessage = ""
	case "enter":
		if m.executeCommand() {
			return nil, true
		}
		if m.mode != NewNote && m.mode != Help && m.mode != DeleteConfirm && m.mode != QuitConfirm {
			m.mode = Normal
			m.CmdInput.SetValue("")
			m.CmdInput.Blur()
		}
	default:
		var cmd tea.Cmd
		m.CmdInput, cmd = m.CmdInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	return cmds, false
}
