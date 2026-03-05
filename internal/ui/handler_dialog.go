package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/RNAV2019/quasar/internal/editor"
	"github.com/RNAV2019/quasar/internal/errors"
	"github.com/atotto/clipboard"
)

// handleNewNoteMode processes key events in new note dialog mode.
func (m *Model) handleNewNoteMode(msg tea.KeyPressMsg) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.mode = Normal
		m.NewNoteDialog.Deactivate()
		m.StatusMessage = ""
		m.KeyPreview = ""
	case "enter":
		noteName := m.NewNoteDialog.Value()
		if noteName != "" {
			if err := m.createNewNote(noteName); err != nil {
				m.StatusMessage = fmt.Sprintf("Error: %v", err)
			} else {
				m.StatusMessage = ""
			}
		}
		m.mode = Normal
		m.NewNoteDialog.Deactivate()
		m.KeyPreview = ""
	default:
		m.NewNoteDialog.Update(msg)
	}
}

// handleHelpMode processes key events in help dialog mode.
func (m *Model) handleHelpMode(_ tea.KeyPressMsg) {
	m.mode = Normal
	m.HelpDialog.Deactivate()
	m.KeyPreview = ""
}

// handleErrorMode processes key events in error dialog mode.
func (m *Model) handleErrorMode(msg tea.KeyPressMsg) {
	keyStr := msg.String()
	switch keyStr {
	case ",":
		// Clear all errors
		errors.ClearErrors()
		m.mode = Normal
		m.ErrorDialog.Deactivate()
		m.KeyPreview = ""
	case "y":
		// Copy all errors to clipboard
		errs := errors.GetErrors()
		if len(errs) > 0 {
			var errTexts []string
			for _, err := range errs {
				errTexts = append(errTexts, "["+err.Source+"] "+err.Message)
			}
			clipboard.WriteAll(strings.Join(errTexts, "\n"))
			m.StatusMessage = "Copied errors to clipboard"
		}
	case "esc", "q":
		m.mode = Normal
		m.ErrorDialog.Deactivate()
		m.KeyPreview = ""
	default:
		// Any other key closes the dialog
		m.mode = Normal
		m.ErrorDialog.Deactivate()
		m.KeyPreview = ""
	}
}

// handleDeleteConfirmMode processes key events in delete confirmation dialog mode.
func (m *Model) handleDeleteConfirmMode(msg tea.KeyPressMsg) {
	switch msg.String() {
	case "left", "h":
		m.DeleteConfirmDialog.SelectYes()
	case "right", "l":
		m.DeleteConfirmDialog.SelectNo()
	case "enter":
		if m.DeleteConfirmDialog.IsYesSelected() {
			if m.CurrentFile != "" {
				if err := os.Remove(m.CurrentFile); err != nil {
					m.StatusMessage = fmt.Sprintf("Error deleting file: %v", err)
				} else {
					m.StatusMessage = "Note deleted"
					m.CurrentFile = ""
					m.Editor = editor.NewModel()
					m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
					m.FileTree.Refresh()
				}
			}
		}
		m.mode = Normal
		m.DeleteConfirmDialog.Deactivate()
		m.KeyPreview = ""
	case "esc":
		m.mode = Normal
		m.DeleteConfirmDialog.Deactivate()
		m.KeyPreview = ""
	}
}

// handleQuitConfirmMode processes key events in quit confirmation dialog mode.
// Returns true if the app should quit.
func (m *Model) handleQuitConfirmMode(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "left", "h":
		m.QuitConfirmDialog.SelectYes()
	case "right", "l":
		m.QuitConfirmDialog.SelectNo()
	case "enter":
		if m.QuitConfirmDialog.IsYesSelected() {
			return true
		}
		m.mode = Normal
		m.QuitConfirmDialog.Deactivate()
		m.KeyPreview = ""
	case "esc":
		m.mode = Normal
		m.QuitConfirmDialog.Deactivate()
		m.KeyPreview = ""
	}
	return false
}

// handleFileTreeDeleteMode processes key events in file tree delete confirmation mode.
func (m *Model) handleFileTreeDeleteMode(msg tea.KeyPressMsg) {
	switch msg.String() {
	case "left", "h":
		m.FileTreeDeleteDialog.SelectYes()
	case "right", "l":
		m.FileTreeDeleteDialog.SelectNo()
	case "enter":
		if m.FileTreeDeleteDialog.IsYesSelected() {
			m.deleteFileTreeItem()
		}
		m.mode = Normal
		m.FileTreeDeleteDialog.Deactivate()
		m.KeyPreview = ""
	case "esc":
		m.mode = Normal
		m.FileTreeDeleteDialog.Deactivate()
		m.KeyPreview = ""
	}
}

// handleFileTreeRenameMode processes key events in file tree rename mode.
func (m *Model) handleFileTreeRenameMode(msg tea.KeyPressMsg) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.mode = Normal
		m.RenameDialog.Deactivate()
		m.StatusMessage = ""
		m.KeyPreview = ""
	case "enter":
		newName := m.RenameDialog.Value()
		if newName != "" {
			if err := m.renameFileTreeItem(newName); err != nil {
				m.StatusMessage = fmt.Sprintf("Error: %v", err)
			} else {
				m.StatusMessage = ""
			}
		}
		m.mode = Normal
		m.RenameDialog.Deactivate()
		m.KeyPreview = ""
	default:
		m.RenameDialog.Update(msg)
	}
}

// deleteFileTreeItem deletes the currently selected file or folder.
func (m *Model) deleteFileTreeItem() {
	if m.FileTree.CursorIdx >= len(m.FileTree.Visible) {
		return
	}

	node := m.FileTree.Visible[m.FileTree.CursorIdx]
	path := node.Path

	// Check if trying to delete the currently open file
	if m.CurrentFile == path || (node.IsDir && strings.HasPrefix(m.CurrentFile, path+string(os.PathSeparator))) {
		// Close the current file
		m.CurrentFile = ""
		m.Editor = editor.NewModel()
		m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
		m.OriginalMetadata = nil
	}

	var err error
	if node.IsDir {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}

	if err != nil {
		m.StatusMessage = fmt.Sprintf("Error deleting: %v", err)
		return
	}

	m.StatusMessage = fmt.Sprintf("Deleted %s", node.Name)
	m.FileTree.Refresh()

	// Adjust cursor if needed
	if m.FileTree.CursorIdx >= len(m.FileTree.Visible) {
		m.FileTree.CursorIdx = len(m.FileTree.Visible) - 1
	}
}

// renameFileTreeItem renames the currently selected file or folder and updates front matter.
func (m *Model) renameFileTreeItem(newName string) error {
	if m.FileTree.CursorIdx >= len(m.FileTree.Visible) {
		return fmt.Errorf("no item selected")
	}

	node := m.FileTree.Visible[m.FileTree.CursorIdx]
	oldPath := node.Path

	// Determine new path
	var newPath string
	if node.IsDir {
		// For directories, rename directly
		parentDir := filepath.Dir(oldPath)
		newPath = filepath.Join(parentDir, newName)
	} else {
		// For files, ensure .md extension
		if !strings.HasSuffix(newName, ".md") {
			newName = newName + ".md"
		}
		parentDir := filepath.Dir(oldPath)
		newPath = filepath.Join(parentDir, newName)
	}

	// Check if new name already exists
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("a file/folder with this name already exists")
	}

	// For files, we need to update front matter and potentially move
	if !node.IsDir {
		return m.renameMarkdownFile(oldPath, newPath, newName)
	}

	// For directories (tags), we need to update the tag in all contained files
	if err := m.renameTagFolder(oldPath, newName); err != nil {
		return err
	}

	// Rename the directory
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	// Update current file path and reload editor if it was moved
	if m.CurrentFile != "" && strings.HasPrefix(m.CurrentFile, oldPath+string(os.PathSeparator)) {
		newCurrentFile := strings.Replace(m.CurrentFile, oldPath, newPath, 1)
		m.CurrentFile = newCurrentFile
		// Reload the editor with the updated content
		if model, err := editor.LoadFromFile(newCurrentFile); err == nil {
			m.Editor = *model
			m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
			for i := range m.Editor.Blocks {
				m.Editor.Blocks[i].IsDirty = true
			}
			// Update original metadata
			var allLines []string
			for _, block := range m.Editor.Blocks {
				allLines = append(allLines, block.Lines...)
			}
			m.OriginalMetadata, _, _ = editor.ExtractFrontMatter(allLines)
		}
	}

	m.FileTree.Refresh()
	m.StatusMessage = fmt.Sprintf("Renamed to %s", filepath.Base(newPath))
	return nil
}

// renameMarkdownFile renames a single markdown file and updates its title in front matter.
func (m *Model) renameMarkdownFile(oldPath, newPath, newName string) error {
	// Read the file content
	content, err := os.ReadFile(oldPath)
	if err != nil {
		return err
	}

	// Parse the file to update front matter
	lines := strings.Split(string(content), "\n")
	metadata, remainingLines, err := editor.ExtractFrontMatter(lines)
	if err != nil && err.Error() != "unclosed front matter block" {
		// If no front matter, just rename
		if err := os.Rename(oldPath, newPath); err != nil {
			return err
		}
		// Update current file path if this was the current file
		if m.CurrentFile == oldPath {
			m.CurrentFile = newPath
		}
		m.FileTree.Refresh()
		return nil
	}

	// Strip .md extension for title
	title := newName
	if strings.HasSuffix(title, ".md") {
		title = title[:len(title)-3]
	}

	// Update the title in front matter
	if metadata == nil {
		metadata = &editor.Metadata{}
	}
	metadata.Title = title

	// Rebuild the file content with updated front matter
	var newLines []string
	newLines = append(newLines, "---")
	if metadata.Title != "" {
		newLines = append(newLines, fmt.Sprintf("title: %s", metadata.Title))
	}
	if metadata.Date != "" {
		newLines = append(newLines, fmt.Sprintf("date: %s", metadata.Date))
	}
	if metadata.Tag != "" {
		newLines = append(newLines, fmt.Sprintf("tag: %s", metadata.Tag))
	}
	newLines = append(newLines, "---")
	newLines = append(newLines, remainingLines...)

	// Write to new path
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(newPath, []byte(newContent), 0644); err != nil {
		return err
	}

	// Remove old file
	if err := os.Remove(oldPath); err != nil {
		return err
	}

	// Update current file path and reload editor if this was the current file
	if m.CurrentFile == oldPath {
		m.CurrentFile = newPath
		m.OriginalMetadata = metadata
		// Reload the editor with the new content
		if model, err := editor.LoadFromFile(newPath); err == nil {
			m.Editor = *model
			m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
			for i := range m.Editor.Blocks {
				m.Editor.Blocks[i].IsDirty = true
			}
		}
	}

	m.FileTree.Refresh()
	return nil
}

// renameTagFolder renames a tag folder and updates all contained files.
func (m *Model) renameTagFolder(oldPath, newTagName string) error {
	// Get all markdown files in the directory
	var files []string
	err := filepath.Walk(oldPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Update each file's tag in front matter
	for _, file := range files {
		if err := m.updateFileTag(file, newTagName); err != nil {
			return err
		}
	}

	return nil
}

// updateFileTag updates the tag field in a file's front matter.
func (m *Model) updateFileTag(filePath, newTag string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	metadata, remainingLines, err := editor.ExtractFrontMatter(lines)
	if err != nil && err.Error() != "unclosed front matter block" {
		// No front matter, nothing to update
		return nil
	}

	if metadata == nil {
		metadata = &editor.Metadata{}
	}
	metadata.Tag = newTag

	// Rebuild file content
	var newLines []string
	newLines = append(newLines, "---")
	if metadata.Title != "" {
		newLines = append(newLines, fmt.Sprintf("title: %s", metadata.Title))
	}
	if metadata.Date != "" {
		newLines = append(newLines, fmt.Sprintf("date: %s", metadata.Date))
	}
	if metadata.Tag != "" {
		newLines = append(newLines, fmt.Sprintf("tag: %s", metadata.Tag))
	}
	newLines = append(newLines, "---")
	newLines = append(newLines, remainingLines...)

	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(filePath, []byte(newContent), 0644)
}
