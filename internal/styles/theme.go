// Package styles defines the shared color palette, lipgloss styles, and mode
// constants used across the quasar TUI.
package styles

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Color palette (Catppuccin Mocha inspired)
var (
	ColorBackground = lipgloss.Color("#181825")
	ColorSurface    = lipgloss.Color("#1e1e2e")
	ColorOverlay    = lipgloss.Color("#313244")
	ColorText       = lipgloss.Color("#CDD6F4")
	ColorTextDim    = lipgloss.Color("#6C7086")

	ColorBlue   = lipgloss.Color("#89B4FA")
	ColorGreen  = lipgloss.Color("#A6E3A1")
	ColorPink   = lipgloss.Color("#F5C2E7")
	ColorPeach  = lipgloss.Color("#FAB387")
	ColorPurple = lipgloss.Color("#CBA6F7")
	ColorRed    = lipgloss.Color("#F38BA8")
	ColorYellow = lipgloss.Color("#F9E2AF")

	ColorNormalMode  = ColorBlue
	ColorInsertMode  = ColorGreen
	ColorSelectMode  = ColorPurple
	ColorCommandMode = ColorPeach
)

// Statusline styles
var (
	NormalModeStyle = lipgloss.NewStyle().
			Foreground(ColorBackground).
			Background(ColorNormalMode).
			Bold(true)

	InsertModeStyle = lipgloss.NewStyle().
			Foreground(ColorBackground).
			Background(ColorInsertMode).
			Bold(true)

	SelectModeStyle = lipgloss.NewStyle().
			Foreground(ColorBackground).
			Background(ColorSelectMode).
			Bold(true)

	CommandModeStyle = lipgloss.NewStyle().
				Foreground(ColorBackground).
				Background(ColorCommandMode).
				Bold(true)

	ClearStyle = lipgloss.NewStyle()
)

// Message styles for CLI output
var (
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed).
			Padding(1, 2)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Padding(1, 2)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorBlue).
			Padding(1, 2)

	BoldStyle = lipgloss.NewStyle().Bold(true)
)

// Dialog styles
var (
	BorderColor = ColorBlue
	TextColor   = ColorText
	AccentColor = ColorPink
	KeyColor    = ColorGreen
	DimColor    = ColorTextDim

	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
			Foreground(AccentColor).
			Bold(true)

	PromptStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	ConfirmButtonStyle = lipgloss.NewStyle().
				Foreground(KeyColor).
				Bold(true)

	CancelButtonStyle = lipgloss.NewStyle().
				Foreground(ColorRed)
)

// Editor styles
var (
	GutterStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	CurrentLineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Bold(true)
	TabHighlightStyle = lipgloss.NewStyle().Background(ColorOverlay)
	SelectionStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#5c5c8a"))

	MathGutterIndicator  = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Render("│")
	TextGutterIndicator  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")
	ErrorGutterIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("│")
)

// File tree styles
var (
	TreeDirStyle      = lipgloss.NewStyle().Foreground(ColorBlue)
	TreeFileStyle     = lipgloss.NewStyle().Foreground(ColorText)
	TreeSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#45475a")).
				Foreground(ColorPink)
	TreeEmptyStyle  = lipgloss.NewStyle().Foreground(ColorTextDim)
	TreeIndentStyle = lipgloss.NewStyle().Foreground(ColorTextDim)
)

// Autocomplete styles
var (
	DimStyle = lipgloss.NewStyle().Foreground(ColorTextDim)
)

// Mode constants for use across packages.
const (
	ModeNormal  = 0
	ModeInsert  = 1
	ModeSelect  = 2
	ModeCommand = 3
)

// DialogStyleConfig holds the styling configuration for dialogs.
type DialogStyleConfig struct {
	BorderColor color.Color
	TitleColor  color.Color
	TextColor   color.Color
	KeyColor    color.Color
	DimColor    color.Color
}

// DefaultDialogStyle returns the default Catppuccin-inspired styling.
func DefaultDialogStyle() DialogStyleConfig {
	return DialogStyleConfig{
		BorderColor: ColorBlue,
		TitleColor:  ColorPink,
		TextColor:   ColorText,
		KeyColor:    ColorGreen,
		DimColor:    ColorTextDim,
	}
}

// GetModeStyle returns the appropriate style for the given mode.
func GetModeStyle(mode int) lipgloss.Style {
	switch mode {
	case ModeInsert:
		return InsertModeStyle
	case ModeSelect:
		return SelectModeStyle
	case ModeCommand:
		return CommandModeStyle
	default:
		return NormalModeStyle
	}
}

// GetModeColor returns the color for the given mode.
func GetModeColor(mode int) color.Color {
	switch mode {
	case ModeInsert:
		return ColorInsertMode
	case ModeSelect:
		return ColorSelectMode
	case ModeCommand:
		return ColorCommandMode
	default:
		return ColorNormalMode
	}
}
