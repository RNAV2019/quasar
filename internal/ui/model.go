// Package ui implements the main Bubble Tea TUI model, view rendering, and
// keyboard event handling for the quasar editor.
package ui

import (
	"regexp"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/RNAV2019/quasar/internal/config"
	"github.com/RNAV2019/quasar/internal/editor"
	"github.com/RNAV2019/quasar/internal/ui/autocomplete"
	"github.com/RNAV2019/quasar/internal/ui/dialog"
	"github.com/RNAV2019/quasar/internal/ui/filetree"
)

// Mode represents the current editor interaction mode.
type Mode int

const (
	// Normal is the default navigation mode.
	Normal Mode = iota
	// Insert is the text editing mode.
	Insert
	// Select is the visual selection mode.
	Select
	// Command is the command-line input mode.
	Command
	// NewNote is the new note dialog mode.
	NewNote
	// Help is the help dialog mode.
	Help
	// Error is the error dialog mode.
	Error
	// DeleteConfirm is the delete confirmation dialog mode.
	DeleteConfirm
	// QuitConfirm is the quit confirmation dialog mode.
	QuitConfirm
	// FileTreeDelete is the file tree delete confirmation mode.
	FileTreeDelete
	// FileTreeRename is the file tree rename dialog mode.
	FileTreeRename
)

// Model is the top-level Bubble Tea model for the quasar TUI.
type Model struct {
	mode               Mode
	width              int
	height             int
	Time               time.Time
	Editor             editor.Model
	Config             *config.Config
	InlineRenders      map[string]InlineMathRender
	PendingRenders     int
	TotalRenders       int // Total renders needed for current document
	CompiledMath       []string
	CmdInput           textinput.Model
	StatusMessage      string
	ParsedDoc          *editor.Document
	FileTree           *filetree.FileTree
	ShowFileTree       bool
	pendingSpace       bool
	NewNoteDialog        dialog.InputDialog
	HelpDialog           dialog.HelpDialog
	ErrorDialog          dialog.ErrorDialog
	DeleteConfirmDialog  dialog.ConfirmDialog
	QuitConfirmDialog    dialog.ConfirmDialog
	FileTreeDeleteDialog dialog.ConfirmDialog
	RenameDialog         dialog.InputDialog
	NotebookName       string
	NotebookPath       string
	CurrentFile        string
	OriginalMetadata   *editor.Metadata // Track original front matter for file renaming
	Autocomplete       autocomplete.Box
	slashStartCol      int
	Dirty              bool
	DocumentLoading    bool   // True while initial document images are being compiled
	fileGeneration     uint64 // Increments on each file load to discard stale render results

	PendingOp       string
	YankBuffer      string
	YankWasLineWise bool
	CopyBuffer      string
	KeyPreview      string // Shows current key sequence being entered
}

// TickMsg is sent on every tick to drive periodic updates.
type TickMsg time.Time

// BlockProcessedMsg is sent when a math block finishes compiling.
type BlockProcessedMsg struct {
	BlockIdx    int
	ImageID     uint32
	ImageCols   int
	ImageHeight int
	Error       error
	Source      string
	Generation  uint64
}

// InlineMathProcessedMsg is sent when an inline math expression finishes compiling.
type InlineMathProcessedMsg struct {
	BlockIdx    int
	LineIdx     int
	StartCol    int
	EndCol      int
	ImageID     uint32
	ImageCols   int
	ImageHeight int
	Error       error
	Generation  uint64
}

// InlineMathRender holds the rendered image data for an inline math expression.
type InlineMathRender struct {
	ImageID     uint32
	ImageCols   int
	ImageHeight int
	Length      int // Width in columns for placeholder
	TextLength  int // Original text length for hover detection
}

var inlineMathRe = regexp.MustCompile(`\$[^\$]*\$`)

// InitialModel creates the default Model with the given configuration.
func InitialModel(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = ""
	ti.SetVirtualCursor(false)
	ti.CharLimit = 30

	m := Model{
		mode:                Normal,
		Time:                time.Now(),
		Editor:              editor.NewModel(),
		Config:              cfg,
		InlineRenders:       make(map[string]InlineMathRender),
		CmdInput:            ti,
		FileTree:            filetree.New(cfg.NotesDir),
		ShowFileTree:        false,
		NewNoteDialog:        dialog.NewInputDialog(),
		HelpDialog:           dialog.NewHelpDialog(),
		ErrorDialog:          dialog.NewErrorDialog(),
		DeleteConfirmDialog:  dialog.NewConfirmDialog("Delete Note", "Delete this note?"),
		QuitConfirmDialog:    dialog.NewConfirmDialog("Quit", "Quit without saving?"),
		FileTreeDeleteDialog: dialog.NewConfirmDialog("Delete", ""),
		RenameDialog:         dialog.NewInputDialog(),
		Autocomplete:        autocomplete.NewBox(),
		PendingOp:           "",
		YankBuffer:          "",
		YankWasLineWise:     false,
		CopyBuffer:          "",
	}
	m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
	return m
}

// InitialModelWithNotebook creates a Model pre-loaded with the given notebook.
func InitialModelWithNotebook(cfg *config.Config, notebookPath, notebookName string) Model {
	m := InitialModel(cfg)
	m.NotebookName = notebookName
	m.NotebookPath = notebookPath
	m.FileTree = filetree.NewForNotebook(notebookPath, notebookName)
	m.ShowFileTree = true
	m.FileTree.Focused = true
	return m
}

// Init returns the initial commands for the Bubble Tea runtime.
func (m Model) Init() tea.Cmd {
	return tea.Batch(doTick(), m.processDirtyBlocks())
}

func doTick() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
