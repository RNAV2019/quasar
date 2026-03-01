package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// Metadata represents the YAML front matter
type Metadata struct {
	Title  string `yaml:"title"`
	Date   string `yaml:"date,omitempty"`
	Tags   []string `yaml:"tags,omitempty"`
}

// ExtractFrontMatter parses YAML front matter from markdown content
func ExtractFrontMatter(content []string) (*Metadata, []string, error) {
	if len(content) == 0 {
		return &Metadata{}, content, nil
	}

	// Check if content starts with front matter delimiter
	if content[0] != "---" {
		return &Metadata{}, content, nil
	}

	// Find the closing delimiter
	closingIndex := -1
	for i := 1; i < len(content); i++ {
		if content[i] == "---" {
			closingIndex = i
			break
		}
	}

	if closingIndex == -1 {
		return nil, content, fmt.Errorf("unclosed front matter block")
	}

	// Extract front matter YAML
	frontMatterLines := content[1:closingIndex]
	yamlContent := strings.Join(frontMatterLines, "\n")

	var metadata Metadata
	if err := yaml.Unmarshal([]byte(yamlContent), &metadata); err != nil {
		return nil, content, fmt.Errorf("invalid YAML front matter: %w", err)
	}

	// Return metadata and content without front matter
	remainingContent := content[closingIndex+1:]
	return &metadata, remainingContent, nil
}

// ToMarkdownContent converts blocks to markdown with front matter
// If preserveFrontMatter is provided, it uses those front matter lines
// and skips front matter in the first block to avoid duplication
func (m *Model) ToMarkdownContent(preserveFrontMatter []string) ([]string, error) {
	var lines []string

	// Preserve original front matter if it exists
	if len(preserveFrontMatter) > 0 {
		lines = append(lines, preserveFrontMatter...)
	}

	// Convert blocks to markdown
	for blockIdx, block := range m.Blocks {
		content := block.Lines

		// If we have preserved front matter, skip front matter lines in the first block
		if blockIdx == 0 && len(preserveFrontMatter) > 0 && len(content) > 0 && content[0] == "---" {
			// Find where front matter ends in this block
			for i := 1; i < len(content); i++ {
				if content[i] == "---" {
					// Skip past the closing --- and any leading empty lines
					content = content[i+1:]
					for len(content) > 0 && content[0] == "" {
						content = content[1:]
					}
					break
				}
			}
		}

		switch block.Type {
		case TextBlock:
			lines = append(lines, content...)
		case MathBlock:
			lines = append(lines, content...)
		}
		lines = append(lines, "") // blank line between blocks
	}

	// Remove trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines, nil
}

// ExtractTitle attempts to extract a title from the first text block if no front matter exists
func (m *Model) ExtractTitle() string {
	for _, block := range m.Blocks {
		if block.Type == TextBlock {
			for _, line := range block.Lines {
				line = strings.TrimSpace(line)
				if line != "" {
					// Check for markdown heading
					if strings.HasPrefix(line, "# ") {
						return strings.TrimSpace(line[2:])
					}
					// Use first non-empty line (up to 50 chars)
					if len(line) > 50 {
						return line[:47] + "..."
					}
					return line
				}
			}
		}
	}
	return ""
}

// SanitizeFilename converts a title to a safe filename
func SanitizeFilename(title string) string {
	// Remove or replace problematic characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9\s\-_]`)
	clean := reg.ReplaceAllString(title, "")
	
	// Replace spaces with hyphens
	clean = regexp.MustCompile(`\s+`).ReplaceAllString(clean, "-")
	
	// Remove multiple consecutive hyphens
	clean = regexp.MustCompile(`-+`).ReplaceAllString(clean, "-")
	
	// Trim hyphens from start and end
	clean = strings.Trim(clean, "-")
	
	// Ensure it's not empty
	if clean == "" {
		return "untitled"
	}
	
	// Limit length
	if len(clean) > 90 {
		clean = clean[:90]
		clean = strings.Trim(clean, "-")
	}
	
	return clean
}

// SaveToFile saves the model content to a markdown file
func (m *Model) SaveToFile(notesDir string) error {
	// Convert blocks to lines for front matter extraction
	allLines := []string{}
	for _, block := range m.Blocks {
		allLines = append(allLines, block.Lines...)
	}

	// Extract front matter from current content  
	metadata, _, err := ExtractFrontMatter(allLines)
	if err != nil {
		return fmt.Errorf("failed to parse front matter: %w", err)
	}

	// If no title in front matter, try to extract from content
	if metadata.Title == "" {
		extractedTitle := m.ExtractTitle()
		if extractedTitle == "" {
			return fmt.Errorf("no title found: add a title field to the YAML front matter or start with a heading")
		}
		metadata.Title = extractedTitle
	}

	// Generate filename from title
	filename := SanitizeFilename(metadata.Title) + ".md"
	filepath := filepath.Join(notesDir, filename)

	// Get the original front matter lines to preserve
	var frontMatterLines []string
	if len(allLines) > 0 && allLines[0] == "---" {
		// Find closing delimiter
		for i := 1; i < len(allLines); i++ {
			if allLines[i] == "---" {
				// Include both --- delimiters
				frontMatterLines = allLines[:i+1]
				break
			}
		}
	}

	// Convert to markdown content preserving original front matter
	content, err := m.ToMarkdownContent(frontMatterLines)
	if err != nil {
		return fmt.Errorf("failed to generate markdown content: %w", err)
	}

	// Write file
	fileContent := strings.Join(content, "\n")
	if err := os.WriteFile(filepath, []byte(fileContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// createModelFromLines creates a new Model from a slice of lines, parsing them into blocks
func (m *Model) createModelFromLines(lines []string) *Model {
	if len(lines) == 0 {
		return &Model{Blocks: []Block{{Type: TextBlock, Lines: []string{""}}}}
	}

	blocks := []Block{}
	currentBlock := Block{Type: TextBlock, Lines: []string{}}
	
	for _, line := range lines {
		if line == "$$" && currentBlock.Type == TextBlock {
			// Start of math block - save current text block if not empty
			if len(currentBlock.Lines) > 0 && !(len(currentBlock.Lines) == 1 && currentBlock.Lines[0] == "") {
				blocks = append(blocks, currentBlock)
			}
			currentBlock = Block{Type: MathBlock, Lines: []string{line}}
		} else if line == "$$" && currentBlock.Type == MathBlock {
			// End of math block
			currentBlock.Lines = append(currentBlock.Lines, line)
			blocks = append(blocks, currentBlock)
			currentBlock = Block{Type: TextBlock, Lines: []string{}}
		} else {
			// Regular line
			currentBlock.Lines = append(currentBlock.Lines, line)
		}
	}
	
	// Add final block if not empty
	if len(currentBlock.Lines) > 0 {
		blocks = append(blocks, currentBlock)
	}
	
	if len(blocks) == 0 {
		blocks = []Block{{Type: TextBlock, Lines: []string{""}}}
	}
	
	return &Model{Blocks: blocks}
}