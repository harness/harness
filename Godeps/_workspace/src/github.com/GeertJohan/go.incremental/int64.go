package incremental

import (
	"sync"
)

type Int64 struct {
	increment int64
	lock      sync.Mutex
}

// Next returns with an integer that is exactly one higher as the previous call to Next() for this Int64
func (i *Int64) Next() int64 {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment++
	return i.increment
}

// Last returns the number (int64) that was returned by the most recent call to this instance's Next()
func (i *Int64) Last() int64 {
	return i.increment
}

// Set changes the increment to given value, the succeeding call to Next() will return the given value+1
func (i *Int64) Set(value int64) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment = value
}
