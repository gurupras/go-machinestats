package machinestats

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetstat(t *testing.T) {
	assert := assert.New(t)

	n := NetStat{}
	count, err := n.Measure(nil)
	assert.Nil(err)
	assert.NotZero(count)
}

var bmarkResult float64

func BenchmarkNetStat(b *testing.B) {
	// run the function b.N times
	netstat := NetStat{}
	fmt.Printf("\nN=%v\n", b.N)
	var r float64
	for n := 0; n < b.N; n++ {
		r, _ = netstat.Measure(nil)
	}
	bmarkResult = r
}
