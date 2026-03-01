package latex

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// PNGPath returns the path where the PNG for the given math content would be cached.
func PNGPath(math string, cacheDir string, isInline bool) string {
	processedMath := strings.TrimSpace(math)
	if !isMathEnvironment(processedMath) {
		processedMath = fmt.Sprintf("\\begin{gather*}%s\\end{gather*}", processedMath)
	}
	hash := sha256.Sum256([]byte(processedMath + fmt.Sprintf("%v", isInline)))
	hashStr := hex.EncodeToString(hash[:])
	return filepath.Join(cacheDir, hashStr+".png")
}

// CleanupUnusedPNGs removes PNG files that are no longer referenced by any current math content.
func CleanupUnusedPNGs(cacheDir string, currentMathPaths map[string]bool) error {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".png") {
			continue
		}
		path := filepath.Join(cacheDir, entry.Name())
		if !currentMathPaths[path] {
			os.Remove(path)
		}
	}
	return nil
}

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

func CompileToPNG(math string, cacheDir string, isInline bool) (string, error) {
	// Ensure the content is wrapped in a proper math environment if it isn't already.
	processedMath := strings.TrimSpace(math)
	if !isMathEnvironment(processedMath) {
		// Use gather* for centered, unnumbered display math. It's more stable than $$.
		processedMath = fmt.Sprintf("\\begin{gather*}%s\\end{gather*}", processedMath)
	}

	hash := sha256.Sum256([]byte(processedMath + fmt.Sprintf("%v", isInline)))
	hashStr := hex.EncodeToString(hash[:])
	pngPath := filepath.Join(cacheDir, hashStr+".png")

	// Check if PNG already exists
	if _, err := os.Stat(pngPath); err == nil {
		return pngPath, nil
	}

	// Serialize compilation for the same content to avoid duplicate compile folders
	lock := getCompileLock(hashStr)
	lock.Lock()
	defer lock.Unlock()

	// Check again after acquiring lock (another goroutine may have just finished)
	if _, err := os.Stat(pngPath); err == nil {
		return pngPath, nil
	}

	tmpDir, err := os.MkdirTemp(cacheDir, "compile-*")
	if err != nil {
		return "", fmt.Errorf("Failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	margins := "10 10 10 10"
	if isInline {
		margins = "0"
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

	croppedPdfPath := filepath.Join(tmpDir, hashStr+"-crop.pdf")
	pdfCropCmd := exec.Command("pdfcrop", "--margins", margins, pdfPath, croppedPdfPath)
	if _, err := pdfCropCmd.CombinedOutput(); err != nil {
		croppedPdfPath = pdfPath
	}
	if _, err := os.Stat(croppedPdfPath); os.IsNotExist(err) {
		croppedPdfPath = pdfPath
	}

	convertCmd := exec.Command("magick", "-density", "2500", croppedPdfPath, "-background", "transparent", "-fill", "white", "-opaque", "black", pngPath)
	if output, err := convertCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ImageMagick convert execution failed.\nError: %w\nOutput: %s", err, string(output))
	}

	return pngPath, nil
}
