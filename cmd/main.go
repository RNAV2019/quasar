package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/RNAV2019/quasar/internal/config"
	"github.com/RNAV2019/quasar/internal/ui"
)

func main() {
	cfg, err := config.SetupEnvironment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.InitialModel(cfg))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
