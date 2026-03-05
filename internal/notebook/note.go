package notebook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// NoteSpec describes the parsed components of a note name and its resolved path.
type NoteSpec struct {
	Name     string // Original name input (before colon)
	Filename string // lowercase, spaces -> dashes
	Tag      string // Tag after colon, or empty
	Path     string // Full path to the note file
}

// FrontMatter represents the YAML front matter written into new notes.
type FrontMatter struct {
	Title string `yaml:"title"`
	Tag   string `yaml:"tag,omitempty"`
	Date  string `yaml:"date"`
}

// ParseNoteName parses an input string of the form "Name" or "Name:Tag" into a NoteSpec.
func ParseNoteName(input, notebookPath string) NoteSpec {
	spec := NoteSpec{}

	colonIdx := strings.Index(input, ":")
	if colonIdx == -1 {
		spec.Name = strings.TrimSpace(input)
		spec.Tag = ""
	} else {
		spec.Name = strings.TrimSpace(input[:colonIdx])
		spec.Tag = strings.TrimSpace(input[colonIdx+1:])
	}

	spec.Filename = strings.ToLower(spec.Name)
	spec.Filename = strings.ReplaceAll(spec.Filename, " ", "-")
	spec.Filename = strings.ReplaceAll(spec.Filename, "_", "-")

	if spec.Tag != "" {
		spec.Path = filepath.Join(notebookPath, spec.Tag, spec.Filename+".md")
	} else {
		spec.Path = filepath.Join(notebookPath, spec.Filename+".md")
	}

	return spec
}

// Create writes a new markdown note file with YAML front matter.
func (n NoteSpec) Create() error {
	dir := filepath.Dir(n.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if _, err := os.Stat(n.Path); err == nil {
		return fmt.Errorf("note '%s' already exists", n.Filename)
	}

	fm := FrontMatter{
		Title: n.Filename,
		Tag:   n.Tag,
		Date:  time.Now().Format("2006-01-02"),
	}

	fmData, err := yaml.Marshal(&fm)
	if err != nil {
		return fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	var content strings.Builder
	content.WriteString("---\n")
	content.Write(fmData)
	content.WriteString("---\n\n")

	if err := os.WriteFile(n.Path, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}

	return nil
}

// EnsureDirectoryExists creates the tag subdirectory if one is specified.
func (n NoteSpec) EnsureDirectoryExists() error {
	if n.Tag == "" {
		return nil
	}

	dir := filepath.Dir(n.Path)
	return os.MkdirAll(dir, 0755)
}
