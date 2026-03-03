package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// DialogStyle holds the styling configuration for dialogs
type DialogStyle struct {
	BorderColor string
	TitleColor  string
	TextColor   string
	KeyColor    string
	DimColor    string
}

// DefaultDialogStyle returns the default Catppuccin-inspired styling
func DefaultDialogStyle() DialogStyle {
	return DialogStyle{
		BorderColor: "#89B4FA",
		TitleColor:  "#F5C2E7",
		TextColor:   "#CDD6F4",
		KeyColor:    "#A6E3A1",
		DimColor:    "#6C7086",
	}
}

// BaseDialog provides common dialog functionality
type BaseDialog struct {
	active bool
	width  int
	style  DialogStyle
}

// NewBaseDialog creates a new base dialog with default styling
func NewBaseDialog(width int) BaseDialog {
	return BaseDialog{
		active: false,
		width:  width,
		style:  DefaultDialogStyle(),
	}
}

func (d *BaseDialog) Activate() {
	d.active = true
}

func (d *BaseDialog) Deactivate() {
	d.active = false
}

func (d *BaseDialog) Active() bool {
	return d.active
}

func (d *BaseDialog) SetWidth(w int) {
	d.width = w
}

// centerDialog properly centers a rendered dialog box on the screen
func centerDialog(view, dialogBox string, m Model) string {
	dialogWidth := lipgloss.Width(dialogBox)
	dialogX := (m.width - dialogWidth) / 2
	dialogY := 2

	bgLayer := lipgloss.NewLayer(view)
	dialogLayer := lipgloss.NewLayer(dialogBox).X(dialogX).Y(dialogY).Z(1)
	compositor := lipgloss.NewCompositor(bgLayer, dialogLayer)
	return compositor.Render()
}

// ConfirmDialog is a simple yes/no confirmation dialog
type ConfirmDialog struct {
	BaseDialog
	Title   string
	Message string
	focused int // 0 = Yes, 1 = No
}

// NewConfirmDialog creates a new confirmation dialog
func NewConfirmDialog(title, message string) ConfirmDialog {
	return ConfirmDialog{
		BaseDialog: NewBaseDialog(52),
		Title:      title,
		Message:    message,
		focused:    1, // Default to "No" for safety
	}
}

func (d *ConfirmDialog) Activate() {
	d.BaseDialog.Activate()
	d.focused = 1 // Default to "No" for safety
}

// SelectYes focuses the Yes option
func (d *ConfirmDialog) SelectYes() {
	d.focused = 0
}

// SelectNo focuses the No option
func (d *ConfirmDialog) SelectNo() {
	d.focused = 1
}

// ToggleSelection toggles between Yes and No
func (d *ConfirmDialog) ToggleSelection() {
	d.focused = 1 - d.focused
}

// IsYesSelected returns true if Yes is selected
func (d *ConfirmDialog) IsYesSelected() bool {
	return d.focused == 0
}

// MoveLeft moves selection to Yes
func (d *ConfirmDialog) MoveLeft() {
	d.focused = 0
}

// MoveRight moves selection to No
func (d *ConfirmDialog) MoveRight() {
	d.focused = 1
}

func (d ConfirmDialog) Render(view string, m Model) (string, tea.Cursor) {
	if !d.active {
		return view, tea.Cursor{}
	}

	// Build content lines
	var lines []string

	// Title (centered)
	titleStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color(d.style.TitleColor)).
		Bold(true).
		Render(d.Title)
	lines = append(lines, titleStyled)

	// Empty line
	lines = append(lines, "")

	// Message (centered)
	msgStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color(d.style.TextColor)).
		Render(d.Message)
	lines = append(lines, msgStyled)

	// Empty line
	lines = append(lines, "")

	// Buttons
	yesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(d.style.DimColor))
	noStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(d.style.DimColor))

	if d.focused == 0 {
		yesStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.style.KeyColor)).
			Bold(true)
	} else {
		noStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(d.style.KeyColor)).
			Bold(true)
	}

	yesText := yesStyle.Render("[ Yes ]")
	noText := noStyle.Render("[ No  ]")
	buttons := yesText + "   " + noText
	lines = append(lines, buttons)

	// Empty line
	lines = append(lines, "")

	// Hint
	hintText := lipgloss.NewStyle().
		Foreground(lipgloss.Color(d.style.DimColor)).
		Render("←/→ to select, Enter to confirm")
	lines = append(lines, hintText)

	content := strings.Join(lines, "\n")

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(d.style.BorderColor)).
		Padding(0, 2).
		Render(content)

	return centerDialog(view, dialogBox, m), tea.Cursor{}
}
