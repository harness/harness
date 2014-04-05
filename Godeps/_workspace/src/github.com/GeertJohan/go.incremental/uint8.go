package incremental

import (
	"sync"
)

type Uint8 struct {
	increment uint8
	lock      sync.Mutex
}

// Next returns with an integer that is exactly one higher as the previous call to Next() for this Uint8
func (i *Uint8) Next() uint8 {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment++
	return i.increment
}

// Last returns the number (uint8) that was returned by the most recent call to this instance's Next()
func (i *Uint8) Last() uint8 {
	return i.increment
}

// Set changes the increment to given value, the succeeding call to Next() will return the given value+1
func (i *Uint8) Set(value uint8) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment = value
}
