package main

import (
	"net"
	"os"
	"runtime"
	"strings"
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
	numCPUs           = runtime.NumCPU()
	defaultDebugMode  = getEnv("MACHINESTATSD_DEBUG", "false")
	defaultAllCpus    = initDefaultAllCPUs(numCPUs)
	defaultAddress    = getEnv("STATSD_ADDRESS", ":8125")
	defaultInterval   = getEnv("STATSD_INTERVAL", "3000")
	defaultPrefix     = getEnv("STATSD_PREFIX", "")
	defaultPrefixIP   = getEnv("MACHINESTATSD_PREFIX_IP", "false")
	defaultCoturn     = getEnv("MACHINESTATSD_ENABLE_COTURN", "false")
	defaultCoturnHost = getEnv("MACHINESTATSD_COTURN_HOST", "127.0.0.1")
	defaultCoturnPort = getEnv("MACHINESTATSD_COTURN_PORT", "5558")
	defaultVerbose    = getEnv("MACHINESTATSD_VERBOSE", "false")

	debug    = kingpin.Flag("debug", "Debug mode. Don't sent stats to backend").Short('D').Default(defaultDebugMode).Bool()
	verbose  = kingpin.Flag("verbose", "Verbose logs").Short('v').Default(defaultVerbose).Bool()
	allCPUs  = kingpin.Flag("all-cpus", "Log each individual CPU").Short('C').Default(defaultAllCpus).Bool()
	address  = kingpin.Flag("statsd-address", "Statsd server address").Short('a').Default(defaultAddress).String()
	interval = kingpin.Flag("statsd-interval", "Interval at which stats are collected periodically. In milliseconds").Short('d').Default(defaultInterval).Int()
	prefix   = kingpin.Flag("statsd-prefix", "Prefix with which all metrics are sent").Short('p').Default(defaultPrefix).String()
	prefixIP = kingpin.Flag("prefix-ip", "Add IP address as part of prefix").Default(defaultPrefixIP).Bool()

	enableCoturn   = kingpin.Flag("enable-coturn", "Enable stat collection from Coturn instance").Default(defaultCoturn).Bool()
	coturnHost     = kingpin.Flag("coturn-host", "Coturn server host").Default(defaultCoturnHost).String()
	coturnPort     = kingpin.Flag("coturn-port", "Coturn server CLI port").Default(defaultCoturnPort).Int()
	coturnPassword = kingpin.Flag("coturn-password", "Coturn server CLI password").String()
)

func asFloat64(input interface{}) float64 {
	switch val := input.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case uint64:
		return float64(val)
	}
	return 0
}

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

	netstat, err := machinestats.NewNetStat(nil)
	if err != nil {
		log.Fatalf("Failed to create netstat: %v\n", err)
	}
	cpustat, err := machinestats.NewCPULoadStat(nil)
	if err != nil {
		log.Fatalf("Failed to create cpustat: %v\n", err)
	}
	memstat, err := machinestats.NewMemLoadStat(nil)
	if err != nil {
		log.Fatalf("Failed to create memStat: %v\n", err)
	}
	bwstat, err := machinestats.NewBandwidthStat(nil)
	if err != nil {
		log.Fatalf("Failed to create bandwidthStat: %v\n", err)
	}

	stats := []machinestats.Stat{
		netstat,
		cpustat,
		memstat,
		bwstat,
	}

	if *enableCoturn {
		coturnStat, err := machinestats.NewCoturnStat(*coturnHost, *coturnPort, *coturnPassword)
		if err != nil {
			log.Fatalf("Failed to create coturnStat: %v\n", err)
		}
		stats = append(stats, coturnStat)
	}

	prefixArr := make([]string, 0)
	if strings.Compare(*prefix, "") != 0 {
		prefixArr = append(prefixArr, *prefix)
	}
	if *prefixIP {
		prefixArr = append(prefixArr, ipPrefix)
	}
	finalPrefix := strings.Join(prefixArr, ".")

	channel := make(chan machinestats.Measurement, 0)
	go func() {
		var stat *statsd.Client
		for measurement := range channel {
			if !*debug {
				stat = conn.Clone(
					statsd.Prefix(finalPrefix),
				)
			}
			name := measurement.Name()
			statType := measurement.Type()
			value := measurement.Value()
			if *debug {
				log.Debugf("Logged stat '%v' (%0.2f)\n", name, asFloat64(value))
				continue
			}
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
	}()

	for {
		for _, stat := range stats {
			err := stat.Measure(channel)
			if err != nil {
				log.Errorf("Failed to parse stat '%v': %v\n", stat.Name(), err)
				continue
			}
		}
		time.Sleep(time.Duration(*interval) * time.Millisecond)
	}
}
