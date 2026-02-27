package latex

import (
	"github.com/blacktop/go-termimg"
)

// max returns the larger of x or y.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// EncodeImageForKitty wraps an image in Kitty graphics protocol escape sequences,
// scaling it to a specific height in terminal rows.
func EncodeImageForKitty(pngPath string, targetRows int) (string, int, error) {
	image, err := termimg.Open(pngPath)
	if err != nil {
		return "", 0, err
	}

	// Use the target height, ensuring it's at least 1.
	rows := max(targetRows, 1)

	renderedString, err := image.
		Protocol(termimg.Kitty).
		Height(rows).
		Scale(termimg.ScaleStretch). // Force scale to the exact height
		Render()
	if err != nil {
		return "", 0, err
	}

	return renderedString, rows, nil
}
