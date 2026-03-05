package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RNAV2019/quasar/internal/editor"
	"github.com/RNAV2019/quasar/internal/latex"
	"github.com/RNAV2019/quasar/internal/notebook"
)

// loadFile loads a file into the editor.
func (m *Model) loadFile(path string) error {
	m.fileGeneration++
	m.PendingRenders = 0

	for _, block := range m.Editor.Blocks {
		if block.ImageID != 0 {
			latex.DeleteImage(block.ImageID)
		}
	}
	for _, render := range m.InlineRenders {
		if render.ImageID != 0 {
			latex.DeleteImage(render.ImageID)
		}
	}
	m.InlineRenders = make(map[string]InlineMathRender)

	model, err := editor.LoadFromFile(path)
	if err != nil {
		return err
	}
	m.Editor = *model
	m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
	for i := range m.Editor.Blocks {
		m.Editor.Blocks[i].IsDirty = true
	}
	m.updateEditorSize()
	m.CurrentFile = path

	hasMath := false
	for _, block := range m.Editor.Blocks {
		if block.Type == editor.MathBlock {
			hasMath = true
			break
		}
		for _, line := range block.Lines {
			if inlineMathRe.MatchString(line) {
				hasMath = true
				break
			}
		}
		if hasMath {
			break
		}
	}
	m.DocumentLoading = hasMath

	var allLines []string
	for _, block := range m.Editor.Blocks {
		allLines = append(allLines, block.Lines...)
	}
	m.OriginalMetadata, _, _ = editor.ExtractFrontMatter(allLines)
	return nil
}

// createNewNote creates a new note in the current notebook.
func (m *Model) createNewNote(name string) error {
	spec := notebook.ParseNoteName(name, m.NotebookPath)
	if err := spec.Create(); err != nil {
		return err
	}
	m.FileTree.Refresh()
	if err := m.loadFile(spec.Path); err != nil {
		return err
	}
	return nil
}

// saveFile saves the current file, handling title/tag changes by renaming if needed.
func (m *Model) saveFile() error {
	var allLines []string
	for _, block := range m.Editor.Blocks {
		allLines = append(allLines, block.Lines...)
	}
	currentMetadata, _, err := editor.ExtractFrontMatter(allLines)
	if err != nil {
		return fmt.Errorf("failed to parse front matter: %w", err)
	}

	if currentMetadata.Title == "" {
		currentMetadata.Title = m.Editor.ExtractTitle()
		if currentMetadata.Title == "" {
			return fmt.Errorf("no title found: add a title field to the YAML front matter or start with a heading")
		}
	}

	baseDir := m.NotebookPath
	if baseDir == "" {
		baseDir = m.Config.NotesDir
	}

	needsRename := false
	if m.OriginalMetadata != nil && m.CurrentFile != "" {
		if m.OriginalMetadata.Title != currentMetadata.Title || m.OriginalMetadata.Tag != currentMetadata.Tag {
			needsRename = true
		}
	}

	var targetPath string
	if needsRename {
		filename := editor.SanitizeFilename(currentMetadata.Title) + ".md"
		if currentMetadata.Tag != "" {
			targetPath = filepath.Join(baseDir, currentMetadata.Tag, filename)
		} else {
			targetPath = filepath.Join(baseDir, filename)
		}
	} else if m.CurrentFile != "" {
		targetPath = m.CurrentFile
	} else {
		filename := editor.SanitizeFilename(currentMetadata.Title) + ".md"
		if currentMetadata.Tag != "" {
			targetPath = filepath.Join(baseDir, currentMetadata.Tag, filename)
		} else {
			targetPath = filepath.Join(baseDir, filename)
		}
	}

	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	var frontMatterLines []string
	if len(allLines) > 0 && allLines[0] == "---" {
		for i := 1; i < len(allLines); i++ {
			if allLines[i] == "---" {
				frontMatterLines = allLines[:i+1]
				break
			}
		}
	}

	content, err := m.Editor.ToMarkdownContent(frontMatterLines)
	if err != nil {
		return fmt.Errorf("failed to generate markdown content: %w", err)
	}

	fileContent := strings.Join(content, "\n")
	if err := os.WriteFile(targetPath, []byte(fileContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if needsRename && m.CurrentFile != "" && m.CurrentFile != targetPath {
		if err := os.Remove(m.CurrentFile); err != nil {
			m.StatusMessage = fmt.Sprintf("Saved but failed to delete old file: %v", err)
		}
		m.FileTree.Refresh()
	}

	m.CurrentFile = targetPath
	m.OriginalMetadata = currentMetadata

	return nil
}
