package incremental

import (
	"sync"
)

type Int32 struct {
	increment int32
	lock      sync.Mutex
}

// Next returns with an integer that is exactly one higher as the previous call to Next() for this Int32
func (i *Int32) Next() int32 {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment++
	return i.increment
}

// Last returns the number (int32) that was returned by the most recent call to this instance's Next()
func (i *Int32) Last() int32 {
	return i.increment
}

// Set changes the increment to given value, the succeeding call to Next() will return the given value+1
func (i *Int32) Set(value int32) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment = value
}
