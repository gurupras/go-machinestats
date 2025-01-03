package machinestats

import "time"

var nowFn = func() int64 {
	return time.Now().UnixNano()
}

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
	Measure(chan<- Measurement) error
}

// Measurement represents a measurement
type Measurement interface {
	Name() string
	Type() StatType
	Value() interface{}
}

type BasicMeasurement struct {
	name            string
	measurementType StatType
	value           interface{}
}

func (bm *BasicMeasurement) Name() string {
	return bm.name
}
func (bm *BasicMeasurement) Type() StatType {
	return bm.measurementType
}
func (bm *BasicMeasurement) Value() interface{} {
	return bm.value
}
