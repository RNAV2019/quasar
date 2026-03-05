package editor

// TabWidth is the visual width of a tab character.
const TabWidth = 4

// VisualColToRuneCol converts a visual column position to a rune column position.
// Tabs expand to TabWidth spaces visually.
func VisualColToRuneCol(line string, visualCol int) int {
	runes := []rune(line)
	runeCol := 0
	currentVisualCol := 0

	for runeCol < len(runes) && currentVisualCol < visualCol {
		if runes[runeCol] == '\t' {
			currentVisualCol += TabWidth
		} else {
			currentVisualCol++
		}
		if currentVisualCol <= visualCol {
			runeCol++
		}
	}

	return runeCol
}

// RuneColToVisualCol converts a rune column position to a visual column position.
// Tabs expand to TabWidth spaces visually.
func RuneColToVisualCol(line string, runeCol int) int {
	runes := []rune(line)
	visualCol := 0

	for i := 0; i < runeCol && i < len(runes); i++ {
		if runes[i] == '\t' {
			visualCol += TabWidth
		} else {
			visualCol++
		}
	}

	return visualCol
}

// ExpandTabs converts tab characters to spaces for display.
func ExpandTabs(line string) string {
	runes := []rune(line)
	var result []rune
	for _, r := range runes {
		if r == '\t' {
			for range TabWidth {
				result = append(result, ' ')
			}
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
