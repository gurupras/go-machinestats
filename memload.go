package machinestats

import "github.com/prometheus/procfs"

// MemLoadStat represents all the information obtained from one /proc/meminfo read
type MemLoadStat struct {
	fs    *procfs.FS
	value float64
}

// NewMemLoadStat creates a new instance of MemLoadStat
func NewMemLoadStat(fs *procfs.FS) (*MemLoadStat, error) {
	if err := setupProcFS(); err != nil {
		return nil, err
	}
	if fs == nil {
		fs = procFS
	}
	return &MemLoadStat{fs, 0}, nil
}

// Type of stat
func (m *MemLoadStat) Type() StatType {
	return Gauge
}

// Value of stat
func (m *MemLoadStat) Value() interface{} {
	return m.value
}

// Name of stat
func (m *MemLoadStat) Name() string {
	return "memory-load-stat"
}

// Measure the current memory usage
func (m *MemLoadStat) Measure(channel chan<- Measurement) error {
	meminfo, err := m.fs.Meminfo()
	if err != nil {
		return err
	}
	used := meminfo.MemTotal - meminfo.MemAvailable
	pct := (float64(used) / float64(meminfo.MemTotal)) * 100
	m.value = pct
	channel <- m
	return nil
}
