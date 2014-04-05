package incremental

import (
	"sync"
)

type Uint32 struct {
	increment uint32
	lock      sync.Mutex
}

// Next returns with an integer that is exactly one higher as the previous call to Next() for this Uint32
func (i *Uint32) Next() uint32 {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment++
	return i.increment
}

// Last returns the number (uint32) that was returned by the most recent call to this instance's Next()
func (i *Uint32) Last() uint32 {
	return i.increment
}

// Set changes the increment to given value, the succeeding call to Next() will return the given value+1
func (i *Uint32) Set(value uint32) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment = value
}
