// Package filetree provides a navigable file tree component for the TUI.
package filetree

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/RNAV2019/quasar/internal/styles"
)

// FileNode represents a single file or directory in the tree.
type FileNode struct {
	Name     string
	Path     string
	IsDir    bool
	Expanded bool
	Children []FileNode
}

// FileTree manages directory scanning, expansion state, and rendering.
type FileTree struct {
	Root      FileNode
	CursorIdx int
	Visible   []FileNode
	Width     int
	Focused   bool
	NotesDir  string
}

// New creates a new file tree rooted at notesDir.
func New(notesDir string) *FileTree {
	ft := &FileTree{
		NotesDir:  notesDir,
		Width:     36,
		CursorIdx: 0,
		Focused:   false,
	}
	ft.Refresh()
	return ft
}

// NewForNotebook creates a file tree for a specific notebook.
func NewForNotebook(notebookPath, notebookName string) *FileTree {
	ft := &FileTree{
		NotesDir:  notebookPath,
		Width:     36,
		CursorIdx: 0,
		Focused:   false,
	}
	ft.Root = FileNode{
		Name:     notebookName,
		Path:     notebookPath,
		IsDir:    true,
		Expanded: true,
	}
	ft.Root.Children = ft.scanDirectory(notebookPath)
	ft.flattenVisible()
	return ft
}

// Refresh rebuilds the tree while preserving expansion state.
func (ft *FileTree) Refresh() {
	expandedPaths := ft.getExpandedPaths()

	rootName := filepath.Base(ft.NotesDir)
	if rootName == "." || rootName == "" {
		rootName = "notes"
	}

	ft.Root = FileNode{
		Name:     rootName,
		Path:     ft.NotesDir,
		IsDir:    true,
		Expanded: true,
	}
	ft.Root.Children = ft.scanDirectory(ft.NotesDir)

	for _, path := range expandedPaths {
		ft.Root = ft.updateNodeExpanded(ft.Root, path, true)
	}

	ft.flattenVisible()
}

func (ft *FileTree) getExpandedPaths() []string {
	var paths []string
	ft.collectExpandedPaths(&ft.Root, &paths)
	return paths
}

func (ft *FileTree) collectExpandedPaths(node *FileNode, paths *[]string) {
	if node.IsDir && node.Expanded && node.Path != ft.NotesDir {
		*paths = append(*paths, node.Path)
	}
	for i := range node.Children {
		ft.collectExpandedPaths(&node.Children[i], paths)
	}
}

func (ft *FileTree) scanDirectory(path string) []FileNode {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	var nodes []FileNode
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := filepath.Join(path, name)
		isDir := entry.IsDir()

		node := FileNode{
			Name:  name,
			Path:  fullPath,
			IsDir: isDir,
		}

		if isDir {
			node.Expanded = false
			node.Children = ft.scanDirectory(fullPath)
		}

		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})

	return nodes
}

func (ft *FileTree) flattenVisible() {
	ft.Visible = nil
	ft.flattenNode(ft.Root, 0)
}

func (ft *FileTree) flattenNode(node FileNode, depth int) {
	ft.Visible = append(ft.Visible, node)

	if node.IsDir && node.Expanded {
		for _, child := range node.Children {
			ft.flattenNode(child, depth+1)
		}
	}
}

// MoveUp moves the cursor up one item.
func (ft *FileTree) MoveUp() {
	if ft.CursorIdx > 0 {
		ft.CursorIdx--
	}
}

// MoveDown moves the cursor down one item.
func (ft *FileTree) MoveDown() {
	if ft.CursorIdx < len(ft.Visible)-1 {
		ft.CursorIdx++
	}
}

// ToggleExpand toggles expansion of the selected directory.
func (ft *FileTree) ToggleExpand() {
	if ft.CursorIdx >= len(ft.Visible) {
		return
	}

	node := &ft.Visible[ft.CursorIdx]
	if !node.IsDir {
		return
	}

	node.Expanded = !node.Expanded

	if node.Expanded {
		ft.Root = ft.updateNodeExpanded(ft.Root, node.Path, true)
	} else {
		ft.Root = ft.updateNodeExpanded(ft.Root, node.Path, false)
	}

	ft.flattenVisible()
	if ft.CursorIdx >= len(ft.Visible) {
		ft.CursorIdx = len(ft.Visible) - 1
	}
}

func (ft *FileTree) updateNodeExpanded(root FileNode, targetPath string, expanded bool) FileNode {
	if root.Path == targetPath {
		root.Expanded = expanded
		return root
	}

	for i := range root.Children {
		root.Children[i] = ft.updateNodeExpanded(root.Children[i], targetPath, expanded)
	}

	return root
}

// GetSelectedPath returns the path of the currently selected item.
func (ft *FileTree) GetSelectedPath() string {
	if ft.CursorIdx >= len(ft.Visible) {
		return ""
	}
	return ft.Visible[ft.CursorIdx].Path
}

// IsSelectedDir returns whether the currently selected item is a directory.
func (ft *FileTree) IsSelectedDir() bool {
	if ft.CursorIdx >= len(ft.Visible) {
		return false
	}
	return ft.Visible[ft.CursorIdx].IsDir
}

// Render renders the file tree to fit the given height.
func (ft *FileTree) Render(height int) string {
	var sb strings.Builder

	if len(ft.Visible) == 0 {
		emptyMsg := styles.TreeEmptyStyle.Render("  (empty)")
		sb.WriteString(emptyMsg)
		for i := 1; i < height; i++ {
			sb.WriteString("\n")
		}
		return sb.String()
	}

	for i, node := range ft.Visible {
		line := ft.renderNode(node, i == ft.CursorIdx)
		sb.WriteString(line)
		if i < len(ft.Visible)-1 {
			sb.WriteString("\n")
		}
	}

	for i := len(ft.Visible); i < height; i++ {
		sb.WriteString("\n")
	}

	return sb.String()
}

func (ft *FileTree) renderNode(node FileNode, isSelected bool) string {
	depth := ft.getNodeDepth(node)

	var indent strings.Builder
	for range depth {
		indent.WriteString("│ ")
	}

	var icon string
	if node.IsDir {
		if len(node.Children) == 0 {
			icon = "󰝰 "
		} else if node.Expanded {
			icon = "󰝰 "
		} else {
			icon = " "
		}
	} else if strings.HasSuffix(node.Name, ".md") {
		icon = " "
	} else {
		icon = " "
	}

	prefix := indent.String() + icon + " "
	name := node.Name

	prefixWidth := depth*2 + 2 + 1 + 1
	availableWidth := ft.Width - prefixWidth
	if availableWidth > 0 && ansi.StringWidth(name) > availableWidth {
		name = ansi.Truncate(name, availableWidth, "…")
	}

	line := prefix + name

	if isSelected {
		if ft.Focused {
			return styles.TreeSelectedStyle.Render(" " + line)
		}
		return styles.TreeDirStyle.Render(" " + line)
	}

	if node.IsDir {
		return styles.TreeDirStyle.Render(" " + line)
	}
	return styles.TreeFileStyle.Render(" " + line)
}

func (ft *FileTree) getNodeDepth(node FileNode) int {
	if node.Path == ft.NotesDir {
		return 0
	}

	rel, err := filepath.Rel(ft.NotesDir, node.Path)
	if err != nil {
		return 0
	}

	return strings.Count(rel, string(os.PathSeparator)) + 1
}
