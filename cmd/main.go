package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/RNAV2019/quasar/internal/cli"
	"github.com/RNAV2019/quasar/internal/config"
	"github.com/RNAV2019/quasar/internal/latex"
	"github.com/RNAV2019/quasar/internal/notebook"
	"github.com/RNAV2019/quasar/internal/ui"
)

func main() {
	cfg, err := config.SetupEnvironment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cli.OpenNotebookFunc = func(name string) {
		// Clear any existing kitty graphics before starting
		latex.DeleteAllImages()

		notebookPath := notebook.Path(cfg.NotesDir, name)
		p := tea.NewProgram(ui.InitialModelWithNotebook(cfg, notebookPath, name))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	rootCmd := cli.NewRootCmd(cfg)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
