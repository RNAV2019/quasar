package latex

import (
	// "encoding/base64"
	// "fmt"
	"github.com/blacktop/go-termimg"
)

// EncodeImageForKitty wraps an image in Kitty graphics protocol escape sequences.
// We use the 'file' transmission mode (t=f) for efficiency, which tells Kitty
// to read the image directly from the specified path.
func EncodeImageForKitty(pngPath string) (string, int, error) {
	image, err := termimg.Open(pngPath)
	if err != nil {
		return "", 0, err
	}
	renderedString, err := image.
		Protocol(termimg.Kitty).
		Render()
	if err != nil {
		return "", 0, err
	}

	height := 1
	return renderedString, height, nil
}
