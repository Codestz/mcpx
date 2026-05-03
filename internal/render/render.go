// Package render provides terminal-rendering primitives for mcpx's premium TUI:
// boxes, bars, sparklines, formatted numbers/durations, and width detection.
//
// All helpers are color-agnostic — pass already-colored strings if needed.
package render

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// TermWidth returns the current terminal width in columns. Falls back to 80
// for non-tty / piped output.
func TermWidth() int {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return 100
	}
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

// FormatNumber renders an integer with K/M/B suffix at one decimal place.
//   1234     →  "1.2K"
//   487231   →  "487K"
//   4800000  →  "4.8M"
func FormatNumber(n int64) string {
	abs := n
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case abs >= 1_000_000:
		v := float64(n) / 1_000_000
		if v >= 100 {
			return fmt.Sprintf("%.0fM", v)
		}
		return fmt.Sprintf("%.1fM", v)
	case abs >= 1_000:
		v := float64(n) / 1_000
		if v >= 100 {
			return fmt.Sprintf("%.0fK", v)
		}
		return fmt.Sprintf("%.1fK", v)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// FormatPercent renders a [0,1] float as a percent string with no decimals.
func FormatPercent(p float64) string {
	return fmt.Sprintf("%d%%", int(p*100+0.5))
}

// FormatDuration renders ms as the most natural unit:
//   <1000        → "412ms"
//   1000..59999  → "12s"
//   ≥60000       → "1m23s"
func FormatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		s := int(d.Seconds())
		return fmt.Sprintf("%ds", s)
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) - m*60
	return fmt.Sprintf("%dm%02ds", m, s)
}

// Bar renders a horizontal block-bar of width chars representing pct ∈ [0,1].
// Uses 8 fractional levels for sub-cell precision.
//
// Width 10, pct 0.42 →  "▓▓▓▓▎     "
func Bar(pct float64, width int) string {
	if width <= 0 {
		return ""
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	full := int(pct * float64(width))
	rem := pct*float64(width) - float64(full)
	out := strings.Repeat("█", full)
	if full < width {
		// 8-level partials
		idx := int(rem * 8)
		if idx > 0 {
			parts := []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}
			out += string(parts[idx])
			full++
		}
		out += strings.Repeat(" ", width-full)
	}
	return out
}

// Sparkline renders values into a single-row sparkline using 8 vertical block
// levels. Empty input returns an empty string.
//
//   [1,3,2,5,4]  →  "▁▄▃▇▅"
func Sparkline(values []int64) string {
	if len(values) == 0 {
		return ""
	}
	var max int64
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		return strings.Repeat("▁", len(values))
	}
	levels := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	var b strings.Builder
	for _, v := range values {
		pct := float64(v) / float64(max)
		idx := int(pct*7 + 0.0001)
		if idx > 7 {
			idx = 7
		}
		if idx < 0 {
			idx = 0
		}
		b.WriteRune(levels[idx])
	}
	return b.String()
}

// Truncate clips s to max chars adding an ellipsis if needed.
func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

// PadRight pads s with spaces on the right to fit width chars.
func PadRight(s string, width int) string {
	n := len([]rune(s))
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

// Box-drawing constants for the Premium TUI.
const (
	BoxTL = "╭"
	BoxTR = "╮"
	BoxBL = "╰"
	BoxBR = "╯"
	BoxH  = "─"
	BoxV  = "│"
	BoxT  = "┬"
	BoxB  = "┴"
	BoxX  = "┼"
	BoxL  = "├"
	BoxR  = "┤"
)
