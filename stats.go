package machinestats

// StatType represents the type of stat
type StatType int

const (
	// Gauge for statsd Gauge
	Gauge StatType = iota
	// Counter for statsd Counter
	Counter StatType = iota
)

// Stat is an abstract interface that is used to get some measurement
type Stat interface {
	Name() string
	Type() StatType
	Measure(input interface{}) (float64, error)
}
