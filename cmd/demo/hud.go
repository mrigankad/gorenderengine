package main

import (
	"fmt"
)

// DebugOverlay stores debug information for display
type DebugOverlay struct {
	lines []string
}

func (do *DebugOverlay) AddLine(format string, args ...interface{}) {
	do.lines = append(do.lines, fmt.Sprintf(format, args...))
}

func (do *DebugOverlay) Clear() {
	do.lines = do.lines[:0]
}

func (do *DebugOverlay) GetText() string {
	if len(do.lines) == 0 {
		return ""
	}
	var result string
	for _, line := range do.lines {
		result += line + "\n"
	}
	return result
}
