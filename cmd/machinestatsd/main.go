package main

import (
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	machinestats "github.com/gurupras/go-machinestats"
	log "github.com/sirupsen/logrus"
	statsd "gopkg.in/alexcesaro/statsd.v2"
)

// GetOutboundIP retrieves the preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if strings.Compare(val, "") == 0 {
		val = defaultValue
	}
	return val
}

var (
	defaultAddress  = getEnv("STATSD_ADDRESS", ":8125")
	defaultInterval = getEnv("STATSD_INTERVAL", "3000")
	defaultPrefix   = getEnv("STATSD_PREFIX", GetOutboundIP().String())

	verbose  = kingpin.Flag("verbose", "Verbose logs").Short('v').Bool()
	address  = kingpin.Flag("statsd-address", "Statsd server address").Short('a').Default(defaultAddress).String()
	interval = kingpin.Flag("statsd-interval", "Interval at which stats are collected periodically. In milliseconds").Short('d').Default(defaultInterval).Int()
	prefix   = kingpin.Flag("statsd-prefix", "Prefix with which all metrics are sent").Short('p').Default(defaultPrefix).String()
)

func main() {
	kingpin.Parse()
	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	addr := statsd.Address(*address)

	var conn *statsd.Client
	var err error
	for {
		conn, err = statsd.New(addr)
		if err != nil {
			log.Errorf("Failed to set up connection: %v\n", err)
		} else {
			break
		}
		time.Sleep(1000 * time.Millisecond)
	}
	defer conn.Close()

	stats := []machinestats.Stat{
		&machinestats.NetStat{},
	}

	for idx := -1; idx < runtime.NumCPU(); idx++ {
		cpuLoadStat := machinestats.NewCPULoadStat(idx)
		stats = append(stats, cpuLoadStat)
	}

	publishStats := func() {
		procStatLines, err := machinestats.ReadProcStat()
		if err != nil {
			log.Errorf("Failed to read /proc/stat: %v\n", err)
			return
		}
		wg := sync.WaitGroup{}
		for _, statInterface := range stats {
			wg.Add(1)
			go func(statInterface machinestats.Stat) {
				defer wg.Done()
				stat := conn.Clone(statsd.Prefix(*prefix))
				name := statInterface.Name()
				statType := statInterface.Type()
				// FIXME: We're feeding in proc/stat to all stat objects
				value, err := statInterface.Measure(procStatLines)
				if err != nil {
					log.Errorf("Failed to parse stat '%v': %v\n", name, err)
					return
				}
				switch statType {
				case machinestats.Gauge:
					stat.Gauge(name, value)
					break
				case machinestats.Counter:
					stat.Count(name, value)
				}
				log.Debugf("Logged stat '%v'\n", name)
			}(statInterface)
		}
		wg.Wait()
	}

	for {
		publishStats()
		time.Sleep(time.Duration(*interval) * time.Millisecond)
	}
}
