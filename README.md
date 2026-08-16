# xtui

A beautiful terminal system monitor in Go. CPU, RAM, swap, network, disks, processes, sensors, battery — with sparkline history, themes, alerts, and CSV export.

## Features

- **Tabs**: Overview, CPU, Memory, Network, Disk, Processes
- **Sparkline history** for CPU, per-core, RAM, swap, network rx/tx
- **Top processes** by CPU and memory
- **Network speed** (rx/tx per interface, with auto-scaled sparkline)
- **Disk usage** for all mounted filesystems
- **Battery** percentage and charge state
- **Temperature sensors** (where the OS exposes them)
- **5 themes**: Catppuccin · Dracula · Nord · Gruvbox · Tokyo Night
- **Gradient progress bars** (toggleable)
- **Alerts** when CPU/RAM/temp cross thresholds (blinking header badge)
- **Export**: snapshot to JSON (`s` key), continuous recording to CSV (`r` key)
- **Splash screen** with ASCII logo
- **Adaptive layout** based on terminal width

  <img width="1097" height="603" alt="{2E81E16F-B931-4FF5-8798-4543EF839B12}" src="https://github.com/user-attachments/assets/4803e341-16d7-4f4e-8c4e-c835238d1799" />
  <img width="1043" height="693" alt="{6D3117F2-B11B-449F-B7F0-00EE1091D542}" src="https://github.com/user-attachments/assets/6a4b3869-0619-4a45-8fbf-e9167e0517d5" />
  <img width="854" height="286" alt="{14D5101D-CBEC-47DE-8060-AC10F75961E3}" src="https://github.com/user-attachments/assets/73befb47-9b84-415b-ae08-db528cf6a327" />
  <img width="964" height="432" alt="{93B2D719-C681-4E98-AB45-B40345A55878}" src="https://github.com/user-attachments/assets/5ce8bdcb-6261-4843-a85f-6f418262819f" />


## Build & run

```bash
cd xtui
go mod tidy
go run .
```

Build a binary:

```bash
go build -o xtui
./xtui
```

## Flags

```
--interval 1s            refresh rate (e.g. 500ms, 2s)
--theme catppuccin       catppuccin | dracula | nord | gruvbox | tokyonight
--gradient               gradient progress bars (default true)
--no-splash              skip splash screen
--history 60             sparkline length (samples)
--cpu-alert 90           CPU% alert threshold
--mem-alert 85           memory% alert threshold
--compact                compact mode (reserved for future use)
```

## Hotkeys

| key             | action                                        |
|-----------------|-----------------------------------------------|
| `1`–`6`         | switch tab directly                           |
| `Tab` / `→` / `l` | next tab                                    |
| `Shift+Tab` / `←` / `h` | previous tab                          |
| `t`             | cycle theme                                   |
| `g`             | toggle gradient bars                          |
| `s`             | save current snapshot to `xtui-snapshot-*.json` |
| `r`             | start/stop CSV recording (`xtui-record-*.csv`)  |
| `q` / `Esc` / `Ctrl+C` | quit                                   |

## Project layout

```
xtui/
├── main.go                 # Bubble Tea entry, key handling, export
├── internal/
│   ├── metrics/            # gopsutil-based snapshot collection
│   ├── theme/              # color palettes
│   └── ui/                 # rendering: bars, sparklines, tabs, splash
└── go.mod
```

## Dependencies

- [bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [lipgloss](https://github.com/charmbracelet/lipgloss) — styling
- [gopsutil](https://github.com/shirou/gopsutil) — cross-platform system metrics
- [distatus/battery](https://github.com/distatus/battery) — battery info

## Notes

- **Process CPU%** reads zero on the very first tick — gopsutil needs two samples per process to compute it. Stabilises after the second refresh.
- **Temperatures** depend on OS support: works well on Linux (`/sys/class/thermal`), partial on macOS, often unavailable on Windows without admin.
- **Network rates** are deltas between successive samples — first tick shows `0/s` until there's a previous sample to diff against.
