package metrics

import (
	"sort"
	"time"

	"github.com/distatus/battery"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type ProcInfo struct {
	PID    int32
	Name   string
	CPUPct float64
	MemPct float32
	MemRSS uint64
}

type DiskUsage struct {
	Mount string
	Used  uint64
	Total uint64
	Pct   float64
}

type NetSpeed struct {
	Iface   string
	RxRate  uint64 // bytes/sec
	TxRate  uint64 // bytes/sec
	RxTotal uint64
	TxTotal uint64
}

type Battery struct {
	Present bool
	Pct     float64
	State   string // charging, discharging, full, unknown
}

type TempSensor struct {
	Name string
	Temp float64 // celsius
}

type Snapshot struct {
	Time      time.Time
	CPUTotal  float64
	CPUPerCPU []float64
	MemUsed   uint64
	MemTotal  uint64
	MemPct    float64
	SwapUsed  uint64
	SwapTotal uint64
	SwapPct   float64
	Hostname  string
	Platform  string
	Uptime    uint64
	NumCPU    int

	Disks    []DiskUsage
	Net      []NetSpeed
	Battery  Battery
	Temps    []TempSensor
	TopByCPU []ProcInfo
	TopByMem []ProcInfo
}

// Collector holds state needed for delta calculations between samples.
type Collector struct {
	prevNet     map[string]net.IOCountersStat
	prevNetTime time.Time
}

func NewCollector() *Collector {
	return &Collector{}
}

// Collect gathers a full snapshot. The interval is the CPU sampling window.
func (c *Collector) Collect(interval time.Duration) Snapshot {
	var s Snapshot
	s.Time = time.Now()

	// CPU total — blocks for interval
	if total, err := cpu.Percent(interval, false); err == nil && len(total) > 0 {
		s.CPUTotal = total[0]
	}
	if per, err := cpu.Percent(0, true); err == nil {
		s.CPUPerCPU = per
	}
	if n, err := cpu.Counts(true); err == nil {
		s.NumCPU = n
	}

	// Memory
	if vm, err := mem.VirtualMemory(); err == nil {
		s.MemUsed = vm.Used
		s.MemTotal = vm.Total
		s.MemPct = vm.UsedPercent
	}
	if sw, err := mem.SwapMemory(); err == nil {
		s.SwapUsed = sw.Used
		s.SwapTotal = sw.Total
		s.SwapPct = sw.UsedPercent
	}

	// Host
	if h, err := host.Info(); err == nil {
		s.Hostname = h.Hostname
		s.Platform = h.Platform + " " + h.PlatformVersion
		s.Uptime = h.Uptime
	}

	// Disks (only physical-ish mounts)
	if parts, err := disk.Partitions(false); err == nil {
		for _, p := range parts {
			u, err := disk.Usage(p.Mountpoint)
			if err != nil || u.Total == 0 {
				continue
			}
			s.Disks = append(s.Disks, DiskUsage{
				Mount: p.Mountpoint,
				Used:  u.Used,
				Total: u.Total,
				Pct:   u.UsedPercent,
			})
		}
		// Keep at most 4, sorted by total descending
		sort.Slice(s.Disks, func(i, j int) bool {
			return s.Disks[i].Total > s.Disks[j].Total
		})
		if len(s.Disks) > 4 {
			s.Disks = s.Disks[:4]
		}
	}

	// Network — compute rate from previous sample
	if counters, err := net.IOCounters(true); err == nil {
		now := time.Now()
		curr := make(map[string]net.IOCountersStat, len(counters))
		for _, c2 := range counters {
			curr[c2.Name] = c2
		}
		if c.prevNet != nil {
			elapsed := now.Sub(c.prevNetTime).Seconds()
			if elapsed > 0 {
				for name, currStat := range curr {
					prev, ok := c.prevNet[name]
					if !ok {
						continue
					}
					// Skip loopback / down interfaces
					if name == "lo" || name == "lo0" {
						continue
					}
					rxDelta := safeDelta(currStat.BytesRecv, prev.BytesRecv)
					txDelta := safeDelta(currStat.BytesSent, prev.BytesSent)
					if rxDelta == 0 && txDelta == 0 && currStat.BytesRecv == 0 {
						continue
					}
					s.Net = append(s.Net, NetSpeed{
						Iface:   name,
						RxRate:  uint64(float64(rxDelta) / elapsed),
						TxRate:  uint64(float64(txDelta) / elapsed),
						RxTotal: currStat.BytesRecv,
						TxTotal: currStat.BytesSent,
					})
				}
				// Sort by current activity, top 3
				sort.Slice(s.Net, func(i, j int) bool {
					return s.Net[i].RxRate+s.Net[i].TxRate > s.Net[j].RxRate+s.Net[j].TxRate
				})
				if len(s.Net) > 3 {
					s.Net = s.Net[:3]
				}
			}
		}
		c.prevNet = curr
		c.prevNetTime = now
	}

	// Battery
	if bats, err := battery.GetAll(); err == nil && len(bats) > 0 {
		b := bats[0]
		s.Battery.Present = true
		if b.Full > 0 {
			s.Battery.Pct = (b.Current / b.Full) * 100
		}
		s.Battery.State = b.State.String()
	}

	// Temperatures
	if temps, err := host.SensorsTemperatures(); err == nil {
		for _, t := range temps {
			if t.Temperature == 0 {
				continue
			}
			s.Temps = append(s.Temps, TempSensor{
				Name: t.SensorKey,
				Temp: t.Temperature,
			})
		}
		// Limit and sort by temp descending
		sort.Slice(s.Temps, func(i, j int) bool {
			return s.Temps[i].Temp > s.Temps[j].Temp
		})
		if len(s.Temps) > 5 {
			s.Temps = s.Temps[:5]
		}
	}

	// Top processes — gathering CPU% requires a sample window from prior call
	s.TopByCPU, s.TopByMem = topProcesses()

	return s
}

func safeDelta(curr, prev uint64) uint64 {
	if curr < prev {
		return 0 // counter reset
	}
	return curr - prev
}

func topProcesses() (byCPU []ProcInfo, byMem []ProcInfo) {
	procs, err := process.Processes()
	if err != nil {
		return
	}
	all := make([]ProcInfo, 0, len(procs))
	for _, p := range procs {
		name, _ := p.Name()
		if name == "" {
			continue
		}
		cpuPct, _ := p.CPUPercent()
		memPct, _ := p.MemoryPercent()
		var rss uint64
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			rss = mi.RSS
		}
		all = append(all, ProcInfo{
			PID:    p.Pid,
			Name:   name,
			CPUPct: cpuPct,
			MemPct: memPct,
			MemRSS: rss,
		})
	}

	byCPU = make([]ProcInfo, len(all))
	copy(byCPU, all)
	sort.Slice(byCPU, func(i, j int) bool {
		return byCPU[i].CPUPct > byCPU[j].CPUPct
	})
	if len(byCPU) > 5 {
		byCPU = byCPU[:5]
	}

	byMem = make([]ProcInfo, len(all))
	copy(byMem, all)
	sort.Slice(byMem, func(i, j int) bool {
		return byMem[i].MemRSS > byMem[j].MemRSS
	})
	if len(byMem) > 5 {
		byMem = byMem[:5]
	}
	return
}
