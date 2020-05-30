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
	"github.com/gurupras/statsd"
	log "github.com/sirupsen/logrus"
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

func initDefaultAllCPUs(ncpu int) string {
	if ncpu > 1 {
		return "true"
	}
	return "false"
}

var (
	numCPUs         = runtime.NumCPU()
	defaultAllCpus  = initDefaultAllCPUs(numCPUs)
	defaultAddress  = getEnv("STATSD_ADDRESS", ":8125")
	defaultInterval = getEnv("STATSD_INTERVAL", "3000")
	defaultPrefix   = getEnv("STATSD_PREFIX", "")

	debug    = kingpin.Flag("debug", "Debug mode. Don't sent stats to backend").Short('D').Default("false").Bool()
	verbose  = kingpin.Flag("verbose", "Verbose logs").Short('v').Bool()
	allCPUs  = kingpin.Flag("all-cpus", "Log each individual CPU").Short('C').Default(defaultAllCpus).Bool()
	address  = kingpin.Flag("statsd-address", "Statsd server address").Short('a').Default(defaultAddress).String()
	interval = kingpin.Flag("statsd-interval", "Interval at which stats are collected periodically. In milliseconds").Short('d').Default(defaultInterval).Int()
	prefix   = kingpin.Flag("statsd-prefix", "Prefix with which all metrics are sent").Short('p').Default(defaultPrefix).String()
	prefixIP = kingpin.Flag("prefix-ip", "Add IP address as part of prefix").Default("false").Bool()
)

func main() {
	kingpin.Parse()
	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	addr := statsd.Address(*address)

	ip := GetOutboundIP().String()
	ipPrefix := strings.ReplaceAll(ip, ".", "-")

	var conn *statsd.Client
	var err error
	if !*debug {
		for {
			conn, err = statsd.New(
				addr,
				// Uncomment these once you figure out how to get Grafana to work with tags
				// statsd.TagsFormat(statsd.Datadog),
				// statsd.Tags("ip", ip, "alias", *prefix),
			)
			if err != nil {
				log.Errorf("Failed to set up connection: %v\n", err)
			} else {
				break
			}
			time.Sleep(1000 * time.Millisecond)
		}
		defer conn.Close()
	}

	stats := []machinestats.Stat{
		&machinestats.NetStat{},
		// The overall CPU usage line in /proc/stat is in line #0. Subsequent lines represent CPUs 0, 1, 2, ...
		// Thus, we use cpu-load.-1 to depict the overall CPU utilization
		machinestats.NewCPULoadStat(-1),
	}

	if *allCPUs {
		for idx := 0; idx < runtime.NumCPU(); idx++ {
			cpuLoadStat := machinestats.NewCPULoadStat(idx)
			stats = append(stats, cpuLoadStat)
		}
	}

	prefixArr := make([]string, 0)
	if strings.Compare(*prefix, "") != 0 {
		prefixArr = append(prefixArr, *prefix)
	}
	if *prefixIP {
		prefixArr = append(prefixArr, ipPrefix)
	}
	finalPrefix := strings.Join(prefixArr, ".")

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
				var stat *statsd.Client
				if !*debug {
					stat = conn.Clone(
						statsd.Prefix(finalPrefix),
					)
				}
				name := statInterface.Name()
				statType := statInterface.Type()
				// FIXME: We're feeding in proc/stat to all stat objects
				value, err := statInterface.Measure(procStatLines)
				if err != nil {
					log.Errorf("Failed to parse stat '%v': %v\n", name, err)
					return
				}
				if *debug {
					log.Debugf("Logged stat '%v' (%0.2f)\n", name, value)
				} else {
					switch statType {
					case machinestats.Gauge:
						stat.Gauge(name, value)
						log.Debugf("Logged gauge '%v'\n", name)
						break
					case machinestats.Counter:
						stat.Count(name, value)
						log.Debugf("Logged counter '%v'\n", name)
						break
					}
				}
			}(statInterface)
		}
		wg.Wait()
	}

	for {
		publishStats()
		time.Sleep(time.Duration(*interval) * time.Millisecond)
	}
}
