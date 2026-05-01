package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"xtui/internal/metrics"
	"xtui/internal/theme"
	"xtui/internal/ui"
)

// ============================================================================
// CLI flags
// ============================================================================

type config struct {
	interval    time.Duration
	themeName   string
	compact     bool
	gradient    bool
	noSplash    bool
	historyLen  int
	cpuAlertPct float64
	memAlertPct float64
}

func parseFlags() config {
	c := config{}
	flag.DurationVar(&c.interval, "interval", time.Second, "refresh interval (e.g. 500ms, 2s)")
	flag.StringVar(&c.themeName, "theme", "catppuccin",
		"color theme: catppuccin | dracula | nord | gruvbox | tokyonight")
	flag.BoolVar(&c.compact, "compact", false, "compact mode (no borders/padding)")
	flag.BoolVar(&c.gradient, "gradient", true, "gradient progress bars")
	flag.BoolVar(&c.noSplash, "no-splash", false, "skip splash screen")
	flag.IntVar(&c.historyLen, "history", 60, "sparkline history length (samples)")
	flag.Float64Var(&c.cpuAlertPct, "cpu-alert", 90, "CPU % alert threshold")
	flag.Float64Var(&c.memAlertPct, "mem-alert", 85, "memory % alert threshold")
	flag.Parse()
	return c
}

// ============================================================================
// Messages
// ============================================================================

type snapshotMsg metrics.Snapshot
type splashDoneMsg struct{}

// ============================================================================
// Model
// ============================================================================

type model struct {
	cfg       config
	theme     theme.Theme
	collector *metrics.Collector
	snap      metrics.Snapshot
	hist      *ui.Histories

	width, height int
	activeTab     int
	gradient      bool
	showSplash    bool

	// recording
	recording bool
	recFile   *os.File
	recCSV    *csv.Writer
	recMu     sync.Mutex

	// status flash (e.g. "saved snapshot.json")
	statusMsg   string
	statusUntil time.Time
}

func initialModel(cfg config) model {
	return model{
		cfg:        cfg,
		theme:      theme.ByName(cfg.themeName),
		collector:  metrics.NewCollector(),
		hist:       ui.NewHistories(cfg.historyLen),
		gradient:   cfg.gradient,
		showSplash: !cfg.noSplash,
		width:      100,
		height:     30,
	}
}

// ============================================================================
// Commands
// ============================================================================

func (m model) collectCmd() tea.Cmd {
	interval := m.cfg.interval
	collector := m.collector
	return func() tea.Msg {
		s := collector.Collect(interval)
		return snapshotMsg(s)
	}
}

func splashTimerCmd() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
		return splashDoneMsg{}
	})
}

// ============================================================================
// Bubble Tea
// ============================================================================

func (m model) Init() tea.Cmd {
	if m.showSplash {
		return tea.Batch(splashTimerCmd(), m.collectCmd())
	}
	return m.collectCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case splashDoneMsg:
		m.showSplash = false
		return m, nil

	case snapshotMsg:
		m.snap = metrics.Snapshot(msg)
		m.hist.Push(m.snap)
		if m.recording {
			m.recordSample(m.snap)
		}
		return m, m.collectCmd()
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.stopRecording()
		return m, tea.Quit

	case "1", "2", "3", "4", "5", "6":
		idx := int(msg.String()[0] - '1')
		if idx < len(ui.TabNames) {
			m.activeTab = idx
		}
	case "tab", "right", "l":
		m.activeTab = (m.activeTab + 1) % len(ui.TabNames)
	case "shift+tab", "left", "h":
		m.activeTab = (m.activeTab - 1 + len(ui.TabNames)) % len(ui.TabNames)

	case "t":
		// cycle theme
		idx := theme.Index(m.theme.Name)
		idx = (idx + 1) % len(theme.All)
		m.theme = theme.All[idx]
		m.flash("theme: " + m.theme.Name)

	case "g":
		m.gradient = !m.gradient
		if m.gradient {
			m.flash("gradient bars: on")
		} else {
			m.flash("gradient bars: off")
		}

	case "s":
		if path, err := m.saveSnapshot(); err != nil {
			m.flash("snapshot error: " + err.Error())
		} else {
			m.flash("saved: " + path)
		}

	case "r":
		if m.recording {
			m.stopRecording()
			m.flash("recording stopped")
		} else {
			if path, err := m.startRecording(); err != nil {
				m.flash("record error: " + err.Error())
			} else {
				m.flash("recording: " + path)
			}
		}
	}
	return m, nil
}

func (m *model) flash(msg string) {
	m.statusMsg = msg
	m.statusUntil = time.Now().Add(2 * time.Second)
}

// ============================================================================
// Snapshot / Recording (export feature)
// ============================================================================

func (m *model) saveSnapshot() (string, error) {
	path := fmt.Sprintf("xtui-snapshot-%s.json", time.Now().Format("20060102-150405"))
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m.snap); err != nil {
		return "", err
	}
	return path, nil
}

func (m *model) startRecording() (string, error) {
	path := fmt.Sprintf("xtui-record-%s.csv", time.Now().Format("20060102-150405"))
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	w := csv.NewWriter(f)
	w.Write([]string{
		"timestamp", "cpu_pct", "mem_pct", "mem_used_bytes", "swap_pct",
		"net_rx_bps", "net_tx_bps", "battery_pct",
	})
	w.Flush()
	m.recFile = f
	m.recCSV = w
	m.recording = true
	return path, nil
}

func (m *model) recordSample(s metrics.Snapshot) {
	m.recMu.Lock()
	defer m.recMu.Unlock()
	if m.recCSV == nil {
		return
	}
	var rx, tx uint64
	if len(s.Net) > 0 {
		rx, tx = s.Net[0].RxRate, s.Net[0].TxRate
	}
	m.recCSV.Write([]string{
		s.Time.Format(time.RFC3339),
		fmt.Sprintf("%.2f", s.CPUTotal),
		fmt.Sprintf("%.2f", s.MemPct),
		fmt.Sprintf("%d", s.MemUsed),
		fmt.Sprintf("%.2f", s.SwapPct),
		fmt.Sprintf("%d", rx),
		fmt.Sprintf("%d", tx),
		fmt.Sprintf("%.2f", s.Battery.Pct),
	})
	m.recCSV.Flush()
}

func (m *model) stopRecording() {
	m.recMu.Lock()
	defer m.recMu.Unlock()
	if m.recCSV != nil {
		m.recCSV.Flush()
	}
	if m.recFile != nil {
		m.recFile.Close()
	}
	m.recCSV = nil
	m.recFile = nil
	m.recording = false
}

// ============================================================================
// View
// ============================================================================

func (m model) View() string {
	if m.showSplash {
		return ui.RenderSplash(m.theme, m.width, m.height)
	}
	if m.snap.MemTotal == 0 {
		return "Collecting initial data..."
	}

	r := ui.Renderer{
		Theme:    m.theme,
		Width:    m.width,
		Height:   m.height,
		BarWidth: barWidthFor(m.width),
	}

	alerts := m.activeAlerts()

	var b strings.Builder
	b.WriteString(r.RenderHeader(m.snap, m.theme.Name, alerts))
	b.WriteString("\n")
	b.WriteString(r.RenderTabs(m.activeTab))
	b.WriteString("\n\n")

	switch m.activeTab {
	case ui.TabOverview:
		b.WriteString(r.RenderOverview(m.snap, m.hist, m.gradient))
	case ui.TabCPU:
		b.WriteString(r.RenderCPUTab(m.snap, m.hist, m.gradient))
	case ui.TabMemory:
		b.WriteString(r.RenderMemoryTab(m.snap, m.hist, m.gradient))
	case ui.TabNetwork:
		b.WriteString(r.RenderNetworkTab(m.snap, m.hist))
	case ui.TabDisk:
		b.WriteString(r.RenderDiskTab(m.snap))
	case ui.TabProcesses:
		b.WriteString(r.RenderProcessesTab(m.snap))
	}

	b.WriteString("\n")
	b.WriteString(r.RenderFooter(m.cfg.interval.String(), m.recording))
	if m.statusMsg != "" && time.Now().Before(m.statusUntil) {
		b.WriteString("\n> " + m.statusMsg)
	}
	return b.String()
}

func (m model) activeAlerts() []string {
	var alerts []string
	if m.snap.CPUTotal > m.cfg.cpuAlertPct {
		alerts = append(alerts, fmt.Sprintf("CPU %.0f%%", m.snap.CPUTotal))
	}
	if m.snap.MemPct > m.cfg.memAlertPct {
		alerts = append(alerts, fmt.Sprintf("RAM %.0f%%", m.snap.MemPct))
	}
	for _, t := range m.snap.Temps {
		if t.Temp > 85 {
			alerts = append(alerts, fmt.Sprintf("%s %.0f°C", t.Name, t.Temp))
			break
		}
	}
	return alerts
}

func barWidthFor(termWidth int) int {
	switch {
	case termWidth > 140:
		return 50
	case termWidth > 100:
		return 40
	case termWidth > 80:
		return 30
	default:
		return 20
	}
}

// ============================================================================
// main
// ============================================================================

func main() {
	cfg := parseFlags()
	p := tea.NewProgram(initialModel(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
