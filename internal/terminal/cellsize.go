// Package terminal provides utilities for querying terminal properties.
package terminal

import (
	"os"

	"golang.org/x/sys/unix"
)

// CellSize holds the pixel dimensions of a single terminal cell.
type CellSize struct {
	WidthPx  int
	HeightPx int
}

// GetCellSize queries the terminal for cell pixel dimensions using TIOCGWINSZ.
// Returns reasonable defaults (8x16) if the query fails or returns zero values.
func GetCellSize() CellSize {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Xpixel == 0 || ws.Ypixel == 0 || ws.Col == 0 || ws.Row == 0 {
		return CellSize{WidthPx: 8, HeightPx: 16}
	}
	return CellSize{
		WidthPx:  int(ws.Xpixel) / int(ws.Col),
		HeightPx: int(ws.Ypixel) / int(ws.Row),
	}
}
