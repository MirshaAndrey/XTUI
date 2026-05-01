package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name      string
	Bg        lipgloss.Color
	Fg        lipgloss.Color
	Dim       lipgloss.Color
	Accent    lipgloss.Color
	Label     lipgloss.Color
	Value     lipgloss.Color
	BarEmpty  lipgloss.Color
	LevelLow  lipgloss.Color
	LevelMid  lipgloss.Color
	LevelHigh lipgloss.Color
	Border    lipgloss.Color
	TitleBg   lipgloss.Color
	TitleFg   lipgloss.Color
	AlertBg   lipgloss.Color
	AlertFg   lipgloss.Color
}

var Catppuccin = Theme{
	Name:      "catppuccin",
	Fg:        lipgloss.Color("#CDD6F4"),
	Dim:       lipgloss.Color("#6C7086"),
	Accent:    lipgloss.Color("#7D56F4"),
	Label:     lipgloss.Color("#FFD580"),
	Value:     lipgloss.Color("#A6E3A1"),
	BarEmpty:  lipgloss.Color("#313244"),
	LevelLow:  lipgloss.Color("#A6E3A1"),
	LevelMid:  lipgloss.Color("#F9E2AF"),
	LevelHigh: lipgloss.Color("#F38BA8"),
	Border:    lipgloss.Color("#7D56F4"),
	TitleBg:   lipgloss.Color("#7D56F4"),
	TitleFg:   lipgloss.Color("#FAFAFA"),
	AlertBg:   lipgloss.Color("#F38BA8"),
	AlertFg:   lipgloss.Color("#1E1E2E"),
}

var Dracula = Theme{
	Name:      "dracula",
	Fg:        lipgloss.Color("#F8F8F2"),
	Dim:       lipgloss.Color("#6272A4"),
	Accent:    lipgloss.Color("#BD93F9"),
	Label:     lipgloss.Color("#FFB86C"),
	Value:     lipgloss.Color("#50FA7B"),
	BarEmpty:  lipgloss.Color("#44475A"),
	LevelLow:  lipgloss.Color("#50FA7B"),
	LevelMid:  lipgloss.Color("#F1FA8C"),
	LevelHigh: lipgloss.Color("#FF5555"),
	Border:    lipgloss.Color("#BD93F9"),
	TitleBg:   lipgloss.Color("#BD93F9"),
	TitleFg:   lipgloss.Color("#282A36"),
	AlertBg:   lipgloss.Color("#FF5555"),
	AlertFg:   lipgloss.Color("#F8F8F2"),
}

var Nord = Theme{
	Name:      "nord",
	Fg:        lipgloss.Color("#ECEFF4"),
	Dim:       lipgloss.Color("#4C566A"),
	Accent:    lipgloss.Color("#88C0D0"),
	Label:     lipgloss.Color("#EBCB8B"),
	Value:     lipgloss.Color("#A3BE8C"),
	BarEmpty:  lipgloss.Color("#3B4252"),
	LevelLow:  lipgloss.Color("#A3BE8C"),
	LevelMid:  lipgloss.Color("#EBCB8B"),
	LevelHigh: lipgloss.Color("#BF616A"),
	Border:    lipgloss.Color("#88C0D0"),
	TitleBg:   lipgloss.Color("#5E81AC"),
	TitleFg:   lipgloss.Color("#ECEFF4"),
	AlertBg:   lipgloss.Color("#BF616A"),
	AlertFg:   lipgloss.Color("#ECEFF4"),
}

var Gruvbox = Theme{
	Name:      "gruvbox",
	Fg:        lipgloss.Color("#EBDBB2"),
	Dim:       lipgloss.Color("#928374"),
	Accent:    lipgloss.Color("#D79921"),
	Label:     lipgloss.Color("#FABD2F"),
	Value:     lipgloss.Color("#B8BB26"),
	BarEmpty:  lipgloss.Color("#3C3836"),
	LevelLow:  lipgloss.Color("#B8BB26"),
	LevelMid:  lipgloss.Color("#FABD2F"),
	LevelHigh: lipgloss.Color("#FB4934"),
	Border:    lipgloss.Color("#D79921"),
	TitleBg:   lipgloss.Color("#D79921"),
	TitleFg:   lipgloss.Color("#282828"),
	AlertBg:   lipgloss.Color("#FB4934"),
	AlertFg:   lipgloss.Color("#282828"),
}

var TokyoNight = Theme{
	Name:      "tokyonight",
	Fg:        lipgloss.Color("#C0CAF5"),
	Dim:       lipgloss.Color("#565F89"),
	Accent:    lipgloss.Color("#7AA2F7"),
	Label:     lipgloss.Color("#E0AF68"),
	Value:     lipgloss.Color("#9ECE6A"),
	BarEmpty:  lipgloss.Color("#1A1B26"),
	LevelLow:  lipgloss.Color("#9ECE6A"),
	LevelMid:  lipgloss.Color("#E0AF68"),
	LevelHigh: lipgloss.Color("#F7768E"),
	Border:    lipgloss.Color("#7AA2F7"),
	TitleBg:   lipgloss.Color("#7AA2F7"),
	TitleFg:   lipgloss.Color("#1A1B26"),
	AlertBg:   lipgloss.Color("#F7768E"),
	AlertFg:   lipgloss.Color("#1A1B26"),
}

var All = []Theme{Catppuccin, Dracula, Nord, Gruvbox, TokyoNight}

func ByName(name string) Theme {
	for _, t := range All {
		if t.Name == name {
			return t
		}
	}
	return Catppuccin
}

// Index returns the position of a theme in All (for cycling)
func Index(name string) int {
	for i, t := range All {
		if t.Name == name {
			return i
		}
	}
	return 0
}
