package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"charm.land/lipgloss/v2"
)

type FileNode struct {
	Name     string
	Path     string
	IsDir    bool
	Expanded bool
	Children []FileNode
}

type FileTree struct {
	Root      FileNode
	CursorIdx int
	Visible   []FileNode
	Width     int
	Focused   bool
	NotesDir  string
}

var (
	treeDirStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa"))
	treeFileStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	treeSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#45475a")).
				Foreground(lipgloss.Color("#f5c2e7"))
	treeEmptyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	treeIndentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
)

func NewFileTree(notesDir string) *FileTree {
	ft := &FileTree{
		NotesDir:  notesDir,
		Width:     36,
		CursorIdx: 0,
		Focused:   false,
	}
	ft.Refresh()
	return ft
}

func NewFileTreeForNotebook(notebookPath, notebookName string) *FileTree {
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

func (ft *FileTree) Refresh() {
	// Preserve expanded paths before rebuilding
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

	// Restore expansion state
	for _, path := range expandedPaths {
		ft.Root = ft.updateNodeExpanded(ft.Root, path, true)
	}

	ft.flattenVisible()
}

// getExpandedPaths returns a list of all expanded directory paths
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

func (ft *FileTree) MoveUp() {
	if ft.CursorIdx > 0 {
		ft.CursorIdx--
	}
}

func (ft *FileTree) MoveDown() {
	if ft.CursorIdx < len(ft.Visible)-1 {
		ft.CursorIdx++
	}
}

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

func (ft *FileTree) GetSelectedPath() string {
	if ft.CursorIdx >= len(ft.Visible) {
		return ""
	}
	return ft.Visible[ft.CursorIdx].Path
}

func (ft *FileTree) IsSelectedDir() bool {
	if ft.CursorIdx >= len(ft.Visible) {
		return false
	}
	return ft.Visible[ft.CursorIdx].IsDir
}

func (ft *FileTree) Render(height int) string {
	var sb strings.Builder

	if len(ft.Visible) == 0 {
		emptyMsg := treeEmptyStyle.Render("  (empty)")
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

	// Build indentation with nerd font tree symbols
	var indent string
	for i := 0; i < depth; i++ {
		indent += "│ "
	}

	// Choose icon based on type and state
	var icon string
	if node.IsDir {
		if len(node.Children) == 0 {
			icon = "󰝰 " // Empty folder
		} else if node.Expanded {
			icon = "󰝰 " // Open folder
		} else {
			icon = " " // Closed folder
		}
	} else if strings.HasSuffix(node.Name, ".md") {
		icon = " " // Markdown file
	} else {
		icon = " " // Other file
	}

	prefix := indent + icon + " "
	name := node.Name

	// Calculate available width for name (account for prefix and leading space)
	// indent uses │  (2 chars per depth), icon is 2 chars, plus 1 for space, plus 1 for leading space
	prefixWidth := depth*2 + 2 + 1 + 1
	availableWidth := ft.Width - prefixWidth
	if availableWidth > 0 && ansi.StringWidth(name) > availableWidth {
		name = ansi.Truncate(name, availableWidth, "…")
	}

	var line string
	if node.IsDir {
		line = prefix + name
	} else {
		line = prefix + name
	}

	if isSelected {
		if ft.Focused {
			return treeSelectedStyle.Render(" " + line)
		}
		return treeDirStyle.Render(" " + line)
	}

	if node.IsDir {
		return treeDirStyle.Render(" " + line)
	}
	return treeFileStyle.Render(" " + line)
}

func (ft *FileTree) getNodeDepth(node FileNode) int {
	// Root folder has depth 0
	if node.Path == ft.NotesDir {
		return 0
	}

	rel, err := filepath.Rel(ft.NotesDir, node.Path)
	if err != nil {
		return 0
	}

	// Depth is number of path separators + 1
	// Root children have depth 1, nested items have depth 2, etc.
	return strings.Count(rel, string(os.PathSeparator)) + 1
}
