package editor

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// Metadata represents the YAML front matter.
type Metadata struct {
	Title  string `yaml:"title"`
	Date   string `yaml:"date,omitempty"`
	Tag   string `yaml:"tag,omitempty"`
}

// ExtractFrontMatter parses YAML front matter from markdown content.
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

// ToMarkdownContent converts blocks to markdown with front matter.
// If preserveFrontMatter is provided, it uses those front matter lines
// and skips front matter in the first block to avoid duplication.
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

		// Add blank line before math block if previous content exists
		if block.Type == MathBlock && len(lines) > 0 {
			lastLine := lines[len(lines)-1]
			if lastLine != "" {
				lines = append(lines, "")
			}
		}

		lines = append(lines, content...)

		// Add blank line after math block only if next block doesn't start with empty lines
		if block.Type == MathBlock && blockIdx < len(m.Blocks)-1 {
			nextBlock := m.Blocks[blockIdx+1]
			if len(nextBlock.Lines) > 0 && nextBlock.Lines[0] != "" {
				lines = append(lines, "")
			}
		}

		// Add blank line after text block if next block is also text
		if block.Type == TextBlock && blockIdx < len(m.Blocks)-1 && m.Blocks[blockIdx+1].Type == TextBlock {
			lines = append(lines, "")
		}
	}

	// Remove trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines, nil
}

// ExtractTitle attempts to extract a title from the first text block if no front matter exists.
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

// SanitizeFilename converts a title to a safe filename.
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

