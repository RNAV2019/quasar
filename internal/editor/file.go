package editor

import (
	"fmt"
	"os"
	"strings"
)

// LoadFromFile loads a markdown file and returns a new Model.
func LoadFromFile(path string) (*Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	return CreateModelFromLines(lines), nil
}

// CreateModelFromLines creates a new Model from a slice of lines, parsing them into blocks.
func CreateModelFromLines(lines []string) *Model {
	if len(lines) == 0 {
		return &Model{Blocks: []Block{{Type: TextBlock, Lines: []string{""}}}}
	}

	blocks := []Block{}
	currentBlock := Block{Type: TextBlock, Lines: []string{}}
	skipLeadingEmpty := false

	for _, line := range lines {
		if line == "$$" && currentBlock.Type == TextBlock {
			for len(currentBlock.Lines) > 0 && currentBlock.Lines[len(currentBlock.Lines)-1] == "" {
				currentBlock.Lines = currentBlock.Lines[:len(currentBlock.Lines)-1]
			}
			if len(currentBlock.Lines) > 0 {
				blocks = append(blocks, currentBlock)
			}
			currentBlock = Block{Type: MathBlock, Lines: []string{line}}
		} else if line == "$$" && currentBlock.Type == MathBlock {
			currentBlock.Lines = append(currentBlock.Lines, line)
			blocks = append(blocks, currentBlock)
			currentBlock = Block{Type: TextBlock, Lines: []string{}}
			skipLeadingEmpty = true
		} else {
			if skipLeadingEmpty && line == "" {
				continue
			}
			skipLeadingEmpty = false
			currentBlock.Lines = append(currentBlock.Lines, line)
		}
	}

	for len(currentBlock.Lines) > 0 && currentBlock.Lines[len(currentBlock.Lines)-1] == "" {
		currentBlock.Lines = currentBlock.Lines[:len(currentBlock.Lines)-1]
	}
	if len(currentBlock.Lines) > 0 {
		blocks = append(blocks, currentBlock)
	}

	if len(blocks) == 0 {
		blocks = []Block{{Type: TextBlock, Lines: []string{""}}}
	}

	return &Model{Blocks: blocks}
}
