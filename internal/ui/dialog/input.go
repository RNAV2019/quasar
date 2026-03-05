package dialog

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/RNAV2019/quasar/internal/styles"
)

// InputDialog is a unified text input dialog used for new note creation and
// renaming. It contains a textinput, title, and hint text.
type InputDialog struct {
	TextInput    textinput.Model
	Active       bool
	width        int
	style        styles.DialogStyleConfig
	title        string
	hint         string
	isDir        bool
	originalName string
}

// NewInputDialog creates a new input dialog.
func NewInputDialog() InputDialog {
	ti := textinput.New()
	ti.Placeholder = "NoteName or NoteName:Tag"
	ti.Focus()

	return InputDialog{
		TextInput: ti,
		Active:    false,
		width:     60,
		style:     styles.DefaultDialogStyle(),
	}
}

// ActivateEmpty activates the dialog for new note creation with an empty input.
func (d *InputDialog) ActivateEmpty() {
	d.Active = true
	d.title = "New Note"
	d.hint = "format: Name or Name:Tag"
	d.isDir = false
	d.originalName = ""
	d.TextInput.Placeholder = "NoteName or NoteName:Tag"
	d.TextInput.SetValue("")
	d.TextInput.Focus()
}

// ActivateWithValue activates the dialog for renaming with a pre-filled value.
func (d *InputDialog) ActivateWithValue(name string, isDir bool) {
	d.Active = true
	d.isDir = isDir
	d.originalName = name
	if isDir {
		d.title = "Rename Folder"
		d.hint = "Enter new name, Enter to confirm"
	} else {
		d.title = "Rename File"
		d.hint = ".md extension will be added automatically"
	}
	displayName := name
	if !isDir && strings.HasSuffix(name, ".md") {
		displayName = name[:len(name)-3]
	}
	d.TextInput.Placeholder = "New name"
	d.TextInput.SetValue(displayName)
	d.TextInput.Focus()
}

// Deactivate hides the dialog.
func (d *InputDialog) Deactivate() {
	d.Active = false
	d.TextInput.Blur()
}

// Value returns the current text input value.
func (d *InputDialog) Value() string {
	return d.TextInput.Value()
}

// IsDir returns whether the dialog is renaming a directory.
func (d *InputDialog) IsDir() bool {
	return d.isDir
}

// Update passes a message to the underlying text input.
func (d *InputDialog) Update(msg tea.Msg) {
	d.TextInput, _ = d.TextInput.Update(msg)
}

// Render renders the input dialog centered on the view.
func (d InputDialog) Render(view string, dim Dimensions) (string, tea.Cursor) {
	if !d.Active {
		return view, tea.Cursor{}
	}

	titleStyled := lipgloss.NewStyle().
		Foreground(d.style.TitleColor).
		Bold(true).
		Render(d.title)

	inputView := d.TextInput.View()

	hintStyled := lipgloss.NewStyle().
		Foreground(d.style.DimColor).
		Render(d.hint)

	lines := []string{
		titleStyled,
		"",
		inputView,
		"",
		hintStyled,
	}

	content := strings.Join(lines, "\n")

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.style.BorderColor).
		Padding(0, 2).
		Render(content)

	dialogWidth := lipgloss.Width(dialogBox)
	dialogX := (dim.Width - dialogWidth) / 2
	dialogY := 1

	bgLayer := lipgloss.NewLayer(view)
	dialogLayer := lipgloss.NewLayer(dialogBox).X(dialogX).Y(dialogY).Z(1)
	compositor := lipgloss.NewCompositor(bgLayer, dialogLayer)
	view = compositor.Render()

	var cursorConfig tea.Cursor
	cmdCursor := d.TextInput.Cursor()
	textColor := d.style.TextColor
	if cmdCursor != nil {
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: dialogX + 3 + cmdCursor.Position.X, Y: dialogY + 3 + cmdCursor.Position.Y},
			Shape:    tea.CursorBar,
			Color:    textColor,
		}
	} else {
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: dialogX + 5, Y: dialogY + 3},
			Shape:    tea.CursorBar,
			Color:    textColor,
		}
	}

	return view, cursorConfig
}
