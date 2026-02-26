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

func IsMath(line string) bool {
	line = strings.TrimSpace(line)
	return (strings.HasPrefix(line, "$") && strings.HasSuffix(line, "$")) ||
		(strings.HasPrefix(line, "$$") && strings.HasSuffix(line, "$$"))
}

func CompileToPNG(math string, cacheDir string) (string, error) {
	hash := sha256.Sum256([]byte(math))
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
	\begin{document}
	%s
	\end{document}
	`
	texContent := fmt.Sprintf(template, math)
	texPath := filepath.Join(tmpDir, hashStr+".tex")

	if err := os.WriteFile(texPath, []byte(texContent), 0644); err != nil {
		return "", err
	}

	latexCmd := exec.Command("latex", "-interaction=nonstopmode", hashStr+".tex")
	latexCmd.Dir = tmpDir
	if err := latexCmd.Run(); err != nil {
		return "", fmt.Errorf("Latex error: %w", err)
	}

	dviPath := filepath.Join(tmpDir, hashStr+".dvi")
	tmpPngPath := filepath.Join(tmpDir, hashStr+".png")
	dviPngCmd := exec.Command("dvipng", "-Q", "9", "-D", "1200", "-T", "tight", "-bg", "Transparent", "-fg", "White", "-o", tmpPngPath, dviPath)
	dviPngCmd.Dir = tmpDir

	if err := dviPngCmd.Run(); err != nil {
		return "", fmt.Errorf("DVIPNG error: %w", err)
	}

	magickCmd := exec.Command("magick", "convert", tmpPngPath, "-resize", "30%", pngPath)
	if err := magickCmd.Run(); err != nil {
		return "", fmt.Errorf("ImageMagick error: %w", err)
	}

	return pngPath, nil
}
