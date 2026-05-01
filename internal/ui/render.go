package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"xtui/internal/theme"
)

// HumanBytes returns a human-friendly byte size string.
func HumanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// HumanRate formats bytes-per-second.
func HumanRate(bps uint64) string {
	return HumanBytes(bps) + "/s"
}

// FormatUptime formats seconds as a duration string.
func FormatUptime(sec uint64) string {
	d := sec / 86400
	h := (sec % 86400) / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if d > 0 {
		return fmt.Sprintf("%dd %02d:%02d:%02d", d, h, m, s)
	}
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// LevelColor picks a color for a load percentage.
func LevelColor(t theme.Theme, percent float64) lipgloss.Color {
	switch {
	case percent < 50:
		return t.LevelLow
	case percent < 80:
		return t.LevelMid
	default:
		return t.LevelHigh
	}
}

// Bar renders a solid progress bar of given width with percentage label.
func Bar(t theme.Theme, percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int(float64(width) * percent / 100.0)
	empty := width - filled
	color := LevelColor(t, percent)

	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(t.BarEmpty)

	bar := filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty))

	pctStyle := lipgloss.NewStyle().Bold(true).Foreground(color)
	return fmt.Sprintf("%s  %s", bar, pctStyle.Render(fmt.Sprintf("%5.1f%%", percent)))
}

// GradientBar renders a bar where each cell takes a color along the
// low → mid → high gradient based on its position. Visually richer than Bar.
func GradientBar(t theme.Theme, percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int(float64(width) * percent / 100.0)
	empty := width - filled

	emptyStyle := lipgloss.NewStyle().Foreground(t.BarEmpty)
	var b strings.Builder
	for i := 0; i < filled; i++ {
		// position 0..1 along the FULL bar — gives a stable gradient regardless of fill level
		pos := float64(i) / float64(width-1)
		c := gradientColor(t, pos)
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render("█"))
	}
	b.WriteString(emptyStyle.Render(strings.Repeat("░", empty)))

	tip := LevelColor(t, percent)
	pctStyle := lipgloss.NewStyle().Bold(true).Foreground(tip)
	return fmt.Sprintf("%s  %s", b.String(), pctStyle.Render(fmt.Sprintf("%5.1f%%", percent)))
}

// gradientColor interpolates between low/mid/high theme colors based on position 0..1.
func gradientColor(t theme.Theme, pos float64) lipgloss.Color {
	if pos < 0.5 {
		// low → mid
		return lerpHex(string(t.LevelLow), string(t.LevelMid), pos*2)
	}
	// mid → high
	return lerpHex(string(t.LevelMid), string(t.LevelHigh), (pos-0.5)*2)
}

// lerpHex linearly interpolates between two #RRGGBB color strings.
func lerpHex(a, b string, t float64) lipgloss.Color {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	ar, ag, ab := parseHex(a)
	br, bg, bb := parseHex(b)
	r := int(float64(ar) + (float64(br)-float64(ar))*t)
	g := int(float64(ag) + (float64(bg)-float64(ag))*t)
	bl := int(float64(ab) + (float64(bb)-float64(ab))*t)
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, bl))
}

func parseHex(s string) (r, g, b int) {
	if len(s) == 7 && s[0] == '#' {
		fmt.Sscanf(s[1:], "%02x%02x%02x", &r, &g, &b)
	}
	return
}

// Sparkline renders a series of values 0..max as block characters.
// If max is 0, auto-scales to the max value in the series (with a floor of 1).
func Sparkline(t theme.Theme, values []float64, max float64) string {
	if len(values) == 0 {
		return ""
	}
	if max <= 0 {
		// auto-scale
		for _, v := range values {
			if v > max {
				max = v
			}
		}
		if max < 1 {
			max = 1
		}
	}
	levels := []rune("▁▂▃▄▅▆▇█")
	var b strings.Builder
	for _, v := range values {
		if v < 0 {
			v = 0
		}
		if v > max {
			v = max
		}
		idx := int(v / max * float64(len(levels)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(levels) {
			idx = len(levels) - 1
		}
		// color by relative position
		c := LevelColor(t, v/max*100)
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render(string(levels[idx])))
	}
	return b.String()
}

// History is a fixed-capacity ring buffer for sparkline data.
type History struct {
	data []float64
	cap  int
}

func NewHistory(capacity int) *History {
	return &History{data: make([]float64, 0, capacity), cap: capacity}
}

func (h *History) Push(v float64) {
	if len(h.data) >= h.cap {
		h.data = h.data[1:]
	}
	h.data = append(h.data, v)
}

func (h *History) Values() []float64 {
	return h.data
}

func (h *History) Last() float64 {
	if len(h.data) == 0 {
		return 0
	}
	return h.data[len(h.data)-1]
}

// TruncateMiddle shortens a string to width, inserting … in the middle.
func TruncateMiddle(s string, width int) string {
	if len([]rune(s)) <= width {
		return s
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	r := []rune(s)
	half := (width - 1) / 2
	return string(r[:half]) + "…" + string(r[len(r)-(width-half-1):])
}

// PadRight pads or truncates a string to exactly width visible runes.
func PadRight(s string, width int) string {
	r := []rune(s)
	if len(r) > width {
		return TruncateMiddle(s, width)
	}
	return s + strings.Repeat(" ", width-len(r))
}
