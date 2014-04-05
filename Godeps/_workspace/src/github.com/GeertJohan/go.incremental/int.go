package incremental

import (
	"sync"
)

type Int struct {
	increment int
	lock      sync.Mutex
}

// Next returns with an integer that is exactly one higher as the previous call to Next() for this Int
func (i *Int) Next() int {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment++
	return i.increment
}

// Last returns the number (int) that was returned by the most recent call to this instance's Next()
func (i *Int) Last() int {
	return i.increment
}

// Set changes the increment to given value, the succeeding call to Next() will return the given value+1
func (i *Int) Set(value int) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.increment = value
}
