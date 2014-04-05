package incremental

import (
	"sync"
)

type Int8 struct {
	increment int8
	lock      sync.Mutex
}

// Next returns with an integer that is exactly one higher as the previous call to Next() for this Int8
func (i *Int8) Next() int8 {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment++
	return i.increment
}

// Last returns the number (int8) that was returned by the most recent call to this instance's Next()
func (i *Int8) Last() int8 {
	return i.increment
}

// Set changes the increment to given value, the succeeding call to Next() will return the given value+1
func (i *Int8) Set(value int8) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment = value
}
