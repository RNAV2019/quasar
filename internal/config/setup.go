// Package config manages application configuration, directory setup, and
// LaTeX format file initialization.
package config

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/RNAV2019/quasar/internal/git"
)

// Config holds the resolved directory paths for the application.
type Config struct {
	CacheDir string
	NotesDir string
}

// SetupEnvironment creates the cache and notes directories and returns the resolved Config.
func SetupEnvironment() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("Failed to get home directory: %w", err)
	}

	cachePath := filepath.Join(home, ".cache", "quasar")
	notesPath := filepath.Join(home, "Documents", "quasar")

	// Create cache directory
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create directory %s: %w", cachePath, err)
	}

	// Check if notes directory exists - if not, this is first run
	isFirstRun := false
	if _, err := os.Stat(notesPath); os.IsNotExist(err) {
		isFirstRun = true
	}

	// Create notes directory
	if err := os.MkdirAll(notesPath, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create directory %s: %w", notesPath, err)
	}

	cfg := &Config{
		CacheDir: cachePath,
		NotesDir: notesPath,
	}

	// Initialize Git on first run
	if isFirstRun {
		if err := initializeGit(notesPath); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// initializeGit prompts the user for a Git repo URL and sets up the repository.
func initializeGit(notesDir string) error {
	fmt.Print("\n📚 Git Setup\n")
	fmt.Print("─────────────────────────────────────────────\n")
	fmt.Print("Your notes will be backed up with Git.\n\n")
	fmt.Print("Enter a Git repository URL to clone from (press Enter to create new):\n> ")

	repoURL := readInputLine()
	repoURL = strings.TrimSpace(repoURL)

	if repoURL != "" {
		// Clone from existing repo
		fmt.Printf("\n⏳ Cloning from %s...\n", repoURL)

		// Remove the notes directory since git clone will create it
		if err := os.RemoveAll(notesDir); err != nil {
			return fmt.Errorf("failed to remove notes directory: %w", err)
		}

		if err := git.Clone(repoURL, notesDir); err != nil {
			return err
		}

		fmt.Print("✓ Repository cloned successfully!\n")
		return nil
	}

	// Initialize new repo
	fmt.Print("\n⏳ Initializing new Git repository...\n")

	if err := git.Init(notesDir); err != nil {
		return err
	}

	// Create initial commit with empty .gitkeep
	gitkeepPath := filepath.Join(notesDir, ".gitkeep")
	if err := os.WriteFile(gitkeepPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create .gitkeep: %w", err)
	}

	if err := git.AddAll(notesDir); err != nil {
		return err
	}

	if _, err := git.Commit(notesDir, "Initial commit"); err != nil {
		return err
	}

	fmt.Print("✓ Repository initialized successfully!\n")
	fmt.Print("\nYou can add a remote repository later with:\n")
	fmt.Print("  quasar backup --set-remote <url>\n\n")

	return nil
}

// readInputLine reads an entire line from stdin including spaces.
func readInputLine() string {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSuffix(line, "\n")
}

// NeedsLatexSetup reports whether the LaTeX format files need to be rebuilt.
func (c *Config) NeedsLatexSetup() bool {
	for _, name := range []string{"quasar-math-multi", "quasar-math-inline"} {
		fmtPath := filepath.Join(c.CacheDir, name+".fmt")
		stampPath := filepath.Join(c.CacheDir, name+".sha256")

		sum := fmt.Sprintf("%x", sha256.Sum256([]byte(latexPreambles()[name])))
		if existing, err := os.ReadFile(stampPath); err != nil || string(existing) != sum {
			return true
		}
		if _, err := os.Stat(fmtPath); err != nil {
			return true
		}
	}
	return false
}

// InitLatexFormats builds and caches the LaTeX format files used for math compilation.
func (c *Config) InitLatexFormats() error {
	return initLatexFormat(c.CacheDir)
}

func latexPreambles() map[string]string {
	basePreambleTop := `\let\originaldump\dump
\let\dump\relax
\input latex.ltx
\let\dump\originaldump
`

	multiPreamble := basePreambleTop + `\documentclass{article}
\usepackage[paperwidth=100cm,paperheight=50cm,margin=0pt]{geometry}
\usepackage[T1]{fontenc}
\usepackage{lmodern}
\usepackage{amsmath}
\usepackage{amssymb}
\pagestyle{empty}
\setlength{\abovedisplayskip}{0pt}
\setlength{\belowdisplayskip}{0pt}
\setlength{\abovedisplayshortskip}{0pt}
\setlength{\belowdisplayshortskip}{0pt}
\dump
`

	inlinePreamble := basePreambleTop + `\documentclass[preview,border=1pt]{standalone}
\usepackage[T1]{fontenc}
\usepackage{lmodern}
\usepackage{amsmath}
\usepackage{amssymb}
\dump
`

	return map[string]string{
		"quasar-math-multi":  multiPreamble,
		"quasar-math-inline": inlinePreamble,
	}
}

func initLatexFormat(cacheDir string) error {
	formats := latexPreambles()

	tmpDir, err := os.MkdirTemp(cacheDir, "fmt-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	for jobName, preamble := range formats {
		fmtPath := filepath.Join(cacheDir, jobName+".fmt")
		stampPath := filepath.Join(cacheDir, jobName+".sha256")

		sum := fmt.Sprintf("%x", sha256.Sum256([]byte(preamble)))

		if existing, err := os.ReadFile(stampPath); err == nil && string(existing) == sum {
			if _, err := os.Stat(fmtPath); err == nil {
				continue
			}
		}

		texPath := filepath.Join(tmpDir, jobName+".tex")
		if err := os.WriteFile(texPath, []byte(preamble), 0644); err != nil {
			return fmt.Errorf("failed to write preamble for %s: %w", jobName, err)
		}

		cmd := exec.Command("pdftex", "-ini", "-etex", "-interaction=nonstopmode",
			fmt.Sprintf("-output-directory=%s", tmpDir),
			fmt.Sprintf("-jobname=%s", jobName),
			texPath)

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create format %s: %w\nOutput: %s", jobName, err, string(output))
		}

		generatedFmt := filepath.Join(tmpDir, jobName+".fmt")
		data, err := os.ReadFile(generatedFmt)
		if err != nil {
			return fmt.Errorf("failed to read generated format file %s: %w", jobName, err)
		}
		if err := os.WriteFile(fmtPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write format file %s to cache: %w", jobName, err)
		}
		if err := os.WriteFile(stampPath, []byte(sum), 0644); err != nil {
			return fmt.Errorf("failed to write stamp file %s: %w", stampPath, err)
		}
	}

	return nil
}
