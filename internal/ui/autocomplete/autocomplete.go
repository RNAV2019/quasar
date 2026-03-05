// Package autocomplete provides a slash command autocomplete box for the TUI.
package autocomplete

import (
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/RNAV2019/quasar/internal/styles"
)

// SlashCommand represents a markdown shortcut.
type SlashCommand struct {
	Trigger   string // e.g., "h1", "code", "bold"
	Label     string // Display name
	Snippet   string // What to insert
	CursorPos int    // Where to place cursor after insertion (relative to start)
}

// Commands is the full list of available slash commands.
var Commands = []SlashCommand{
	{Trigger: "h1", Label: "Heading 1", Snippet: "# ", CursorPos: 2},
	{Trigger: "h2", Label: "Heading 2", Snippet: "## ", CursorPos: 3},
	{Trigger: "h3", Label: "Heading 3", Snippet: "### ", CursorPos: 4},
	{Trigger: "h4", Label: "Heading 4", Snippet: "#### ", CursorPos: 5},
	{Trigger: "h5", Label: "Heading 5", Snippet: "##### ", CursorPos: 6},
	{Trigger: "h6", Label: "Heading 6", Snippet: "###### ", CursorPos: 7},
	{Trigger: "code", Label: "Code Block", Snippet: "```\n\n```", CursorPos: 3},
	{Trigger: "inlinecode", Label: "Inline Code", Snippet: "``", CursorPos: 1},
	{Trigger: "inlinemath", Label: "Inline Math", Snippet: "$$", CursorPos: 1},
	{Trigger: "math", Label: "Math Block", Snippet: "$$\n\n$$", CursorPos: 3},
	{Trigger: "bold", Label: "Bold", Snippet: "****", CursorPos: 2},
	{Trigger: "italic", Label: "Italic", Snippet: "**", CursorPos: 1},
	{Trigger: "strikethrough", Label: "Strikethrough", Snippet: "~~~~", CursorPos: 2},
	{Trigger: "quote", Label: "Blockquote", Snippet: "> ", CursorPos: 2},
	{Trigger: "hr", Label: "Horizontal Rule", Snippet: "---\n", CursorPos: 4},
	{Trigger: "link", Label: "Link", Snippet: "[](url)", CursorPos: 1},
	{Trigger: "image", Label: "Image", Snippet: "![](url)", CursorPos: 2},
	{Trigger: "list", Label: "Bullet List", Snippet: "- ", CursorPos: 2},
	{Trigger: "numlist", Label: "Numbered List", Snippet: "1. ", CursorPos: 3},
	{Trigger: "checkbox", Label: "Checkbox", Snippet: "- [ ] ", CursorPos: 6},
	{Trigger: "table", Label: "Table", Snippet: "| Header | Header |\n| ------ | ------ |\n| Cell   | Cell   |\n", CursorPos: 2},
}

// Box manages the slash command autocomplete UI.
type Box struct {
	active      bool
	query       string
	matches     []SlashCommand
	selectedIdx int
	scrollIdx   int
	x, y        int
	width       int
}

// NewBox creates a new autocomplete box.
func NewBox() Box {
	return Box{
		active:  false,
		matches: []SlashCommand{},
	}
}

// IsActive returns whether the autocomplete is active.
func (a *Box) IsActive() bool {
	return a.active && len(a.matches) > 0
}

// Start begins a new autocomplete session.
func (a *Box) Start(query string) {
	a.query = strings.TrimPrefix(query, "/")
	a.active = true
	a.selectedIdx = 0
	a.scrollIdx = 0
	a.updateMatches()
}

// UpdateQuery updates the search query.
func (a *Box) UpdateQuery(query string) {
	a.query = strings.TrimPrefix(query, "/")
	a.selectedIdx = 0
	a.scrollIdx = 0
	a.updateMatches()
	if len(a.matches) == 0 {
		a.active = false
	}
}

func (a *Box) updateMatches() {
	a.matches = nil
	query := strings.ToLower(a.query)

	for _, cmd := range Commands {
		trigger := strings.ToLower(cmd.Trigger)
		if fuzzyMatch(query, trigger) {
			a.matches = append(a.matches, cmd)
		}
	}

	sort.Slice(a.matches, func(i, j int) bool {
		return a.matches[i].Trigger < a.matches[j].Trigger
	})
}

func fuzzyMatch(query, target string) bool {
	if query == "" {
		return true
	}
	if strings.HasPrefix(target, query) {
		return true
	}
	return strings.Contains(target, query)
}

// MoveDown moves selection down.
func (a *Box) MoveDown() {
	if len(a.matches) == 0 {
		return
	}
	a.selectedIdx = (a.selectedIdx + 1) % len(a.matches)
	if a.selectedIdx >= a.scrollIdx+5 {
		a.scrollIdx = a.selectedIdx - 4
	} else if a.selectedIdx < a.scrollIdx {
		a.scrollIdx = a.selectedIdx
	}
}

// MoveUp moves selection up.
func (a *Box) MoveUp() {
	if len(a.matches) == 0 {
		return
	}
	a.selectedIdx--
	if a.selectedIdx < 0 {
		a.selectedIdx = len(a.matches) - 1
	}
	if a.selectedIdx < a.scrollIdx {
		a.scrollIdx = a.selectedIdx
	} else if a.selectedIdx >= a.scrollIdx+5 {
		a.scrollIdx = a.selectedIdx - 4
	}
}

// GetSelected returns the selected command.
func (a *Box) GetSelected() *SlashCommand {
	if len(a.matches) == 0 || a.selectedIdx >= len(a.matches) {
		return nil
	}
	return &a.matches[a.selectedIdx]
}

// Close closes the autocomplete.
func (a *Box) Close() {
	a.active = false
	a.matches = nil
	a.query = ""
	a.scrollIdx = 0
}

// SetPosition sets the render position.
func (a *Box) SetPosition(x, y int) {
	a.x = x
	a.y = y
}

// Render renders the autocomplete box overlaid on the given view.
func (a Box) Render(view string) string {
	if !a.active || len(a.matches) == 0 {
		return view
	}

	style := styles.DefaultDialogStyle()
	selectedStyle := lipgloss.NewStyle().
		Foreground(style.KeyColor).
		Bold(true)
	normalStyle := lipgloss.NewStyle().
		Foreground(style.TextColor)
	triggerStyle := lipgloss.NewStyle().
		Foreground(style.TitleColor)

	endIdx := min(a.scrollIdx+5, len(a.matches))

	var lines []string
	for i := a.scrollIdx; i < endIdx; i++ {
		cmd := a.matches[i]
		trigger := triggerStyle.Render("/" + cmd.Trigger)
		label := cmd.Label
		if i == a.selectedIdx {
			label = selectedStyle.Render(cmd.Label)
		} else {
			label = normalStyle.Render(cmd.Label)
		}
		lines = append(lines, trigger+"  "+label)
	}

	if len(a.matches) > 5 {
		indicator := "..."
		if a.scrollIdx+5 < len(a.matches) {
			indicator = "↓"
		}
		if a.scrollIdx > 0 {
			indicator = "↑"
		}
		if a.scrollIdx > 0 && a.scrollIdx+5 < len(a.matches) {
			indicator = "↕"
		}
		lines = append(lines, styles.DimStyle.Render(indicator))
	}

	content := strings.Join(lines, "\n")

	const fixedWidth = 28
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.BorderColor).
		Padding(0, 1).
		Width(fixedWidth).
		Render(content)

	bgLayer := lipgloss.NewLayer(view)
	dialogLayer := lipgloss.NewLayer(box).X(a.x).Y(a.y).Z(2)
	compositor := lipgloss.NewCompositor(bgLayer, dialogLayer)
	return compositor.Render()
}
