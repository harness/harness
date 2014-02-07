package queue

import (
	"runtime"
)

func init() {
	// get the number of CPUs. Since builds
	// tend to be CPU-intensive we should only
	// execute 1 build per CPU.
	ncpu := runtime.NumCPU()

	// must be at least 1
	if ncpu < 1 {
		ncpu = 1
	}

	// spawn a worker for each CPU
	for i := 0; i < ncpu; i++ {
		go work()
	}
}
