package ui

import (
	"fmt"
	"regexp"

	"github.com/RNAV2019/quasar/internal/editor"
)

// linkRegex matches markdown links and images.
var linkRegex = regexp.MustCompile(`!?\[([^\]]*)\]\(([^)]+)\)`)

// getLinkAtPosition returns the URL if a link is at the given screen position.
func (m Model) getLinkAtPosition(x, y int) string {
	gutterWidth := 0
	totalLines := 0
	for _, block := range m.Editor.Blocks {
		totalLines += len(block.Lines)
	}
	if totalLines > 0 {
		gutterWidth = len(fmt.Sprint(totalLines))
	}

	fileTreeOffset := 0
	if m.ShowFileTree {
		fileTreeOffset = 25
	}

	contentStartX := 2 + gutterWidth + 3 + fileTreeOffset
	contentStartY := 0

	if x < contentStartX {
		return ""
	}

	relX := x - contentStartX
	relY := y - contentStartY

	offsetAbsLine := 0
	for i := 0; i < m.Editor.Offset.BlockIdx && i < len(m.Editor.Blocks); i++ {
		offsetAbsLine += len(m.Editor.Blocks[i].Lines)
	}
	offsetAbsLine += m.Editor.Offset.LineIdx

	absLine := relY + offsetAbsLine

	currentAbsLine := 0
	for _, block := range m.Editor.Blocks {
		for _, line := range block.Lines {
			if currentAbsLine == absLine {
				return m.extractLinkAtColumn(line, relX)
			}
			currentAbsLine++
		}
	}

	return ""
}

// extractLinkAtColumn finds a link at the given visual column and returns its URL.
func (m Model) extractLinkAtColumn(line string, visualCol int) string {
	runes := []rune(line)

	matches := linkRegex.FindAllStringSubmatchIndex(string(runes), -1)

	for _, match := range matches {
		if len(match) >= 6 {
			linkStart := match[0]
			linkEnd := match[1]
			urlStart := match[4]
			urlEnd := match[5]

			visualStart := editor.RuneColToVisualCol(line, linkStart)
			visualEnd := editor.RuneColToVisualCol(line, linkEnd)

			if visualCol >= visualStart && visualCol < visualEnd {
				return string(runes[urlStart:urlEnd])
			}
		}
	}

	return ""
}
