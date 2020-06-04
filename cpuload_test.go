package machinestats

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleProcStr = `cpu  16237003 488024 7674741 943706235 1071665 0 1072139 0 0 0
cpu0 639241 21228 326322 39345419 43681 0 685277 0 0 0
cpu1 611316 23634 309018 39395345 46618 0 191330 0 0 0
cpu2 660324 20703 308762 39355439 47733 0 78288 0 0 0
cpu3 681919 21493 325023 39313594 45523 0 43927 0 0 0
cpu4 648886 20282 324626 39356106 40980 0 15617 0 0 0
cpu5 642872 20554 321832 39351907 44908 0 10070 0 0 0
cpu6 688975 21220 377022 39251627 51747 0 4367 0 0 0
cpu7 669572 21184 334274 39326310 41296 0 4828 0 0 0
`

const sampleProcStr2 = `cpu  16334594 488336 7714010 947760072 1075645 0 1077789 0 0 0
cpu0 643042 21308 328043 39514523 43789 0 689301 0 0 0
cpu1 615338 23684 310567 39564404 46821 0 192208 0 0 0
cpu2 664600 20703 310336 39524229 47913 0 78571 0 0 0
cpu3 686230 21546 326600 39482489 45603 0 44100 0 0 0
cpu4 653093 20282 326248 39524930 41199 0 15662 0 0 0
cpu5 646487 20560 323492 39521227 45076 0 10102 0 0 0
cpu6 694338 21222 379024 39418809 51932 0 4390 0 0 0
cpu7 673592 21188 336084 39495119 41442 0 4846 0 0 0
`

var busy = []float64{
	0.034000065228171235, // Overall
	0.05382524966729666,
	0.03697634856424349,
	0.03502509951285815,
	0.03491938385621027,
	0.033581641578577265,
	0.03039456296016613,
	0.04228729035174557,
	0.033476920260630295,
}

func TestReadProcStat(t *testing.T) {

}

func TestCpuLoadStat(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	dir, err := ioutil.TempDir(os.TempDir(), "test-dir")
	require.Nil(err)
	require.NotNil(dir)

	path := path.Join(dir, "stat")

	err = ioutil.WriteFile(path, []byte(sampleProcStr), 0644)
	require.Nil(err)

	fs, err := procfs.NewFS(dir)
	require.Nil(err)

	t.Run("Individual CPU loads", func(t *testing.T) {
		ncpus := 8

		cpuLoadStat, err := NewCPULoadStat(&fs)
		require.Nil(err)
		require.NotNil(cpuLoadStat)

		channel := make(chan Measurement, 0)

		wg := sync.WaitGroup{}
		wg.Add(ncpus + 1)
		go func() {
			for idx := 0; idx < ncpus+1; idx++ {
				measurement := <-channel
				name := measurement.Name()
				value := measurement.Value().(float64)
				diff := math.Abs(busy[idx] - value)
				assert.True(diff < 1e-7, fmt.Sprintf("[%v]: diff was: %v", name, diff))
				wg.Done()
			}
		}()
		cpuLoadStat.Measure(channel)
		// Now, write the new values into the file
		err = ioutil.WriteFile(path, []byte(sampleProcStr2), 0644)
		require.Nil(err)
		cpuLoadStat.Measure(channel)
		wg.Wait()
		close(channel)
	})
}
