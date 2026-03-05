package ui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/RNAV2019/quasar/internal/editor"
	"github.com/RNAV2019/quasar/internal/errors"
	"github.com/RNAV2019/quasar/internal/latex"
	"github.com/atotto/clipboard"
)

// updateParsedDoc re-parses the document when blocks change.
func (m *Model) updateParsedDoc() {
	m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
}

// updateEditorSize adjusts the editor size based on file tree visibility.
func (m *Model) updateEditorSize() {
	widthAdjust := 0
	if m.ShowFileTree {
		widthAdjust = m.FileTree.Width + 1
	}
	m.Editor.SetSize(m.width-widthAdjust, m.height-1)
}

// calculateContentWidth returns the available width for content in terminal columns.
func (m *Model) calculateContentWidth() int {
	totalLines := 0
	for _, block := range m.Editor.Blocks {
		totalLines += len(block.Lines)
	}
	gutterWidth := len(fmt.Sprint(totalLines))

	fileTreeOffset := 0
	if m.ShowFileTree {
		fileTreeOffset = m.FileTree.Width + 1
	}

	return max(m.width-5-gutterWidth-fileTreeOffset, 40)
}

// Update handles all incoming messages and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		oldCursor := m.Editor.Cursor
		oldMode := m.mode

		if m.mode == Insert {
			cmds = append(cmds, m.handleInsertMode(msg)...)
		} else if m.mode == Command {
			var quit bool
			cmds, quit = m.handleCommandMode(msg)
			if quit {
				return m, tea.Quit
			}
		} else if m.mode == NewNote {
			m.handleNewNoteMode(msg)
		} else if m.mode == Help {
			m.handleHelpMode(msg)
		} else if m.mode == Error {
			m.handleErrorMode(msg)
		} else if m.mode == DeleteConfirm {
			m.handleDeleteConfirmMode(msg)
		} else if m.mode == QuitConfirm {
			if m.handleQuitConfirmMode(msg) {
				return m, tea.Quit
			}
		} else if m.mode == FileTreeDelete {
			m.handleFileTreeDeleteMode(msg)
		} else if m.mode == FileTreeRename {
			m.handleFileTreeRenameMode(msg)
		} else if m.mode == Select {
			m.handleSelectMode(msg)
		} else {
			cmds = append(cmds, m.handleNormalMode(msg)...)
		}

		modeChanged := oldMode != m.mode
		cursorMoved := oldCursor != m.Editor.Cursor

		if modeChanged || m.mode == Insert {
			if m.Editor.Cursor.BlockIdx < len(m.Editor.Blocks) {
				m.Editor.Blocks[m.Editor.Cursor.BlockIdx].IsDirty = true
			}
			if oldCursor.BlockIdx != m.Editor.Cursor.BlockIdx && oldCursor.BlockIdx < len(m.Editor.Blocks) {
				m.Editor.Blocks[oldCursor.BlockIdx].IsDirty = true
			}
		}

		if modeChanged && !cursorMoved {
			cmds = append(cmds, m.processDirtyBlocks())
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateEditorSize()

	case BlockProcessedMsg:
		if msg.Generation != m.fileGeneration {
			if msg.ImageID != 0 {
				latex.DeleteImage(msg.ImageID)
			}
			break
		}
		m.PendingRenders--
		if msg.Source != "" {
			m.CompiledMath = append(m.CompiledMath, msg.Source)
		}
		if msg.BlockIdx < len(m.Editor.Blocks) {
			block := &m.Editor.Blocks[msg.BlockIdx]
			block.IsLoading = false
			if block.ImageID != 0 && block.ImageID != msg.ImageID {
				latex.DeleteImage(block.ImageID)
			}
			block.ImageID = msg.ImageID
			block.ImageCols = msg.ImageCols
			block.ImageHeight = msg.ImageHeight
			block.HasError = msg.Error != nil
			if msg.Error != nil {
				block.ErrorMessage = msg.Error.Error()
				errors.AddError(msg.Error.Error(), "latex")
			} else {
				block.ErrorMessage = ""
			}
		}
		if m.PendingRenders == 0 && m.DocumentLoading {
			m.DocumentLoading = false
		}

	case InlineMathProcessedMsg:
		if msg.Generation != m.fileGeneration {
			if msg.ImageID != 0 {
				latex.DeleteImage(msg.ImageID)
			}
			break
		}
		m.PendingRenders--
		if msg.Error != nil {
			errors.AddError(msg.Error.Error(), "latex")
		} else if msg.BlockIdx < len(m.Editor.Blocks) {
			key := fmt.Sprintf("%d-%d-%d", msg.BlockIdx, msg.LineIdx, msg.StartCol)
			m.InlineRenders[key] = InlineMathRender{
				ImageID:     msg.ImageID,
				ImageCols:   msg.ImageCols,
				ImageHeight: msg.ImageHeight,
				Length:      msg.ImageCols,
				TextLength:  msg.EndCol - msg.StartCol,
			}
		}
		if m.PendingRenders == 0 && m.DocumentLoading {
			m.DocumentLoading = false
		}

	case TickMsg:
		m.Time = time.Time(msg)
		if m.PendingRenders == 0 {
			for _, b := range m.Editor.Blocks {
				if b.IsDirty {
					m.updateParsedDoc()
					cmds = append(cmds, m.processDirtyBlocks())
					break
				}
			}
		}
		return m, tea.Batch(doTick(), tea.Batch(cmds...))

	case tea.MouseClickMsg:
		if url := m.getLinkAtPosition(msg.X, msg.Y); url != "" {
			if err := clipboard.WriteAll(url); err == nil {
				m.StatusMessage = fmt.Sprintf("Copied: %s", url)
			} else {
				m.StatusMessage = fmt.Sprintf("Failed to copy: %v", err)
			}
		}
	}

	return m, tea.Batch(cmds...)
}
