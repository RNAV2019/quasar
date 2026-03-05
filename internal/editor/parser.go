package editor

import (
	"regexp"
	"strings"
)

// InlineMathRegion represents a region of inline math ($...$).
type InlineMathRegion struct {
	StartCol int
	EndCol   int
	Content  string // latex content without $ delimiters
}

// ParsedBlock represents a fully parsed block.
type ParsedBlock struct {
	Type            BlockType
	RawLines        []string        // Original lines
	HasFrontMatter  bool            // Block has YAML front matter
	FrontMatterEnd  int             // Line index where content starts (after ---)
	InlineMath      []InlineMathRegion // Inline math regions in content lines
	GlamourContent  string          // Content ready for glamour (stripped front matter, cleaned)
}

// Document represents a fully parsed document.
type Document struct {
	Blocks         []ParsedBlock
	GlobalMetadata *Metadata // YAML front matter (if present)
}

// inline math regex - matches $...$ but not $$ (which is block math)
// We handle the edge cases in the parsing logic rather than with complex regex
var inlineMathRe = regexp.MustCompile(`\$[^\$\n]+?\$`)

// ParseDocument parses blocks into a structured Document.
func ParseDocument(blocks []Block) *Document {
	doc := &Document{
		Blocks:         make([]ParsedBlock, len(blocks)),
		GlobalMetadata: nil,
	}

	for i, block := range blocks {
		doc.Blocks[i] = ParseBlock(block)
	}

	return doc
}

// ParseBlock parses a single block into a ParsedBlock.
func ParseBlock(block Block) ParsedBlock {
	pb := ParsedBlock{
		Type:           block.Type,
		RawLines:       make([]string, len(block.Lines)),
		HasFrontMatter: false,
		FrontMatterEnd: 0,
		InlineMath:     []InlineMathRegion{},
		GlamourContent: "",
	}
	copy(pb.RawLines, block.Lines)

	if block.Type != TextBlock {
		return pb
	}

	// Parse front matter and inline math from content
	content := strings.Join(block.Lines, "\n")
	pb.GlamourContent, pb.HasFrontMatter, pb.FrontMatterEnd = parseFrontMatter(content)

	// Parse inline math for all content lines
	pb.InlineMath = parseInlineMath(block.Lines, pb.FrontMatterEnd)

	return pb
}

// parseFrontMatter parses YAML front matter from content.
// Returns the content (stripped of front matter if present), whether front matter exists,
// and the line index where content starts (accounting for empty line after front matter).
func parseFrontMatter(content string) (string, bool, int) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 {
		return content, false, 0
	}

	// Check if first line is "---"
	if strings.TrimSpace(lines[0]) != "---" {
		return content, false, 0
	}

	// Find the closing "---"
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			// Skip past the closing "---" and any leading empty lines
			// to find where actual content starts
			contentStartIdx := i + 1
			for contentStartIdx < len(lines) && strings.TrimSpace(lines[contentStartIdx]) == "" {
				contentStartIdx++
			}
			remaining := strings.Join(lines[contentStartIdx:], "\n")
			return remaining, true, contentStartIdx
		}
	}

	return content, false, 0
}

// parseInlineMath parses inline math regions from lines starting at contentStartIdx
func parseInlineMath(lines []string, contentStartIdx int) []InlineMathRegion {
	var regions []InlineMathRegion

	// Only parse inline math from content lines (after front matter)
	startIdx := contentStartIdx
	if startIdx >= len(lines) {
		startIdx = 0
	}

	for lineIdx := startIdx; lineIdx < len(lines); lineIdx++ {
		line := lines[lineIdx]
		matches := inlineMathRe.FindAllStringIndex(line, -1)

		for _, match := range matches {
			// Extract content without $ delimiters
			content := line[match[0]+1 : match[1]-1]
			regions = append(regions, InlineMathRegion{
				StartCol: match[0],
				EndCol:   match[1],
				Content:  content,
			})
		}
	}

	return regions
}

// GetGlamourContent returns content ready for glamour rendering.
func (p *ParsedBlock) GetGlamourContent() string {
	return p.GlamourContent
}

// GetInlineMath returns inline math regions for a given line index within the block.
func (p *ParsedBlock) GetInlineMath(lineIdx int) []InlineMathRegion {
	// Check if lineIdx is within content lines (after front matter)
	if p.HasFrontMatter && lineIdx < p.FrontMatterEnd {
		return nil
	}

	// Adjust line index relative to content start
	adjustedLineIdx := lineIdx
	if p.HasFrontMatter {
		adjustedLineIdx = lineIdx - p.FrontMatterEnd
	}

	// Collect inline math regions that apply to this line
	// Since we store all inline math regions, we need to filter by adjusted line
	var result []InlineMathRegion

	// Get line count in content
	contentLines := p.RawLines
	if p.HasFrontMatter && p.FrontMatterEnd < len(contentLines) {
		contentLines = contentLines[p.FrontMatterEnd:]
	}

	if adjustedLineIdx < 0 || adjustedLineIdx >= len(contentLines) {
		return nil
	}

	// Re-parse this specific line for inline math
	line := contentLines[adjustedLineIdx]
	matches := inlineMathRe.FindAllStringIndex(line, -1)

	for _, match := range matches {
		content := line[match[0]+1 : match[1]-1]
		result = append(result, InlineMathRegion{
			StartCol: match[0],
			EndCol:   match[1],
			Content:  content,
		})
	}

	return result
}

// HasInlineMath checks if a specific line has inline math.
func (p *ParsedBlock) HasInlineMath(lineIdx int) bool {
	return len(p.GetInlineMath(lineIdx)) > 0
}

