package machinestats

import (
	"fmt"

	"github.com/prometheus/procfs"
)

// CPUStat represents a CPU's /proc/stat entry
type CPUStat struct {
	*procfs.CPUStat
	total float64
	idle  float64
}

func newCPUStat(s *procfs.CPUStat) *CPUStat {
	ret := &CPUStat{s, 0, 0}
	ret.total = ret.computeTotalCPUTime()
	ret.idle = ret.computeIdleTime()
	return ret
}

func (s *CPUStat) computeTotalCPUTime() float64 {
	user := s.User - s.Guest
	nice := s.Nice - s.GuestNice
	idle := s.Idle + s.Iowait
	system := s.System + s.IRQ + s.SoftIRQ
	virt := s.Guest + s.GuestNice
	total := user + nice + system + idle + s.Steal + virt
	return total
}

func (s *CPUStat) computeIdleTime() float64 {
	return s.Idle + s.Iowait
}

func calculateBusyness(now, old *CPUStat) float64 {
	idle := now.idle - old.idle
	total := now.total - old.total
	active := total - idle
	return active / total
}

// CPULoadStat measures and tracks the CPU load
type CPULoadStat struct {
	prevStat []*CPUStat
	fs       *procfs.FS
}

// NewCPULoadStat creates a CPULoadStat for the given CPU
func NewCPULoadStat(fs *procfs.FS) (*CPULoadStat, error) {
	if err := setupProcFS(); err != nil {
		return nil, err
	}
	if fs == nil {
		fs = procFS
	}
	return &CPULoadStat{
		nil,
		fs,
	}, nil
}

type cpuBusyMeasurement struct {
	cpu      int
	busyness float64
}

// Name of the measurement
func (c *cpuBusyMeasurement) Name() string {
	return fmt.Sprintf("cpu-load.%02d", c.cpu)
}

// Type of stat
func (c *cpuBusyMeasurement) Type() StatType {
	return Gauge
}

func (c *cpuBusyMeasurement) Value() interface{} {
	return c.busyness
}

// Name of this stat
func (c *CPULoadStat) Name() string {
	return "cpu-load-stat"
}

// Measure the CPU load
func (c *CPULoadStat) Measure(channel chan<- Measurement) error {
	stat, err := c.fs.Stat()
	if err != nil {
		return err
	}
	cpuStatArray := make([]*CPUStat, len(stat.CPU)+1) // + 1 for the total
	cpuStatArray[0] = newCPUStat(&stat.CPUTotal)
	for idx, entry := range stat.CPU {
		cpuStatArray[idx+1] = newCPUStat(&entry)
	}

	if c.prevStat == nil {
		c.prevStat = cpuStatArray
		return nil
	}

	for idx, current := range cpuStatArray {
		prev := c.prevStat[idx]
		busyness := calculateBusyness(current, prev)
		m := &cpuBusyMeasurement{
			idx - 1,
			busyness,
		}
		channel <- m
	}
	c.prevStat = cpuStatArray
	return nil
}
