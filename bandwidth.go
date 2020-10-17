package machinestats

import (
	"fmt"
	"time"

	"github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
)

const megaByte = float64(1024 * 1024)

// BandwidthStat represents a /proc/net/dev entry
type BandwidthStat struct {
	fs *procfs.FS
	procfs.NetDev
	lastMeasurementTime int64
}

// Name of BandwidthStat
func (b *BandwidthStat) Name() string {
	return "bandwidth-stat"
}

// NewBandwidthStat creates a BandwidthStat for monitoring bandwidth
func NewBandwidthStat(fs *procfs.FS) (*BandwidthStat, error) {
	if err := setupProcFS(); err != nil {
		return nil, err
	}
	if fs == nil {
		fs = procFS
	}
	return &BandwidthStat{
		fs,
		nil,
		0,
	}, nil
}

type bandwidthMeasurement struct {
	value float64
	name  string
}

func (b *bandwidthMeasurement) Type() StatType {
	return Gauge
}

func (b *bandwidthMeasurement) Value() interface{} {
	return b.value
}

func (b *bandwidthMeasurement) Name() string {
	return b.name
}

func sendBandwidthDiffs(channel chan<- Measurement, iface string, timeDelta time.Duration, newData, oldData procfs.NetDevLine) {
	downloaded := newData.RxBytes - oldData.RxBytes
	uploaded := newData.TxBytes - oldData.TxBytes

	log.Debugf("%v - downloaded (%v)", iface, downloaded)
	log.Debugf("%v - uploaded   (%v)", iface, uploaded)

	elapsedTimeInSeconds := float64(timeDelta / (time.Second))
	downloadSpeed := ((float64(downloaded) / elapsedTimeInSeconds) / megaByte) * 8
	uploadSpeed := ((float64(uploaded) / elapsedTimeInSeconds) / megaByte) * 8

	bm := &bandwidthMeasurement{
		downloadSpeed,
		fmt.Sprintf("%v.download.mbps", iface),
	}
	um := &bandwidthMeasurement{
		uploadSpeed,
		fmt.Sprintf("%v.upload.mbps", iface),
	}
	channel <- bm
	channel <- um
}

// Measure bandwidth
func (b *BandwidthStat) Measure(channel chan<- Measurement) error {
	oldData := b.NetDev
	now := nowFn()
	newData, err := b.fs.NetDev()
	if err != nil {
		return err
	}
	defer func() {
		b.NetDev = newData
		b.lastMeasurementTime = now
	}()

	if b.NetDev == nil {
		log.Debug("Returning nil due to no measurements")
		return nil
	}
	timeDelta := time.Duration(now-b.lastMeasurementTime) * time.Nanosecond
	log.Debugf("Elapsed time: %vs", int64(time.Duration(timeDelta)/time.Second))
	oldTotal := oldData.Total()
	newTotal := newData.Total()

	for iface := range newData {
		oldIfaceData := oldData[iface]
		newIfaceData := newData[iface]

		sendBandwidthDiffs(channel, iface, timeDelta, newIfaceData, oldIfaceData)
	}
	sendBandwidthDiffs(channel, "total", timeDelta, newTotal, oldTotal)
	return nil
}
