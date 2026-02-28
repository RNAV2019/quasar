package latex

import (
	"github.com/blacktop/go-termimg"
)

// EncodeImageForKitty wraps an image in Kitty graphics protocol escape sequences.
func EncodeImageForKitty(pngPath string, targetRows int) (string, int, error) {
	image, err := termimg.Open(pngPath)
	if err != nil {
		return "", 0, err
	}

	image.Protocol(termimg.Kitty)

	if targetRows > 0 {
		image.Height(targetRows)
		if targetRows == 1 {
			// Proportional scaling for inline math to avoid distortion
			image.Scale(termimg.ScaleFit)
		} else {
			// Precise height matching for multiline math blocks
			image.Scale(termimg.ScaleStretch)
		}
	} else {
		image.Scale(termimg.ScaleFit)
	}

	renderedString, err := image.Render()
	if err != nil {
		return "", 0, err
	}

	return renderedString, targetRows, nil
}
