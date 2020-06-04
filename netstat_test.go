package machinestats

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNetstat(t *testing.T) {
	require := require.New(t)

	n, err := NewNetStat(nil)
	require.Nil(err)

	channel := make(chan Measurement, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		measurement := <-channel
		value := measurement.Value().(int)
		require.NotZero(value)
	}()
	err = n.Measure(channel)
	require.Nil(err)
	wg.Wait()
	close(channel)
}

func BenchmarkNetStat(b *testing.B) {
	// run the function b.N times
	require := require.New(b)
	netstat, err := NewNetStat(nil)
	require.Nil(err)

	channel := make(chan Measurement, 0)
	count := 0
	go func() {
		for range channel {
			count++
		}
	}()
	for n := 0; n < b.N; n++ {
		netstat.Measure(channel)
	}
	close(channel)
}
