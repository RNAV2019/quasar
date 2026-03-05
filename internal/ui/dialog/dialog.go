// Package dialog provides overlay dialog components for the TUI.
package dialog

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/RNAV2019/quasar/internal/styles"
)

// Dimensions holds the terminal width and height needed for centering dialogs.
type Dimensions struct {
	Width  int
	Height int
}

// BaseDialog provides common dialog functionality.
type BaseDialog struct {
	Active bool
	Width  int
	Style  styles.DialogStyleConfig
}

// NewBaseDialog creates a new base dialog with default styling.
func NewBaseDialog(width int) BaseDialog {
	return BaseDialog{
		Active: false,
		Width:  width,
		Style:  styles.DefaultDialogStyle(),
	}
}

// Activate makes the dialog visible.
func (d *BaseDialog) Activate() {
	d.Active = true
}

// Deactivate hides the dialog.
func (d *BaseDialog) Deactivate() {
	d.Active = false
}

// centerDialog properly centers a rendered dialog box on the screen.
func centerDialog(view, dialogBox string, dim Dimensions) string {
	dialogWidth := lipgloss.Width(dialogBox)
	dialogX := (dim.Width - dialogWidth) / 2
	dialogY := 2

	bgLayer := lipgloss.NewLayer(view)
	dialogLayer := lipgloss.NewLayer(dialogBox).X(dialogX).Y(dialogY).Z(1)
	compositor := lipgloss.NewCompositor(bgLayer, dialogLayer)
	return compositor.Render()
}

// ConfirmDialog is a simple yes/no confirmation dialog.
type ConfirmDialog struct {
	BaseDialog
	Title   string
	Message string
	focused int // 0 = Yes, 1 = No
}

// NewConfirmDialog creates a new confirmation dialog.
func NewConfirmDialog(title, message string) ConfirmDialog {
	return ConfirmDialog{
		BaseDialog: NewBaseDialog(52),
		Title:      title,
		Message:    message,
		focused:    1, // Default to "No" for safety
	}
}

// Activate makes the dialog visible and defaults to "No".
func (d *ConfirmDialog) Activate() {
	d.BaseDialog.Activate()
	d.focused = 1
}

// SelectYes focuses the Yes option.
func (d *ConfirmDialog) SelectYes() {
	d.focused = 0
}

// SelectNo focuses the No option.
func (d *ConfirmDialog) SelectNo() {
	d.focused = 1
}

// ToggleSelection toggles between Yes and No.
func (d *ConfirmDialog) ToggleSelection() {
	d.focused = 1 - d.focused
}

// IsYesSelected returns true if Yes is selected.
func (d *ConfirmDialog) IsYesSelected() bool {
	return d.focused == 0
}

// MoveLeft moves selection to Yes.
func (d *ConfirmDialog) MoveLeft() {
	d.focused = 0
}

// MoveRight moves selection to No.
func (d *ConfirmDialog) MoveRight() {
	d.focused = 1
}

// Render renders the confirm dialog centered on the view.
func (d ConfirmDialog) Render(view string, dim Dimensions) (string, tea.Cursor) {
	if !d.Active {
		return view, tea.Cursor{}
	}

	var lines []string

	titleStyled := lipgloss.NewStyle().
		Foreground(d.Style.TitleColor).
		Bold(true).
		Render(d.Title)
	lines = append(lines, titleStyled)
	lines = append(lines, "")

	msgStyled := lipgloss.NewStyle().
		Foreground(d.Style.TextColor).
		Render(d.Message)
	lines = append(lines, msgStyled)
	lines = append(lines, "")

	yesStyle := lipgloss.NewStyle().Foreground(d.Style.DimColor)
	noStyle := lipgloss.NewStyle().Foreground(d.Style.DimColor)

	if d.focused == 0 {
		yesStyle = lipgloss.NewStyle().
			Foreground(d.Style.KeyColor).
			Bold(true)
	} else {
		noStyle = lipgloss.NewStyle().
			Foreground(d.Style.KeyColor).
			Bold(true)
	}

	yesText := yesStyle.Render("[ Yes ]")
	noText := noStyle.Render("[ No  ]")
	buttons := yesText + "   " + noText
	lines = append(lines, buttons)
	lines = append(lines, "")

	hintText := lipgloss.NewStyle().
		Foreground(d.Style.DimColor).
		Render("←/→ to select, Enter to confirm")
	lines = append(lines, hintText)

	content := strings.Join(lines, "\n")

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.Style.BorderColor).
		Padding(0, 2).
		Render(content)

	return centerDialog(view, dialogBox, dim), tea.Cursor{}
}
