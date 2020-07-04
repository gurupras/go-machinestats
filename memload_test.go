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

	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var meminfoStr = strings.TrimSpace(`
MemTotal:       32896100 kB
MemFree:          959580 kB
MemAvailable:   14431764 kB
Buffers:         6626224 kB
Cached:          6413704 kB
SwapCached:          796 kB
Active:         24800088 kB
Inactive:        4506504 kB
Active(anon):   15157992 kB
Inactive(anon):  1490512 kB
Active(file):    9642096 kB
Inactive(file):  3015992 kB
Unevictable:         272 kB
Mlocked:             272 kB
SwapTotal:      67108860 kB
SwapFree:       67039984 kB
Dirty:               480 kB
Writeback:             0 kB
AnonPages:      16266668 kB
Mapped:          2098564 kB
Shmem:            970048 kB
KReclaimable:    1278036 kB
Slab:            1707056 kB
SReclaimable:    1278036 kB
SUnreclaim:       429020 kB
KernelStack:       53696 kB
PageTables:       275604 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:    83556908 kB
Committed_AS:   43814344 kB
VmallocTotal:   34359738367 kB
VmallocUsed:       91896 kB
VmallocChunk:          0 kB
Percpu:            31744 kB
HardwareCorrupted:     0 kB
AnonHugePages:         0 kB
ShmemHugePages:        0 kB
ShmemPmdMapped:        0 kB
CmaTotal:              0 kB
CmaFree:               0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
Hugetlb:               0 kB
DirectMap4k:     2578796 kB
DirectMap2M:    30924800 kB
DirectMap1G:     1048576 kB
`)

func TestMemLoadStat(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "test-dir")
	require.Nil(err)
	require.NotNil(dir)

	path := path.Join(dir, "meminfo")

	err = ioutil.WriteFile(path, []byte(meminfoStr), 0644)
	require.Nil(err)

	fs, err := procfs.NewFS(dir)
	require.Nil(err)

	t.Run("Test procfs parsing", func(t *testing.T) {
		memLoadStat, err := NewMemLoadStat(&fs)
		require.Nil(err)
		channel := make(chan Measurement)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := <-channel
			value := data.Value()
			diff := math.Abs(56.1292554 - value.(float64))
			assert.True(diff < 1e-7, fmt.Sprintf("diff was: %v", diff))
		}()
		memLoadStat.Measure(channel)
		wg.Wait()
	})
}
