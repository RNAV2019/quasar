package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/RNAV2019/quasar/internal/editor"
	"github.com/RNAV2019/quasar/internal/latex"
)

// processDirtyBlocks compiles math blocks and inline math that need rendering.
func (m *Model) processDirtyBlocks() tea.Cmd {
	var cmds []tea.Cmd

	for i := range m.Editor.Blocks {
		block := &m.Editor.Blocks[i]
		if !block.IsDirty {
			continue
		}

		switch block.Type {
		case editor.MathBlock:
			if m.Editor.Cursor.BlockIdx == i {
				continue
			}
			block.IsDirty = false
			block.IsLoading = true
			m.PendingRenders++
			blockIdx := i
			lines := make([]string, len(block.Lines))
			copy(lines, block.Lines)
			numLines := len(block.Lines)
			gen := m.fileGeneration
			cmds = append(cmds, func() tea.Msg {
				contentLines := lines
				if len(contentLines) >= 2 && contentLines[0] == "$$" && contentLines[len(contentLines)-1] == "$$" {
					contentLines = contentLines[1 : len(contentLines)-1]
				}
				start := 0
				for start < len(contentLines) && strings.TrimSpace(contentLines[start]) == "" {
					start++
				}
				content := strings.Join(contentLines[start:], "\n")
				if strings.TrimSpace(content) == "" {
					return BlockProcessedMsg{
						BlockIdx: blockIdx, ImageID: 0, ImageCols: 0,
						ImageHeight: 0, Error: nil, Source: "",
						Generation: gen,
					}
				}
				path, err := latex.CompileToPNG(content, m.Config.CacheDir, false)
				var info latex.ImageInfo
				if err == nil {
					info, err = latex.TransmitImageForKitty(path, numLines, 0)
				}
				return BlockProcessedMsg{
					BlockIdx: blockIdx, ImageID: info.ImageID, ImageCols: info.Cols,
					ImageHeight: info.Rows, Error: err, Source: content,
					Generation: gen,
				}
			})

		case editor.TextBlock:
			anyProcessed := false
			for lineIdx, line := range block.Lines {
				isCursorLine := m.Editor.Cursor.BlockIdx == i && m.Editor.Cursor.LineIdx == lineIdx && m.mode == Insert
				matches := inlineMathRe.FindAllStringIndex(line, -1)

				if isCursorLine {
					filtered := make([][]int, 0, len(matches))
					for _, match := range matches {
						if m.Editor.Cursor.Col < match[0] || m.Editor.Cursor.Col >= match[1] {
							filtered = append(filtered, match)
						}
					}
					matches = filtered
				}

				prefix := fmt.Sprintf("%d-%d-", i, lineIdx)
				for key, render := range m.InlineRenders {
					if strings.HasPrefix(key, prefix) {
						if render.ImageID != 0 {
							latex.DeleteImage(render.ImageID)
						}
						delete(m.InlineRenders, key)
					}
				}

				for _, match := range matches {
					m.PendingRenders++
					blockIdx := i
					lIdx := lineIdx
					start, end := match[0], match[1]
					content := line[start+1 : end-1]
					gen := m.fileGeneration
					cmds = append(cmds, func() tea.Msg {
						path, err := latex.CompileToPNG(content, m.Config.CacheDir, true)
						var info latex.ImageInfo
						if err == nil {
							info, err = latex.TransmitImageForKitty(path, 1, 0)
						}
						return InlineMathProcessedMsg{
							BlockIdx: blockIdx, LineIdx: lIdx, StartCol: start, EndCol: end,
							ImageID: info.ImageID, ImageCols: info.Cols,
							ImageHeight: info.Rows, Error: err, Generation: gen,
						}
					})
					anyProcessed = true
				}
			}
			if m.Editor.Cursor.BlockIdx != i || anyProcessed {
				block.IsDirty = false
			}
		}
	}
	return tea.Batch(cmds...)
}

// insertMathBlock creates a properly structured math block at cursor position.
func (m *Model) insertMathBlock() {
	blockIdx := m.Editor.Cursor.BlockIdx
	lineIdx := m.Editor.Cursor.LineIdx
	col := m.Editor.Cursor.Col

	block := &m.Editor.Blocks[blockIdx]
	line := block.Lines[lineIdx]
	runes := []rune(line)

	leftPart := string(runes[:col])
	rightPart := string(runes[col:])

	var leftBlockLines []string
	var rightBlockLines []string

	leftBlockLines = append(leftBlockLines, block.Lines[:lineIdx]...)
	if leftPart != "" {
		leftBlockLines = append(leftBlockLines, leftPart)
	}
	if rightPart != "" {
		rightBlockLines = append(rightBlockLines, rightPart)
	}
	rightBlockLines = append(rightBlockLines, block.Lines[lineIdx+1:]...)

	mathBlock := editor.Block{
		Type:    editor.MathBlock,
		Lines:   []string{"$$", "", "$$"},
		IsDirty: true,
	}

	var newBlocks []editor.Block

	for _, block := range m.Editor.Blocks[:blockIdx] {
		newBlocks = append(newBlocks, block)
	}

	if len(leftBlockLines) > 0 {
		newBlocks = append(newBlocks, editor.Block{
			Type:    editor.TextBlock,
			Lines:   leftBlockLines,
			IsDirty: true,
		})
	}

	newBlocks = append(newBlocks, mathBlock)

	if len(rightBlockLines) > 0 {
		newBlocks = append(newBlocks, editor.Block{
			Type:    editor.TextBlock,
			Lines:   rightBlockLines,
			IsDirty: true,
		})
	}

	for i := blockIdx + 1; i < len(m.Editor.Blocks); i++ {
		newBlocks = append(newBlocks, m.Editor.Blocks[i])
	}

	if len(newBlocks) == 0 {
		newBlocks = []editor.Block{
			{Type: editor.TextBlock, Lines: []string{""}},
		}
	}

	m.Editor.Blocks = newBlocks

	for i, b := range m.Editor.Blocks {
		if b.Type == editor.MathBlock && len(b.Lines) == 3 && b.Lines[0] == "$$" && b.Lines[2] == "$$" {
			m.Editor.Cursor.BlockIdx = i
			m.Editor.Cursor.LineIdx = 1
			m.Editor.Cursor.Col = 0
			break
		}
	}
}
