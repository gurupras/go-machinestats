package machinestats

import (
	"github.com/prometheus/procfs"
)

// NetStat measures the network statistics
type NetStat struct {
	fs *procfs.FS
}

type netStatMeasurement struct {
	protocol string
	value    int
}

// NewNetStat returns a network statistics measurer
func NewNetStat(fs *procfs.FS) (*NetStat, error) {
	if fs == nil {
		newFS, err := procfs.NewFS("/proc")
		if err != nil {
			return nil, err
		}
		fs = &newFS
	}
	return &NetStat{fs}, nil
}

// Name of this stat
func (n *NetStat) Name() string {
	return "net-stat"
}

// Name of the stat
func (n *netStatMeasurement) Name() string {
	return n.protocol
}

// Type of stat
func (n *netStatMeasurement) Type() StatType {
	return Gauge
}

func (n *netStatMeasurement) Value() interface{} {
	return n.value
}

// Measure returns the number of open sockets
func (n *NetStat) Measure(channel chan<- Measurement) error {
	sockstat, err := n.fs.NetSockstat()
	if err != nil {
		return err
	}
	// First, send total
	channel <- &netStatMeasurement{
		"connections",
		*sockstat.Used,
	}
	return nil
}
