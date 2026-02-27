package latex

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// isMathEnvironment checks if a string starts with a LaTeX environment that provides its own math mode.
func isMathEnvironment(s string) bool {
	// List of environments that create their own math mode
	envs := []string{"align", "align*", "equation", "equation*", "gather", "gather*", "multline", "multline*"}
	for _, env := range envs {
		if strings.HasPrefix(s, "\\begin{"+env+"}") {
			return true
		}
	}
	return false
}

func CompileToPNG(math string, cacheDir string) (string, error) {
	// Ensure the content is wrapped in a proper math environment if it isn't already.
	processedMath := strings.TrimSpace(math)
	if !isMathEnvironment(processedMath) {
		// Use gather* for centered, unnumbered display math. It's more stable than $$.
		processedMath = fmt.Sprintf("\\begin{gather*}%s\\end{gather*}", processedMath)
	}

	hash := sha256.Sum256([]byte(processedMath))
	hashStr := hex.EncodeToString(hash[:])
	pngPath := filepath.Join(cacheDir, hashStr+".png")

	if _, err := os.Stat(pngPath); err == nil {
		return pngPath, nil
	}

	tmpDir, err := os.MkdirTemp(cacheDir, "compile-*")
	if err != nil {
		return "", fmt.Errorf("Failed to create temporary directory: %w", err)
	}

	template := `\documentclass[preview,border=2pt]{standalone}
	\usepackage[T1]{fontenc}
	\usepackage{lmodern}
	\usepackage{amsmath}
	\usepackage{amssymb}
	\setlength{\abovedisplayskip}{0pt}
	\setlength{\belowdisplayskip}{0pt}
	\begin{document}
	%s
	\end{document}
	`
	texContent := fmt.Sprintf(template, processedMath)
	texPath := filepath.Join(tmpDir, hashStr+".tex")

	if err := os.WriteFile(texPath, []byte(texContent), 0644); err != nil {
		return "", err
	}

	// --- New Pipeline: pdflatex -> pdfcrop -> convert ---

	// 1. Run pdflatex
	pdfLatexCmd := exec.Command("pdflatex", "-interaction=nonstopmode", "-output-directory="+tmpDir, texPath)
	if output, err := pdfLatexCmd.CombinedOutput(); err != nil {
		logPath := filepath.Join(tmpDir, hashStr+".log")
		logData, _ := os.ReadFile(logPath)
		return "", fmt.Errorf("pdflatex compilation failed.\nError: %w\nOutput: %s\nLog: %s", err, string(output), string(logData))
	}

	pdfPath := filepath.Join(tmpDir, hashStr+".pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return "", fmt.Errorf("PDF file not found after successful pdflatex compilation")
	}

	// 2. Run pdfcrop
	croppedPdfPath := filepath.Join(tmpDir, hashStr+"-crop.pdf")
	pdfCropCmd := exec.Command("pdfcrop", "--margins", "10", pdfPath, croppedPdfPath)
	if _, err := pdfCropCmd.CombinedOutput(); err != nil {
		// pdfcrop can be noisy; we don't treat its failure as fatal.
		// We'll just use the uncropped PDF as a fallback.
		croppedPdfPath = pdfPath
	}
	if _, err := os.Stat(croppedPdfPath); os.IsNotExist(err) {
		croppedPdfPath = pdfPath
	}

	// 3. Run convert (ImageMagick) to create the final PNG
	//    -transparent white: makes the white background transparent
	//    -negate: inverts the colors, turning the black text to white
	convertCmd := exec.Command("magick", "-density", "1800", croppedPdfPath, "-background", "transparent", "-fill", "white", "-opaque", "black", pngPath)
	if output, err := convertCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ImageMagick convert execution failed.\nError: %w\nOutput: %s", err, string(output))
	}

	return pngPath, nil
}
