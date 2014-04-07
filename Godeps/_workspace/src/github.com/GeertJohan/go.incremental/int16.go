package incremental

import (
	"sync"
)

type Int16 struct {
	increment int16
	lock      sync.Mutex
}

// Next returns with an integer that is exactly one higher as the previous call to Next() for this Int16
func (i *Int16) Next() int16 {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment++
	return i.increment
}

// Last returns the number (int16) that was returned by the most recent call to this instance's Next()
func (i *Int16) Last() int16 {
	return i.increment
}

// Set changes the increment to given value, the succeeding call to Next() will return the given value+1
func (i *Int16) Set(value int16) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment = value
}
