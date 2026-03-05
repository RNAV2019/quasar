// Package latex compiles LaTeX math expressions to PNG images and transmits
// them to the terminal via the Kitty graphics protocol.
package latex

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	compileLocks   = make(map[string]*sync.Mutex)
	compileLocksMu sync.Mutex
)

func getCompileLock(hash string) *sync.Mutex {
	compileLocksMu.Lock()
	defer compileLocksMu.Unlock()
	if _, ok := compileLocks[hash]; !ok {
		compileLocks[hash] = &sync.Mutex{}
	}
	return compileLocks[hash]
}

func sanitizeMath(math string, isInline bool) string {
	processedMath := strings.TrimSpace(math)
	lines := strings.Split(processedMath, "\n")
	var filtered []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			filtered = append(filtered, l)
		}
	}
	processedMath = strings.Join(filtered, "\n")
	if !isMathEnvironment(processedMath) {
		if isInline {
			processedMath = fmt.Sprintf("$%s$", processedMath)
		} else {
			processedMath = fmt.Sprintf("\\begin{gather*}%s\\end{gather*}", processedMath)
		}
	}
	return processedMath
}

func isMathEnvironment(s string) bool {
	envs := []string{"align", "align*", "equation", "equation*", "gather", "gather*", "multline", "multline*"}
	for _, env := range envs {
		if strings.HasPrefix(s, "\\begin{"+env+"}") {
			return true
		}
	}
	return false
}

func addTransparentPadding(src image.Image, top, right, bottom, left int) image.Image {
	bounds := src.Bounds()
	newW := bounds.Dx() + left + right
	newH := bounds.Dy() + top + bottom
	dst := image.NewNRGBA(image.Rect(0, 0, newW, newH))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.NRGBA{0, 0, 0, 0}), image.Point{}, draw.Src)
	draw.Draw(dst, image.Rect(left, top, left+bounds.Dx(), top+bounds.Dy()), src, bounds.Min, draw.Over)
	return dst
}

// CompileToPNG compiles a LaTeX math expression to a cached PNG file and returns its path.
func CompileToPNG(math string, cacheDir string, isInline bool) (string, error) {
	processedMath := sanitizeMath(math, isInline)

	hash := sha256.Sum256([]byte(processedMath + fmt.Sprintf("%v", isInline)))
	hashStr := hex.EncodeToString(hash[:])
	pngPath := filepath.Join(cacheDir, hashStr+".png")

	if _, err := os.Stat(pngPath); err == nil {
		return pngPath, nil
	}

	lock := getCompileLock(hashStr)
	lock.Lock()
	defer lock.Unlock()

	if _, err := os.Stat(pngPath); err == nil {
		return pngPath, nil
	}

	formatName := "quasar-math-multi"
	if isInline {
		formatName = "quasar-math-inline"
	}

	fmtPath := filepath.Join(cacheDir, formatName+".fmt")
	if _, err := os.Stat(fmtPath); os.IsNotExist(err) {
		return "", fmt.Errorf("LaTeX format file not found - please restart quasar")
	}

	tmpDir, err := os.MkdirTemp(cacheDir, "compile-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	texContent := fmt.Sprintf(`\begin{document}
%s
\end{document}
`, processedMath)

	texPath := filepath.Join(tmpDir, hashStr+".tex")
	if err := os.WriteFile(texPath, []byte(texContent), 0644); err != nil {
		return "", err
	}

	dviPath := filepath.Join(tmpDir, hashStr+".dvi")
	latexCmd := exec.Command("pdftex", "-output-mode=dvi", "-interaction=nonstopmode",
		fmt.Sprintf("-output-directory=%s", tmpDir),
		fmt.Sprintf("-fmt=%s", fmtPath),
		texPath)
	output, err := latexCmd.CombinedOutput()
	if err != nil {
		logPath := filepath.Join(tmpDir, hashStr+".log")
		logData, _ := os.ReadFile(logPath)
		return "", fmt.Errorf("latex compilation failed: %w\nOutput: %s\nLog: %s", err, string(output), string(logData))
	}

	if _, err := os.Stat(dviPath); os.IsNotExist(err) {
		return "", fmt.Errorf("DVI file not found after compilation")
	}

	tmpPngPath := filepath.Join(tmpDir, hashStr+".png")
	dvipngCmd := exec.Command("dvipng",
		"-D", "2500",
		"-T", "tight",
		"-bg", "Transparent",
		"-fg", "rgb 1.0 1.0 1.0",
		"-o", tmpPngPath,
		dviPath)
	if output, err := dvipngCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("dvipng failed: %w\nOutput: %s", err, string(output))
	}

	if _, err := os.Stat(tmpPngPath); os.IsNotExist(err) {
		return "", fmt.Errorf("PNG file was not created at %s", tmpPngPath)
	}

	f, err := os.Open(tmpPngPath)
	if err != nil {
		return "", fmt.Errorf("failed to open temporary PNG: %w", err)
	}
	srcImg, err := png.Decode(f)
	f.Close()
	if err != nil {
		return "", fmt.Errorf("failed to decode temporary PNG: %w", err)
	}

	var padded image.Image
	if isInline {
		padded = addTransparentPadding(srcImg, 4, 4, 4, 4)
	} else {
		w := srcImg.Bounds().Dx()
		vPad := w * 40 / 100 / 2
		hPad := w * 10 / 100 / 2
		if vPad < 60 {
			vPad = 60
		}
		if hPad < 30 {
			hPad = 30
		}
		padded = addTransparentPadding(srcImg, vPad, hPad, vPad, hPad)
	}

	outFile, err := os.Create(pngPath)
	if err != nil {
		return "", fmt.Errorf("failed to create PNG file: %w", err)
	}
	if err := png.Encode(outFile, padded); err != nil {
		outFile.Close()
		return "", fmt.Errorf("failed to encode padded PNG: %w", err)
	}
	outFile.Close()

	return pngPath, nil
}
