package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// Snippet represents a user-defined math snippet.
type Snippet struct {
	Trigger           string `yaml:"trigger"`
	Label             string `yaml:"label"`
	Body              string `yaml:"body"`
	CursorPlaceholder string `yaml:"cursor"`
}

type snippetsFile struct {
	Snippets []Snippet `yaml:"snippets"`
}

const defaultSnippetsYAML = `# Custom math snippets for quasar
# These appear in autocomplete when editing inside math blocks.
#
# snippets:
#   - trigger: matrix
#     label: "Matrix"
#     body: |
#       \begin{bmatrix}
#         $0
#       \end{bmatrix}
#     cursor: "$0"
`

// LoadSnippets reads and parses snippets from the config directory.
func LoadSnippets(configDir string) ([]Snippet, error) {
	path := filepath.Join(configDir, "snippets.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var f snippetsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}

	for i := range f.Snippets {
		if f.Snippets[i].CursorPlaceholder == "" {
			f.Snippets[i].CursorPlaceholder = "$0"
		}
		// Trim trailing newline that YAML block scalars add
		f.Snippets[i].Body = strings.TrimRight(f.Snippets[i].Body, "\n")
	}

	return f.Snippets, nil
}
