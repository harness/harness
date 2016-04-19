package stream

import (
	"sync"
	"sync/atomic"
)

type broadcaster struct {
	sync.RWMutex
	closed uint32
	*sync.Cond
}

func newBroadcaster() *broadcaster {
	var b broadcaster
	b.Cond = sync.NewCond(b.RWMutex.RLocker())
	return &b
}

func (b *broadcaster) Wait() {
	if b.IsOpen() {
		b.Cond.Wait()
	}
}

func (b *broadcaster) IsOpen() bool {
	return atomic.LoadUint32(&b.closed) == 0
}

func (b *broadcaster) Close() error {
	atomic.StoreUint32(&b.closed, 1)
	b.Cond.Broadcast()
	return nil
}
