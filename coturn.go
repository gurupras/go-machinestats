package machinestats

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var totalSessionsPattern = regexp.MustCompile(`Total sessions: (?P<numSessions>\d+)\s+`)

// CoturnStat measures coturn statistics
type CoturnStat struct {
	host     string
	port     int
	password string
	reader   *bufio.Reader
	writer   *bufio.Writer
}

func (c *CoturnStat) waitUntil(text string, pattern *regexp.Regexp, timeout time.Duration) (string, error) {
	satisfied := false
	var result string
	var err error
	var b byte
	var lineBytes []byte

	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		time.Sleep(timeout)
		mutex.Lock()
		defer mutex.Unlock()
		defer wg.Done()
		err = fmt.Errorf("Timed out")
		satisfied = true
	}()

	go func() {
		for {
			var line string
			b, err = c.reader.ReadByte()
			if err != nil {
				return
			}
			mutex.Lock()
			if satisfied {
				return
			}
			lineBytes = append(lineBytes, b)
			line = string(lineBytes)
			if b == '\n' {
				log.Debugf("coturn line: %v\n", line)
			}
			if pattern != nil {
				match := pattern.FindStringSubmatch(line)
				if len(match) > 0 {
					satisfied = true
					result = line
					break
				}
			} else if b == text[len(text)-1] && strings.Contains(line, text) {
				satisfied = true
				result = line
				break
			}
			mutex.Unlock()
		}
		wg.Done()
	}()
	wg.Wait()
	return result, err
}

func (c *CoturnStat) waitForPasswordPrompt() error {
	_, err := c.waitUntil("Enter password:", nil, time.Duration(1*time.Second))
	return err
}

func (c *CoturnStat) waitForCommandPrompt() error {
	_, err := c.waitUntil("> ", nil, time.Duration(1*time.Second))
	return err
}

func (c *CoturnStat) login() error {
	c.waitForPasswordPrompt()
	log.Debugf("Received password prompt")
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.waitForCommandPrompt()
	}()
	n, err := c.writer.WriteString(fmt.Sprintf("%v\r\n", c.password))
	if err != nil {
		log.Errorf("Failed to write password: %v\n", err)
		return err
	}
	err = c.writer.Flush()
	if err != nil {
		return err
	}
	log.Debugf("Wrote password: (%v bytes)", n)
	wg.Wait()
	return nil
}

func (c *CoturnStat) get() (uint64, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%v:%v", c.host, c.port))
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	c.reader = reader
	c.writer = writer

	if err = c.login(); err != nil {
		return 0, err
	}

	wg := sync.WaitGroup{}
	var result string
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err = c.waitUntil("", totalSessionsPattern, time.Duration(1*time.Second))
	}()
	n, err := writer.WriteString("pu\r\n")
	if err != nil {
		log.Errorf("Failed to 'pu' command: %v\n", err)
	}
	if err := writer.Flush(); err != nil {
		return 0, err
	}
	log.Debugf("Wrote 'pu' command (%v bytes)", n)
	wg.Wait()
	if err != nil {
		return 0, err
	}
	match := totalSessionsPattern.FindStringSubmatch(result)
	if (len(match)) < 1 {
		return 0, fmt.Errorf(fmt.Sprintf("Pattern did not match: '%v'", result))
	}
	numStr := match[1]
	numSessions, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return numSessions, nil
}

type coturnStatMeasurement struct {
	numSessions uint64
}

// NewCoturnStat returns a coturn statistics measurer
func NewCoturnStat(host string, port int, password string) (*CoturnStat, error) {
	stats := &CoturnStat{
		host,
		port,
		password,
		nil,
		nil,
	}
	return stats, nil
}

// Name of this stat
func (c *CoturnStat) Name() string {
	return "coturn-stats"
}

// Name of the stat
func (c *coturnStatMeasurement) Name() string {
	return "coturn.numSessions"
}

// Type of stat
func (c *coturnStatMeasurement) Type() StatType {
	return Gauge
}

func (c *coturnStatMeasurement) Value() interface{} {
	return c.numSessions
}

// Measure returns the number of open sockets
func (c *CoturnStat) Measure(channel chan<- Measurement) error {
	numSessions, err := c.get()
	if err != nil {
		log.Errorf("Failed to get coturn stat: %v\n", err)
		return err
	}
	channel <- &coturnStatMeasurement{
		numSessions,
	}
	return nil
}
