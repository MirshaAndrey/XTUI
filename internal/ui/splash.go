package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"xtui/internal/theme"
)

// Logo is a small ASCII banner shown on splash.
const Logo = `
   ██╗  ██╗████████╗██╗   ██╗██╗
   ╚██╗██╔╝╚══██╔══╝██║   ██║██║
    ╚███╔╝    ██║   ██║   ██║██║
    ██╔██╗    ██║   ██║   ██║██║
   ██╔╝ ██╗   ██║   ╚██████╔╝██║
   ╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚═╝
`

// RenderSplash renders the centered splash screen.
func RenderSplash(t theme.Theme, width, height int) string {
	style := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	tagline := lipgloss.NewStyle().
		Foreground(t.Dim).
		Italic(true).
		Render("a beautiful system monitor")

	content := style.Render(strings.TrimRight(Logo, "\n")) + "\n\n" +
		lipgloss.PlaceHorizontal(60, lipgloss.Center, tagline)

	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
