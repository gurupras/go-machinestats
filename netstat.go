package machinestats

import "github.com/cakturk/go-netstat/netstat"

// NetStat measures the network statistics
type NetStat struct {
}

func measureTCP() (int, error) {
	tabs, err := netstat.TCPSocks(netstat.NoopFilter)
	if err != nil {
		return -1, err
	}
	return len(tabs), nil
}

func measureUDP() (int, error) {
	tabs, err := netstat.UDPSocks(netstat.NoopFilter)
	if err != nil {
		return -1, err
	}
	return len(tabs), nil
}

// MeasureConnections returns the number of open sockets
func MeasureConnections() (int64, error) {
	count := int64(0)
	tcpCount, err := measureTCP()
	if err != nil {
		return -1, nil
	}
	count += int64(tcpCount)

	udpCount, err := measureUDP()
	if err != nil {
		return -1, nil
	}
	count += int64(udpCount)
	return count, nil
}

// Name of the stat
func (n *NetStat) Name() string {
	return "connections"
}

// Type of stat
func (n *NetStat) Type() StatType {
	return Gauge
}

// Measure returns the number of open sockets
func (n *NetStat) Measure(input interface{}) (float64, error) {
	count, err := MeasureConnections()
	if err != nil {
		return -1, err
	}
	return float64(count), nil
}
