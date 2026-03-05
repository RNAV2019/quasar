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
	if !isMathEnvironment(processedMath) && !isLaTeXEnvironment(processedMath) {
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

func isLaTeXEnvironment(s string) bool {
	return strings.HasPrefix(s, "\\begin{")
}

// NeedsPDFPipeline returns true for content that uses PostScript specials
// (tikz, pgf, etc.) which dvipng cannot handle.
func NeedsPDFPipeline(s string) bool {
	return strings.Contains(s, "\\begin{tikzpicture}") ||
		strings.Contains(s, "\\begin{pgfpicture}") ||
		strings.Contains(s, "\\tikz")
}

// invertToWhiteOnTransparent converts a black-on-white image to white-on-transparent.
func invertToWhiteOnTransparent(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	dst := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := src.At(x, y).RGBA()
			// Luminance as alpha (white background → transparent, black content → opaque white)
			lum := (r*299 + g*587 + b*114) / 1000
			alpha := uint8(255 - lum>>8)
			dst.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: alpha})
		}
	}
	return dst
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

	tmpPngPath := filepath.Join(tmpDir, hashStr+".png")
	usedPDFPipeline := false

	if NeedsPDFPipeline(processedMath) {
		usedPDFPipeline = true
		// PDF pipeline for tikz/pgf content — dvipng can't handle PostScript specials,
		// and format files are DVI-mode, so use pdflatex with a full document.
		texContent := fmt.Sprintf(`\documentclass[border=2pt]{standalone}
\usepackage[T1]{fontenc}
\usepackage{lmodern}
\usepackage{amsmath}
\usepackage{amssymb}
\usepackage{tikz}
\usetikzlibrary{automata,positioning,arrows,calc,shapes,decorations.pathmorphing}
\usepackage{pgfplots}
\pgfplotsset{compat=1.18}
\begin{document}
%s
\end{document}
`, processedMath)

		texPath := filepath.Join(tmpDir, hashStr+".tex")
		if err := os.WriteFile(texPath, []byte(texContent), 0644); err != nil {
			return "", err
		}

		pdfPath := filepath.Join(tmpDir, hashStr+".pdf")
		latexCmd := exec.Command("pdflatex", "-interaction=nonstopmode",
			fmt.Sprintf("-output-directory=%s", tmpDir),
			texPath)
		output, err := latexCmd.CombinedOutput()
		if err != nil {
			logPath := filepath.Join(tmpDir, hashStr+".log")
			logData, _ := os.ReadFile(logPath)
			return "", fmt.Errorf("latex compilation failed: %w\nOutput: %s\nLog: %s", err, string(output), string(logData))
		}

		if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
			return "", fmt.Errorf("PDF file not found after compilation")
		}

		pdftoppmPrefix := filepath.Join(tmpDir, hashStr+"-out")
		pdftoppmCmd := exec.Command("pdftoppm",
			"-png", "-r", "2500",
			"-singlefile",
			pdfPath, pdftoppmPrefix)
		if output, err := pdftoppmCmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("pdftoppm failed: %w\nOutput: %s", err, string(output))
		}

		pdftoppmOut := pdftoppmPrefix + ".png"
		if _, err := os.Stat(pdftoppmOut); os.IsNotExist(err) {
			return "", fmt.Errorf("PNG file was not created by pdftoppm")
		}
		if err := os.Rename(pdftoppmOut, tmpPngPath); err != nil {
			return "", fmt.Errorf("failed to rename pdftoppm output: %w", err)
		}
	} else {
		texContent := fmt.Sprintf(`\begin{document}
%s
\end{document}
`, processedMath)

		texPath := filepath.Join(tmpDir, hashStr+".tex")
		if err := os.WriteFile(texPath, []byte(texContent), 0644); err != nil {
			return "", err
		}

		// DVI pipeline for regular math (faster)
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

	// PDF pipeline produces black-on-white; convert to white-on-transparent
	// to match the DVI pipeline's dvipng output.
	if usedPDFPipeline {
		srcImg = invertToWhiteOnTransparent(srcImg)
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
