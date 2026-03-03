package latex

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/blacktop/go-termimg"
)

const placeholderChar = "\U0010EEEE"

var kittyDiacritics = []rune{
	0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F, 0x0346, 0x034A,
	0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357, 0x035B, 0x0363, 0x0364, 0x0365,
	0x0366, 0x0367, 0x0368, 0x0369, 0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F,
	0x0483, 0x0484, 0x0485, 0x0486, 0x0487, 0x0592, 0x0593, 0x0594, 0x0595, 0x0597,
	0x0598, 0x0599, 0x059C, 0x059D, 0x059E, 0x059F, 0x05A0, 0x05A1, 0x05A8, 0x05A9,
	0x05AB, 0x05AC, 0x05AF, 0x05C4, 0x0610, 0x0611, 0x0612, 0x0613, 0x0614, 0x0615,
	0x0616, 0x0617, 0x0653, 0x0654, 0x0657, 0x0658, 0x0659, 0x065A, 0x065B, 0x065D,
	0x065E, 0x06D6, 0x06D7, 0x06D8, 0x06D9, 0x06DA, 0x06DB, 0x06DC, 0x06DF, 0x06E0,
	0x06E1, 0x06E2, 0x06E4, 0x06E7, 0x06E8, 0x06EB, 0x06EC,
}

var virtualPlacementRe = regexp.MustCompile(`a=p,U=1,i=\d+,c=(\d+),r=(\d+)`)

func diacritic(pos uint16) rune {
	if int(pos) >= len(kittyDiacritics) {
		return kittyDiacritics[0]
	}
	return kittyDiacritics[pos]
}

func placeholderColorStart(imageID uint32) string {
	r := (imageID >> 16) & 0xFF
	g := (imageID >> 8) & 0xFF
	b := imageID & 0xFF
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

func placeholderCell(row, col uint16, idExtra byte) string {
	var b strings.Builder
	b.WriteString(placeholderChar)
	b.WriteRune(diacritic(row))
	b.WriteRune(diacritic(col))
	b.WriteRune(diacritic(uint16(idExtra)))
	return b.String()
}

type ImageInfo struct {
	ImageID uint32
	Rows    int
	Cols    int
}

func TransmitImageForKitty(pngPath string, targetRows, targetCols int) (ImageInfo, error) {
	image, err := termimg.Open(pngPath)
	if err != nil {
		return ImageInfo{}, err
	}

	image.Protocol(termimg.Kitty)
	image.UseUnicode(true)

	if targetRows > 0 {
		image.Height(targetRows)
	}
	if targetCols > 0 {
		image.Width(targetCols)
	}

	// Use ScaleFit for all cases to maintain aspect ratio
	image.Scale(termimg.ScaleFit)

	rendered, err := image.Render()
	if err != nil {
		return ImageInfo{}, err
	}

	renderer, err := image.GetRenderer()
	if err != nil {
		return ImageInfo{}, err
	}

	type idGetter interface {
		GetLastImageID() uint32
	}
	kr, ok := renderer.(idGetter)
	if !ok {
		return ImageInfo{}, fmt.Errorf("renderer does not support GetLastImageID")
	}
	imageID := kr.GetLastImageID()

	cols, rows := extractPlacementDimensions(rendered, targetRows)

	transmitEnd := strings.Index(rendered, placeholderChar)
	if transmitEnd > 0 {
		colorStartIdx := strings.LastIndex(rendered[:transmitEnd], "\x1b[38;2;")
		if colorStartIdx > 0 {
			os.Stdout.WriteString(rendered[:colorStartIdx])
		} else {
			os.Stdout.WriteString(rendered[:transmitEnd])
		}
	} else {
		os.Stdout.WriteString(rendered)
	}

	return ImageInfo{ImageID: imageID, Rows: rows, Cols: cols}, nil
}

func extractPlacementDimensions(rendered string, fallbackRows int) (cols, rows int) {
	matches := virtualPlacementRe.FindStringSubmatch(rendered)
	if len(matches) == 3 {
		c, err1 := strconv.Atoi(matches[1])
		r, err2 := strconv.Atoi(matches[2])
		if err1 == nil && err2 == nil {
			return c, r
		}
	}
	return 40, fallbackRows
}

func PlaceholderString(imageID uint32, rows, cols int) string {
	if rows <= 0 || cols <= 0 {
		return ""
	}

	idExtra := byte(imageID >> 24)
	colorStart := placeholderColorStart(imageID)

	var b strings.Builder
	b.WriteString(colorStart)
	for row := range rows {
		first := placeholderCell(uint16(row), 0, idExtra)
		b.WriteString(first)
		for col := 1; col < cols; col++ {
			b.WriteString(placeholderChar)
		}
		if row < rows-1 {
			b.WriteString("\x1b[39m\n")
			b.WriteString(colorStart)
		}
	}
	b.WriteString("\x1b[39m")
	return b.String()
}

func PlaceholderRow(imageID uint32, row uint16, cols int) string {
	if cols <= 0 {
		return ""
	}

	idExtra := byte(imageID >> 24)
	colorStart := placeholderColorStart(imageID)

	var b strings.Builder
	b.WriteString(colorStart)
	first := placeholderCell(row, 0, idExtra)
	b.WriteString(first)
	for col := 1; col < cols; col++ {
		b.WriteString(placeholderChar)
	}
	b.WriteString("\x1b[39m")
	return b.String()
}

func DeleteImage(imageID uint32) {
	seq := fmt.Sprintf("\x1b_Ga=d,d=i,i=%d,q=2\x1b\\", imageID)
	os.Stdout.WriteString(seq)
}

func DeleteAllImages() {
	os.Stdout.WriteString("\x1b_Ga=d,d=A\x1b\\")
}
