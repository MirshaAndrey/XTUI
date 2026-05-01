package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"xtui/internal/metrics"
	"xtui/internal/theme"
)

// Tab indices.
const (
	TabOverview = iota
	TabCPU
	TabMemory
	TabNetwork
	TabDisk
	TabProcesses
)

var TabNames = []string{"Overview", "CPU", "Memory", "Network", "Disk", "Processes"}

// Histories bundles per-metric ring buffers used for sparklines.
type Histories struct {
	CPU     *History
	Mem     *History
	Swap    *History
	NetRx   *History
	NetTx   *History
	PerCore []*History
}

func NewHistories(capacity int) *Histories {
	return &Histories{
		CPU:   NewHistory(capacity),
		Mem:   NewHistory(capacity),
		Swap:  NewHistory(capacity),
		NetRx: NewHistory(capacity),
		NetTx: NewHistory(capacity),
	}
}

// Push appends current snapshot values to all relevant histories.
func (h *Histories) Push(s metrics.Snapshot) {
	h.CPU.Push(s.CPUTotal)
	h.Mem.Push(s.MemPct)
	h.Swap.Push(s.SwapPct)
	if len(s.Net) > 0 {
		h.NetRx.Push(float64(s.Net[0].RxRate))
		h.NetTx.Push(float64(s.Net[0].TxRate))
	} else {
		h.NetRx.Push(0)
		h.NetTx.Push(0)
	}
	// Resize per-core buffers if cpu count changed
	if len(h.PerCore) != len(s.CPUPerCPU) {
		h.PerCore = make([]*History, len(s.CPUPerCPU))
		for i := range h.PerCore {
			h.PerCore[i] = NewHistory(h.CPU.cap)
		}
	}
	for i, v := range s.CPUPerCPU {
		h.PerCore[i].Push(v)
	}
}

// Renderer holds shared style state for view rendering.
type Renderer struct {
	Theme    theme.Theme
	Width    int
	Height   int
	BarWidth int
}

func (r Renderer) labelStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(r.Theme.Label)
}
func (r Renderer) valueStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(r.Theme.Value)
}
func (r Renderer) dimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(r.Theme.Dim)
}

// boxWithTitle wraps content in a rounded border with a title embedded in the top edge.
// Renders as:  ╭─ Title ──────╮
//
//	│ content      │
//	╰──────────────╯
func (r Renderer) boxWithTitle(title, content string) string {
	return r.boxWithTitleWidth(title, content, 0)
}

// boxWithTitleWidth is like boxWithTitle but forces an exact outer width
// (including borders). Pass 0 to use the natural width of the content.
// Use this when you need two boxes to line up in side-by-side columns.
func (r Renderer) boxWithTitleWidth(title, content string, width int) string {
	border := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		Border(border).
		BorderForeground(r.Theme.Border).
		Padding(0, 1)
	if width > 0 {
		// Width() in lipgloss sets the *content* area width; subtract borders+padding.
		inner := width - 4 // 2 border chars + 2 padding chars
		if inner < 1 {
			inner = 1
		}
		style = style.Width(inner)
	}

	box := style.Render(content)
	if title == "" {
		return box
	}

	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}

	titleText := " " + title + " "
	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(r.Theme.Label).
		Render(titleText)

	top := lines[0]
	topRunes := []rune(top)
	titleVisibleLen := lipgloss.Width(titleStyled)
	visibleWidth := lipgloss.Width(top)

	if titleVisibleLen+4 > visibleWidth {
		return box // too narrow — fall back to plain box
	}

	if len(topRunes) >= 2 {
		borderColorStyle := lipgloss.NewStyle().Foreground(r.Theme.Border)
		dashes := visibleWidth - 2 - titleVisibleLen - 1 // -1 for trailing corner
		if dashes < 1 {
			dashes = 1
		}
		newTop := borderColorStyle.Render("╭─") + titleStyled +
			borderColorStyle.Render(strings.Repeat("─", dashes)+"╮")
		lines[0] = newTop
	}
	return strings.Join(lines, "\n")
}

// RenderTabs draws the tab strip.
func (r Renderer) RenderTabs(active int) string {
	active = active % len(TabNames)
	tabActive := lipgloss.NewStyle().
		Bold(true).
		Foreground(r.Theme.TitleFg).
		Background(r.Theme.Accent).
		Padding(0, 2)
	tabInactive := lipgloss.NewStyle().
		Foreground(r.Theme.Dim).
		Padding(0, 2)

	parts := make([]string, len(TabNames))
	for i, name := range TabNames {
		key := fmt.Sprintf("%d:%s", i+1, name)
		if i == active {
			parts[i] = tabActive.Render(key)
		} else {
			parts[i] = tabInactive.Render(key)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// RenderHeader renders the top title bar with hostname/uptime/theme/alerts.
func (r Renderer) RenderHeader(s metrics.Snapshot, themeName string, alerts []string) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(r.Theme.TitleFg).
		Background(r.Theme.TitleBg).
		Padding(0, 2).
		Render("XTUI")

	info := r.dimStyle().Render(fmt.Sprintf(
		"%s · %s · up %s · theme: %s",
		s.Hostname, s.Platform, FormatUptime(s.Uptime), themeName,
	))

	left := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", info)

	if len(alerts) == 0 {
		return left
	}

	alertStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(r.Theme.AlertFg).
		Background(r.Theme.AlertBg).
		Padding(0, 1).
		Blink(true)
	alertText := alertStyle.Render("! " + strings.Join(alerts, " · "))

	pad := r.Width - lipgloss.Width(left) - lipgloss.Width(alertText)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + alertText
}

// RenderFooter shows hotkey help.
func (r Renderer) RenderFooter(interval string, recording bool) string {
	keys := []string{
		"[1-6] tab", "[t] theme", "[g] gradient", "[s] snapshot",
		"[r] record", "[q] quit",
	}
	help := r.dimStyle().Italic(true).
		Render(fmt.Sprintf("refresh %s · %s", interval, strings.Join(keys, " · ")))
	if recording {
		rec := lipgloss.NewStyle().
			Foreground(r.Theme.AlertBg).
			Bold(true).
			Blink(true).
			Render("REC")
		help = rec + "  " + help
	}
	return help
}

// ---------- Tab content renderers ----------

// padToHeight adds blank lines so the rendered string has exactly `height` lines.
// If the content is already taller, it is returned unchanged (we don't truncate
// because doing so would slice into ANSI escape sequences and break colors).
func padToHeight(s string, height int) string {
	current := lipgloss.Height(s)
	if current >= height {
		return s
	}
	return s + strings.Repeat("\n", height-current)
}

// RenderOverview is the default dashboard.
//
// Layout is locked to fixed block heights AND equal column widths so the screen
// never jitters when per-tick content shrinks or grows (e.g. network interfaces
// appearing, sensors disappearing, or one block having shorter text than the
// other). Each summary helper produces a body shorter than its target height;
// padToHeight tops it up with blanks, and boxWithTitleWidth pins the outer
// width so columns line up perfectly.
func (r Renderer) RenderOverview(s metrics.Snapshot, h *Histories, gradient bool) string {
	const (
		topRowHeight    = 9 // CPU + Memory bodies
		bottomRowHeight = 9 // Network + Sensors bodies
		gap             = 2 // spaces between left and right columns
	)

	// Split the available width evenly between the two columns.
	colWidth := (r.Width - gap) / 2
	if colWidth < 30 {
		colWidth = 30 // sane minimum
	}

	cpuBody := padToHeight(r.cpuSummaryBody(s, h, gradient), topRowHeight)
	memBody := padToHeight(r.memSummaryBody(s, h, gradient), topRowHeight)
	netBody := padToHeight(r.netSummaryBody(s, h), bottomRowHeight)
	extBody := padToHeight(r.extrasSummaryBody(s), bottomRowHeight)

	cpuBlock := r.boxWithTitleWidth("CPU", cpuBody, colWidth)
	memBlock := r.boxWithTitleWidth("Memory", memBody, colWidth)
	netBlock := r.boxWithTitleWidth("Network", netBody, colWidth)
	extBlock := r.boxWithTitleWidth("Sensors", extBody, colWidth)

	spacer := strings.Repeat(" ", gap)
	top := lipgloss.JoinHorizontal(lipgloss.Top, cpuBlock, spacer, memBlock)
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, netBlock, spacer, extBlock)
	return top + "\n" + bottom
}

// --- bodies (no border) so RenderOverview can pad them to a fixed height ---

func (r Renderer) cpuSummaryBody(s metrics.Snapshot, h *Histories, gradient bool) string {
	var b strings.Builder
	bar := Bar(r.Theme, s.CPUTotal, r.BarWidth)
	if gradient {
		bar = GradientBar(r.Theme, s.CPUTotal, r.BarWidth)
	}
	b.WriteString(bar)
	b.WriteString("\n\n")
	b.WriteString(r.dimStyle().Render("history: "))
	b.WriteString(Sparkline(r.Theme, h.CPU.Values(), 100))
	b.WriteString("\n\n")
	b.WriteString(r.dimStyle().Render(fmt.Sprintf("cores: %d", s.NumCPU)))
	return b.String()
}

func (r Renderer) memSummaryBody(s metrics.Snapshot, h *Histories, gradient bool) string {
	var b strings.Builder

	memBar := Bar(r.Theme, s.MemPct, r.BarWidth)
	if gradient {
		memBar = GradientBar(r.Theme, s.MemPct, r.BarWidth)
	}
	b.WriteString(r.labelStyle().Render("RAM"))
	b.WriteString("\n")
	b.WriteString(memBar)
	b.WriteString("\n")
	b.WriteString(r.dimStyle().Render(fmt.Sprintf("%s / %s",
		HumanBytes(s.MemUsed), HumanBytes(s.MemTotal))))
	b.WriteString("\n\n")

	b.WriteString(r.labelStyle().Render("SWAP"))
	b.WriteString("\n")
	if s.SwapTotal > 0 {
		swapBar := Bar(r.Theme, s.SwapPct, r.BarWidth)
		if gradient {
			swapBar = GradientBar(r.Theme, s.SwapPct, r.BarWidth)
		}
		b.WriteString(swapBar)
		b.WriteString("\n")
		b.WriteString(r.dimStyle().Render(fmt.Sprintf("%s / %s",
			HumanBytes(s.SwapUsed), HumanBytes(s.SwapTotal))))
	} else {
		b.WriteString(r.dimStyle().Render("(not configured)"))
	}
	return b.String()
}

func (r Renderer) netSummaryBody(s metrics.Snapshot, h *Histories) string {
	var b strings.Builder
	// Always emit exactly 2 interface rows (real or placeholder) so the
	// vertical position of the rx/tx sparklines doesn't shift when the
	// number of active interfaces changes between ticks.
	const ifaceSlots = 2
	for i := 0; i < ifaceSlots; i++ {
		if i < len(s.Net) {
			n := s.Net[i]
			b.WriteString(r.labelStyle().Render(n.Iface))
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("  ↓ %s   ↑ %s\n",
				r.valueStyle().Render(HumanRate(n.RxRate)),
				r.valueStyle().Render(HumanRate(n.TxRate))))
		} else {
			b.WriteString(r.dimStyle().Render("—"))
			b.WriteString("\n")
			b.WriteString(r.dimStyle().Render("  ↓ —          ↑ —"))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(r.dimStyle().Render("rx: "))
	b.WriteString(Sparkline(r.Theme, h.NetRx.Values(), 0))
	b.WriteString("\n")
	b.WriteString(r.dimStyle().Render("tx: "))
	b.WriteString(Sparkline(r.Theme, h.NetTx.Values(), 0))
	return b.String()
}

func (r Renderer) extrasSummaryBody(s metrics.Snapshot) string {
	var b strings.Builder

	// Battery section — always rendered (uses placeholders if absent) so the
	// total height of the body is stable.
	b.WriteString(r.labelStyle().Render("Battery"))
	b.WriteString("\n")
	if s.Battery.Present {
		b.WriteString(Bar(r.Theme, s.Battery.Pct, r.BarWidth))
		b.WriteString("\n")
		b.WriteString(r.dimStyle().Render(strings.ToLower(s.Battery.State)))
	} else {
		b.WriteString(r.dimStyle().Render("(not present)"))
		b.WriteString("\n")
	}
	b.WriteString("\n\n")

	// Temperature section — show up to 3 sensors, padded with empty rows
	// so the block height is constant whether 0, 1, 2, or 3 are reported.
	const tempSlots = 3
	b.WriteString(r.labelStyle().Render("Temperature"))
	b.WriteString("\n")
	for i := 0; i < tempSlots; i++ {
		if i < len(s.Temps) {
			t := s.Temps[i]
			color := r.Theme.LevelLow
			switch {
			case t.Temp > 80:
				color = r.Theme.LevelHigh
			case t.Temp > 60:
				color = r.Theme.LevelMid
			}
			val := lipgloss.NewStyle().Foreground(color).
				Render(fmt.Sprintf("%5.1f°C", t.Temp))
			b.WriteString(fmt.Sprintf("%s %s\n",
				r.dimStyle().Render(PadRight(t.Name, 18)),
				val))
		} else if i == 0 {
			b.WriteString(r.dimStyle().Render("(no sensors)"))
			b.WriteString("\n")
		} else {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// Backwards-compatible callers (RenderMemoryTab still uses memSummary).
func (r Renderer) memSummary(s metrics.Snapshot, h *Histories, gradient bool) string {
	return r.boxWithTitle("Memory", r.memSummaryBody(s, h, gradient))
}

// RenderCPUTab — detailed per-core view with sparklines.
func (r Renderer) RenderCPUTab(s metrics.Snapshot, h *Histories, gradient bool) string {
	var b strings.Builder
	b.WriteString(r.labelStyle().Render("Total"))
	b.WriteString("\n")
	bar := Bar(r.Theme, s.CPUTotal, r.BarWidth)
	if gradient {
		bar = GradientBar(r.Theme, s.CPUTotal, r.BarWidth)
	}
	b.WriteString(bar)
	b.WriteString("\n")
	b.WriteString(Sparkline(r.Theme, h.CPU.Values(), 100))
	b.WriteString("\n\n")

	b.WriteString(r.labelStyle().Render(fmt.Sprintf("Per core (%d)", len(s.CPUPerCPU))))
	b.WriteString("\n")
	for i, p := range s.CPUPerCPU {
		coreBar := Bar(r.Theme, p, r.BarWidth-8)
		if gradient {
			coreBar = GradientBar(r.Theme, p, r.BarWidth-8)
		}
		spark := ""
		if i < len(h.PerCore) && h.PerCore[i] != nil {
			spark = "  " + Sparkline(r.Theme, h.PerCore[i].Values(), 100)
		}
		b.WriteString(fmt.Sprintf("%s %s%s\n",
			r.dimStyle().Render(fmt.Sprintf("c%-2d", i)),
			coreBar, spark))
	}
	return r.boxWithTitle("CPU Detail", b.String())
}

func (r Renderer) RenderMemoryTab(s metrics.Snapshot, h *Histories, gradient bool) string {
	return r.memSummary(s, h, gradient) + "\n" +
		r.boxWithTitle("Top processes by memory", r.processTable(s.TopByMem, true))
}

func (r Renderer) RenderNetworkTab(s metrics.Snapshot, h *Histories) string {
	var b strings.Builder
	if len(s.Net) == 0 {
		b.WriteString(r.dimStyle().Render("no active interfaces yet"))
	} else {
		header := lipgloss.NewStyle().Bold(true).Foreground(r.Theme.Label).Render(
			fmt.Sprintf("%-12s %14s %14s %14s %14s",
				"iface", "rx/s", "tx/s", "rx total", "tx total"))
		b.WriteString(header)
		b.WriteString("\n")
		for _, n := range s.Net {
			b.WriteString(fmt.Sprintf("%-12s %14s %14s %14s %14s\n",
				PadRight(n.Iface, 12),
				HumanRate(n.RxRate),
				HumanRate(n.TxRate),
				HumanBytes(n.RxTotal),
				HumanBytes(n.TxTotal)))
		}
		b.WriteString("\n")
		b.WriteString(r.dimStyle().Render("rx history: "))
		b.WriteString(Sparkline(r.Theme, h.NetRx.Values(), 0))
		b.WriteString("\n")
		b.WriteString(r.dimStyle().Render("tx history: "))
		b.WriteString(Sparkline(r.Theme, h.NetTx.Values(), 0))
	}
	return r.boxWithTitle("Network", b.String())
}

func (r Renderer) RenderDiskTab(s metrics.Snapshot) string {
	var b strings.Builder
	if len(s.Disks) == 0 {
		b.WriteString(r.dimStyle().Render("no disks detected"))
	} else {
		for _, d := range s.Disks {
			b.WriteString(r.labelStyle().Render(d.Mount))
			b.WriteString("\n")
			b.WriteString(Bar(r.Theme, d.Pct, r.BarWidth))
			b.WriteString("\n")
			b.WriteString(r.dimStyle().Render(fmt.Sprintf("%s / %s",
				HumanBytes(d.Used), HumanBytes(d.Total))))
			b.WriteString("\n\n")
		}
	}
	return r.boxWithTitle("Disks", b.String())
}

func (r Renderer) RenderProcessesTab(s metrics.Snapshot) string {
	left := r.boxWithTitle("Top by CPU", r.processTable(s.TopByCPU, false))
	right := r.boxWithTitle("Top by Memory", r.processTable(s.TopByMem, true))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func (r Renderer) processTable(procs []metrics.ProcInfo, byMem bool) string {
	var b strings.Builder
	header := lipgloss.NewStyle().Bold(true).Foreground(r.Theme.Label).Render(
		fmt.Sprintf("%6s %-22s %8s %10s", "PID", "NAME", "CPU%", "MEM"))
	b.WriteString(header)
	b.WriteString("\n")
	if len(procs) == 0 {
		b.WriteString(r.dimStyle().Render("no data"))
		return b.String()
	}
	for _, p := range procs {
		var memCol string
		if byMem {
			memCol = HumanBytes(p.MemRSS)
		} else {
			memCol = fmt.Sprintf("%5.1f%%", p.MemPct)
		}
		b.WriteString(fmt.Sprintf("%6d %s %8.1f %10s\n",
			p.PID, PadRight(p.Name, 22), p.CPUPct, memCol))
	}
	return b.String()
}
