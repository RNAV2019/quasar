package ui

import (
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

// SlashCommand represents a markdown shortcut
type SlashCommand struct {
	Trigger   string // e.g., "h1", "code", "bold"
	Label     string // Display name
	Snippet   string // What to insert
	CursorPos int    // Where to place cursor after insertion (relative to start)
}

var slashCommands = []SlashCommand{
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

// AutocompleteBox manages the slash command autocomplete UI
type AutocompleteBox struct {
	active      bool
	query       string
	matches     []SlashCommand
	selectedIdx int
	scrollIdx   int // Index of first visible item
	x, y        int
	width       int
}

// NewAutocompleteBox creates a new autocomplete box
func NewAutocompleteBox() AutocompleteBox {
	return AutocompleteBox{
		active:  false,
		matches: []SlashCommand{},
	}
}

// IsActive returns whether the autocomplete is active
func (a *AutocompleteBox) IsActive() bool {
	return a.active && len(a.matches) > 0
}

// Start begins a new autocomplete session
func (a *AutocompleteBox) Start(query string) {
	a.query = strings.TrimPrefix(query, "/")
	a.active = true
	a.selectedIdx = 0
	a.scrollIdx = 0
	a.updateMatches()
}

// UpdateQuery updates the search query
func (a *AutocompleteBox) UpdateQuery(query string) {
	a.query = strings.TrimPrefix(query, "/")
	a.selectedIdx = 0
	a.scrollIdx = 0
	a.updateMatches()
	if len(a.matches) == 0 {
		a.active = false
	}
}

// updateMatches filters commands by fuzzy matching
func (a *AutocompleteBox) updateMatches() {
	a.matches = nil
	query := strings.ToLower(a.query)

	for _, cmd := range slashCommands {
		trigger := strings.ToLower(cmd.Trigger)
		if fuzzyMatch(query, trigger) {
			a.matches = append(a.matches, cmd)
		}
	}

	// Sort alphabetically by trigger
	sort.Slice(a.matches, func(i, j int) bool {
		return a.matches[i].Trigger < a.matches[j].Trigger
	})
}

// fuzzyMatch performs simple fuzzy matching
func fuzzyMatch(query, target string) bool {
	if query == "" {
		return true
	}
	if strings.HasPrefix(target, query) {
		return true
	}

	// Simple substring match
	return strings.Contains(target, query)
}

// MoveDown moves selection down
func (a *AutocompleteBox) MoveDown() {
	if len(a.matches) == 0 {
		return
	}
	a.selectedIdx = (a.selectedIdx + 1) % len(a.matches)
	// Adjust scroll to keep selection visible
	if a.selectedIdx >= a.scrollIdx+5 {
		a.scrollIdx = a.selectedIdx - 4
	} else if a.selectedIdx < a.scrollIdx {
		a.scrollIdx = a.selectedIdx
	}
}

// MoveUp moves selection up
func (a *AutocompleteBox) MoveUp() {
	if len(a.matches) == 0 {
		return
	}
	a.selectedIdx--
	if a.selectedIdx < 0 {
		a.selectedIdx = len(a.matches) - 1
	}
	// Adjust scroll to keep selection visible
	if a.selectedIdx < a.scrollIdx {
		a.scrollIdx = a.selectedIdx
	} else if a.selectedIdx >= a.scrollIdx+5 {
		a.scrollIdx = a.selectedIdx - 4
	}
}

// GetSelected returns the selected command
func (a *AutocompleteBox) GetSelected() *SlashCommand {
	if len(a.matches) == 0 || a.selectedIdx >= len(a.matches) {
		return nil
	}
	return &a.matches[a.selectedIdx]
}

// Close closes the autocomplete
func (a *AutocompleteBox) Close() {
	a.active = false
	a.matches = nil
	a.query = ""
	a.scrollIdx = 0
}

// SetPosition sets the render position
func (a *AutocompleteBox) SetPosition(x, y int) {
	a.x = x
	a.y = y
}

// Render renders the autocomplete box
func (a AutocompleteBox) Render(view string, m Model) string {
	if !a.active || len(a.matches) == 0 {
		return view
	}

	style := DefaultDialogStyle()
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(style.KeyColor)).
		Bold(true)
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(style.TextColor))
	triggerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(style.TitleColor))

	// Calculate visible range
	endIdx := a.scrollIdx + 5
	if endIdx > len(a.matches) {
		endIdx = len(a.matches)
	}

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

	// Add scroll indicator if needed
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
		lines = append(lines, dimStyle.Render(indicator))
	}

	content := strings.Join(lines, "\n")

	const fixedWidth = 28
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(style.BorderColor)).
		Padding(0, 1).
		Width(fixedWidth).
		Render(content)

	bgLayer := lipgloss.NewLayer(view)
	dialogLayer := lipgloss.NewLayer(box).X(a.x).Y(a.y).Z(2)
	compositor := lipgloss.NewCompositor(bgLayer, dialogLayer)
	return compositor.Render()
}

var dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))