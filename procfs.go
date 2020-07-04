package machinestats

import (
	"sync"

	"github.com/prometheus/procfs"
)

var procFS *procfs.FS

var once sync.Once

func setupProcFS() error {
	var err error
	once.Do(func() {
		_procfs, _err := procfs.NewFS("/proc")
		err = _err
		procFS = &_procfs
	})
	return err
}
