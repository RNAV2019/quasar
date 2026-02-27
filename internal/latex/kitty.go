package latex

import (
	// "encoding/base64"
	// "fmt"
	// "fmt"
	"math"

	"github.com/blacktop/go-termimg"
)

// EncodeImageForKitty wraps an image in Kitty graphics protocol escape sequences.
func EncodeImageForKitty(pngPath string) (string, int, error) {
	image, err := termimg.Open(pngPath)
	if err != nil {
		return "", 0, err
	}

	imageHeight := image.Bounds.Dy()

	termFeatures := termimg.QueryTerminalFeatures()
	characterHeight := termFeatures.FontHeight
	if characterHeight == 0 {
		characterHeight = 16 // Fallback
	}

	rows := int(math.Round(float64(imageHeight) / float64(characterHeight) / 6))
	rows = max(rows, 1)

	renderedString, err := image.
		Protocol(termimg.Kitty).
		Height(rows).
		Scale(termimg.ScaleFit).
		Render()
	if err != nil {
		return "", 0, err
	}

	return renderedString, rows, nil
}
