package machinestats

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

// ProcStatFile represents the path to the stat file. This is overridden for testing purposes
var ProcStatFile = "/proc/stat"

type statLine struct {
	user          int64
	nice          int64
	system        int64
	idle          int64
	ioWait        int64
	irq           int64
	softIRQ       int64
	steal         int64
	guest         int64
	guestNice     int64
	userTime      int64
	niceTime      int64
	idleAllTime   int64
	systemAllTime int64
	virtAllTime   int64
	totalAllTime  int64
}

func (s *statLine) total() int64 {
	user := s.user - s.guest
	nice := s.nice - s.guestNice
	idle := s.idle + s.ioWait
	system := s.system + s.irq + s.softIRQ
	virt := s.guest + s.guestNice
	total := user + nice + system + idle + s.steal + virt
	return total
}

func calculateBusyness(now, old *statLine) float64 {
	deltaTotal := now.totalAllTime - old.totalAllTime
	deltaIdle := now.idleAllTime - old.idleAllTime
	return float64(deltaTotal-deltaIdle) / float64(deltaTotal)
}

// processCPUStatLine processes a stat line from /proc/stat corresponding to a CPU
func processCPUStatLine(str string) (*statLine, error) {
	// Replace all double-spaces with single space
	str = strings.ReplaceAll(str, "  ", " ")
	tokens := strings.Split(str, " ")
	user, err := strconv.ParseInt(tokens[1], 10, 64)
	if err != nil {
		return nil, err
	}
	nice, err := strconv.ParseInt(tokens[2], 10, 64)
	if err != nil {
		return nil, err
	}
	system, err := strconv.ParseInt(tokens[3], 10, 64)
	if err != nil {
		return nil, err
	}
	idle, err := strconv.ParseInt(tokens[4], 10, 64)
	if err != nil {
		return nil, err
	}
	ioWait, err := strconv.ParseInt(tokens[5], 10, 64)
	if err != nil {
		return nil, err
	}
	irq, err := strconv.ParseInt(tokens[6], 10, 64)
	if err != nil {
		return nil, err
	}
	softIRQ, err := strconv.ParseInt(tokens[7], 10, 64)
	if err != nil {
		return nil, err
	}
	steal, err := strconv.ParseInt(tokens[8], 10, 64)
	if err != nil {
		return nil, err
	}
	guest, err := strconv.ParseInt(tokens[9], 10, 64)
	if err != nil {
		return nil, err
	}
	guestNice, err := strconv.ParseInt(tokens[10], 10, 64)
	if err != nil {
		return nil, err
	}

	userTime := user - guest
	niceTime := nice - guestNice
	idleAllTime := idle + ioWait
	systemAllTime := system + irq + softIRQ
	virtAllTime := guest + guestNice
	total := userTime + niceTime + systemAllTime + idleAllTime + steal + virtAllTime

	return &statLine{
		user,
		nice,
		system,
		idle,
		ioWait,
		irq,
		softIRQ,
		steal,
		guest,
		guestNice,
		userTime,
		niceTime,
		idleAllTime,
		systemAllTime,
		virtAllTime,
		total,
	}, nil
}

// CPULoadStat measures and tracks the CPU load
type CPULoadStat struct {
	// CPU represented by this struct
	CPU  int
	prev *statLine
}

// NewCPULoadStat creates a CPULoadStat for the given CPU
func NewCPULoadStat(cpu int) *CPULoadStat {
	return &CPULoadStat{
		cpu,
		nil,
	}
}

// ReadProcStat reads /proc/stat and returns the contents as a string
func ReadProcStat() ([]string, error) {
	b, err := ioutil.ReadFile(ProcStatFile)
	if err != nil {
		return nil, err
	}
	str := string(b)
	return strings.Split(str, "\n"), nil
}

// Name of the stat
func (c *CPULoadStat) Name() string {
	return fmt.Sprintf("cpu-load.%02d", c.CPU)
}

// Type of stat
func (c *CPULoadStat) Type() StatType {
	return Gauge
}

// Measure the CPU load
func (c *CPULoadStat) Measure(input interface{}) (float64, error) {
	lines := input.([]string)
	cpuLine := lines[c.CPU+1] // Line-0 is aggregated stats
	statLine, err := processCPUStatLine(cpuLine)
	if err != nil {
		return 0, err
	}
	if c.prev == nil {
		c.prev = statLine
		return 0, nil
	}
	busyness := calculateBusyness(statLine, c.prev)
	return busyness, nil
}
