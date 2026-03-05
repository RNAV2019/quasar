// Package layout assembles the standard TUI layout of file tree, separator,
// editor content, and status line.
package layout

import (
	"charm.land/lipgloss/v2"
	"github.com/RNAV2019/quasar/internal/styles"
)

// Params holds the values needed to assemble a frame.
type Params struct {
	Width         int
	ContentHeight int
	ShowFileTree  bool
	FileTreeWidth int
	FileTreeView  string
	ContentView   string
	StatusLine    string
}

// FileTreeOffset returns the horizontal space consumed by the file tree and
// its separator when visible.
func (p Params) FileTreeOffset() int {
	if p.ShowFileTree {
		return p.FileTreeWidth + 1
	}
	return 0
}

// Render assembles the standard layout: file tree | separator | content over
// status line.
func Render(p Params) string {
	if p.ShowFileTree {
		fileTreeRender := styles.ClearStyle.
			Width(p.FileTreeWidth).
			Render(p.FileTreeView)

		separator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#45475a")).
			Render("│")

		editorContent := styles.ClearStyle.
			MaxWidth(p.Width - p.FileTreeOffset()).
			PaddingLeft(2).
			Render(p.ContentView)

		mainContent := lipgloss.JoinHorizontal(lipgloss.Top, fileTreeRender, separator, editorContent)
		return lipgloss.JoinVertical(lipgloss.Top, mainContent, p.StatusLine)
	}

	renderContent := styles.ClearStyle.
		MaxWidth(p.Width).
		PaddingLeft(2).
		Render(p.ContentView)
	return lipgloss.JoinVertical(lipgloss.Top, renderContent, p.StatusLine)
}
