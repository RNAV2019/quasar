package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/RNAV2019/quasar/internal/editor"
	"github.com/RNAV2019/quasar/internal/errors"
	"github.com/RNAV2019/quasar/internal/ui/dialog"
)

// handleNormalMode processes key events in normal mode.
// Handles file tree navigation, pending operations, and standard editing commands.
func (m *Model) handleNormalMode(msg tea.KeyPressMsg) (cmds []tea.Cmd) {
	keyStr := msg.String()

	if m.pendingSpace {
		m.pendingSpace = false
		m.KeyPreview = "space+" + keyStr
		switch keyStr {
		case "f":
			m.ShowFileTree = !m.ShowFileTree
			if m.ShowFileTree {
				m.FileTree.Refresh()
				m.FileTree.Focused = true
			} else {
				m.FileTree.Focused = false
			}
			m.updateEditorSize()
		case "/":
			if m.ShowFileTree {
				m.FileTree.Focused = !m.FileTree.Focused
			}
		default:
			// Unknown space sequence, keep the preview briefly
		}
		return cmds
	}

	// Handle 'g' prefix commands
	if m.PendingOp == "g" {
		m.PendingOp = ""
		switch keyStr {
		case "h":
			m.Editor.MoveToStartOfLine()
			m.KeyPreview = "gh"
		case "l":
			m.Editor.MoveToEndOfLine()
			m.KeyPreview = "gl"
		default:
			m.KeyPreview = keyStr
		}
		return cmds
	}

	if m.ShowFileTree && m.FileTree.Focused {
		return m.handleFileTree(msg)
	}

	// Clear pending op if it doesn't match expected sequence
	if m.PendingOp != "" {
		m.PendingOp = ""
		m.KeyPreview = ""
	}

	switch keyStr {
	case "h", "left":
		m.Editor.ClearSelection()
		m.Editor.MoveCursor(0, -1)
		m.KeyPreview = keyStr
	case "j", "down":
		m.Editor.ClearSelection()
		m.Editor.MoveCursor(1, 0)
		m.KeyPreview = keyStr
	case "k", "up":
		m.Editor.ClearSelection()
		m.Editor.MoveCursor(-1, 0)
		m.KeyPreview = keyStr
	case "l", "right":
		m.Editor.ClearSelection()
		m.Editor.MoveCursor(0, 1)
		m.KeyPreview = keyStr
	case "d":
		m.Editor.DeleteChar()
		m.KeyPreview = "d"
	case "i":
		m.Editor.ClearSelection()
		m.mode = Insert
		m.KeyPreview = ""
	case "v":
		m.Editor.ClearSelection()
		m.Editor.Selection.Active = true
		m.Editor.Selection.Start = m.Editor.Cursor
		m.Editor.Selection.End = m.Editor.Cursor
		m.mode = Select
		m.KeyPreview = ""
	case "e":
		m.Editor.SelectWord()
		m.mode = Select
		m.KeyPreview = ""
	case "o":
		m.Editor.ClearSelection()
		m.Editor.MoveToEndOfLine()
		m.Editor.InsertNewLine()
		m.mode = Insert
		m.KeyPreview = ""
	case ":":
		m.mode = Command
		m.CmdInput.SetValue("")
		cmds = append(cmds, m.CmdInput.Focus())
		m.KeyPreview = ""
	case "space":
		m.pendingSpace = true
		m.KeyPreview = "space"
	case "w":
		m.Editor.ClearSelection()
		m.Editor.MoveWordForward()
		m.KeyPreview = "w"
	case "b":
		m.Editor.ClearSelection()
		m.Editor.MoveWordBackward()
		m.KeyPreview = "b"
	case "x":
		m.Editor.ClearSelection()
		lineIdx := m.Editor.Cursor.LineIdx
		blockIdx := m.Editor.Cursor.BlockIdx
		if blockIdx < len(m.Editor.Blocks) && lineIdx < len(m.Editor.Blocks[blockIdx].Lines) {
			m.Editor.Cursor.Col = 0
			m.Editor.Selection.Active = true
			m.Editor.Selection.Start = editor.Position{BlockIdx: blockIdx, LineIdx: lineIdx, Col: 0}
			m.Editor.Selection.End = editor.Position{BlockIdx: blockIdx, LineIdx: lineIdx, Col: len(m.Editor.Blocks[blockIdx].Lines[lineIdx])}
			m.Editor.Selection.WasLineWise = true
		}
		m.mode = Select
		m.KeyPreview = ""
	case "y":
		m.handleYank()
		m.KeyPreview = "y"
	case "p":
		m.handlePaste()
		m.KeyPreview = "p"
	case "g":
		m.PendingOp = "g"
		m.KeyPreview = "g"
	case "#":
		m.mode = Error
		m.ErrorDialog.Activate()
		m.KeyPreview = ""
	default:
		m.KeyPreview = keyStr
	}
	return cmds
}

// handleFileTree processes key events when the file tree is focused.
func (m *Model) handleFileTree(msg tea.KeyPressMsg) (cmds []tea.Cmd) {
	keyStr := msg.String()
	switch keyStr {
	case "j", "down":
		m.FileTree.MoveDown()
		m.KeyPreview = keyStr
	case "k", "up":
		m.FileTree.MoveUp()
		m.KeyPreview = keyStr
	case "enter":
		if m.FileTree.IsSelectedDir() {
			m.FileTree.ToggleExpand()
		} else {
			path := m.FileTree.GetSelectedPath()
			if path != "" {
				if err := m.loadFile(path); err != nil {
					m.StatusMessage = fmt.Sprintf("Error: %v", err)
					errors.AddError(err.Error(), "file")
				} else {
					m.FileTree.Focused = false
					m.ShowFileTree = false
					m.updateEditorSize()
					m.StatusMessage = ""
					cmds = append(cmds, m.processDirtyBlocks())
				}
			}
		}
		m.KeyPreview = "enter"
	case "x":
		// Delete confirmation for file/folder
		if m.FileTree.CursorIdx < len(m.FileTree.Visible) {
			node := m.FileTree.Visible[m.FileTree.CursorIdx]
			if node.Path != m.NotebookPath { // Don't allow deleting root notebook
				var message string
				if node.IsDir {
					message = fmt.Sprintf("Delete folder '%s' and all contents?", node.Name)
				} else {
					message = fmt.Sprintf("Delete file '%s'?", node.Name)
				}
				m.FileTreeDeleteDialog = dialog.NewConfirmDialog("Delete", message)
				m.mode = FileTreeDelete
				m.FileTreeDeleteDialog.Activate()
			}
		}
		m.KeyPreview = ""
	case "r":
		// Rename file/folder
		if m.FileTree.CursorIdx < len(m.FileTree.Visible) {
			node := m.FileTree.Visible[m.FileTree.CursorIdx]
			if node.Path != m.NotebookPath { // Don't allow renaming root notebook
				m.RenameDialog.ActivateWithValue(node.Name, node.IsDir)
				m.mode = FileTreeRename
			}
		}
		m.KeyPreview = ""
	case "esc":
		m.FileTree.Focused = false
		m.KeyPreview = "esc"
	case ":":
		m.mode = Command
		m.CmdInput.SetValue("")
		cmds = append(cmds, m.CmdInput.Focus())
		m.KeyPreview = ""
	case "space":
		m.pendingSpace = true
		m.KeyPreview = "space"
	default:
		m.KeyPreview = keyStr
	}
	return cmds
}
