package ui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

const (
	BoxTop             = "╭─────────────────────────────────────────────────────────────╮"
	BoxBottom          = "╰─────────────────────────────────────────────────────────────╯"
	BoxDivider         = "├─────────────────────────────────────────────────────────────┤"
	BoxDivider2Col     = "├──────────────────────────────┬──────────────────────────────┤"
	BoxDivider2ColWide = "├───────────────────────┬─────────────────────────────────────┤"
	BoxDivider3Col     = "├───────────────────────┬──────────┬──────────────────────────┤"
	BoxDivider4Col     = "├──────────┬──────────────────┬──────────┬────────────────────┤"
)

func PrintBoxTop() {
	color.Cyan(BoxTop)
}

func PrintBoxBottom() {
	color.Cyan(BoxBottom)
}

func PrintBoxDivider() {
	color.Cyan(BoxDivider)
}

func PrintBoxDivider2Col() {
	color.Cyan(BoxDivider2Col)
}

func PrintBoxDivider2ColWide() {
	color.Cyan(BoxDivider2ColWide)
}

func PrintBoxDivider3Col() {
	color.Cyan(BoxDivider3Col)
}

func PrintBoxDivider4Col() {
	color.Cyan(BoxDivider4Col)
}

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(b)/float64(div), units[exp])
}

func RenderBar(pct float64, width int) string {
	if width <= 0 {
		width = 18
	}
	filled := int((pct / 100.0) * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 1 && pct > 0 {
		filled = 1
	}
	return color.GreenString(strings.Repeat("█", filled)) + color.HiBlackString(strings.Repeat("░", width-filled))
}

