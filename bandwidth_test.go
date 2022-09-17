package machinestats

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	procfs "github.com/prometheus/procfs"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var netdevLine1 = strings.TrimSpace(`Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth1:    5494     111    0    0    0     0          0         0     2126      29    0    0    0     0       0          0
    lo: 16569742  243321    0    0    0     0          0         0 16569742  243321    0    0    0     0       0          0
  eth0: 12400545989 18047311    0    0    0     0          0         0 4675925594017 36850401    0    0    0     0       0          0`)

var netdevLine2 = strings.TrimSpace(`Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth1:    5564     112    0    0    0     0          0         0     2126      29    0    0    0     0       0          0
    lo: 17612915  260347    0    0    0     0          0         0 17612915  260347    0    0    0     0       0          0
  eth0: 14364103758 20338788    0    0    0     0          0         0 4675984158805 44888527    0    0    0     0       0          0`)

func TestBandwidthStat(t *testing.T) {
	// log.SetLevel(log.DebugLevel)
	assert := assert.New(t)
	require := require.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "test-dir")
	require.Nil(err)
	require.NotNil(dir)

	netDir := path.Join(dir, "net")
	_ = os.Mkdir(netDir, 0775)

	path := path.Join(dir, "net", "dev")

	err = ioutil.WriteFile(path, []byte(netdevLine1), 0644)
	require.Nil(err)

	fs, err := procfs.NewFS(dir)
	require.Nil(err)

	b, err := NewBandwidthStat(&fs)
	require.Nil(err)
	require.NotNil(b)

	channel := make(chan Measurement)

	numMeasurements := (3 + 1) * 2
	wg := sync.WaitGroup{}
	wg.Add(numMeasurements)

	expectedResults := map[string]float64{
		"eth1.upload.mbps":    0.000178019,
		"eth1.download.mbps":  0.000002782,
		"eth0.upload.mbps":    148.937957764,
		"eth0.download.mbps":  4993.585634867,
		"lo.upload.mbps":      2.652926127,
		"lo.download.mbps":    2.652926127,
		"total.upload.mbps":   151.590883891,
		"total.download.mbps": 4996.238739014,
	}
	go func() {
		for idx := 0; idx < numMeasurements; idx++ {
			data := <-channel
			// Strip initial network.interfaces. part
			name := strings.Replace(data.Name(), "network.interfaces.", "", 1)
			got := data.Value().(float64)
			log.Debugf("Got measurement: %v - %v\n", name, got)
			expected := expectedResults[name]
			diff := math.Abs(expected - got)
			assert.True(diff < 1e-3, fmt.Sprintf("[%v]: diff was: %v", name, diff))
			wg.Done()
		}
	}()
	now := nowFn()
	nowFn = func() int64 { return now }
	err = b.Measure(channel)
	require.Nil(err)

	// Update time
	nowFn = func() int64 { return now + int64(3*time.Second) }

	err = ioutil.WriteFile(path, []byte(netdevLine2), 0644)
	require.Nil(err)

	err = b.Measure(channel)
	require.Nil(err)

	wg.Wait()
}
