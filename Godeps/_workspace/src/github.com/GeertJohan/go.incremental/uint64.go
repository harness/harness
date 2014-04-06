package incremental

import (
	"sync"
)

type Uint64 struct {
	increment uint64
	lock      sync.Mutex
}

// Next returns with an integer that is exactly one higher as the previous call to Next() for this Uint64
func (i *Uint64) Next() uint64 {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment++
	return i.increment
}

// Last returns the number (uint64) that was returned by the most recent call to this instance's Next()
func (i *Uint64) Last() uint64 {
	return i.increment
}

// Set changes the increment to given value, the succeeding call to Next() will return the given value+1
func (i *Uint64) Set(value uint64) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment = value
}
