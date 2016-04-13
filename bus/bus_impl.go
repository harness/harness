package bus

import (
	"sync"
)

type eventbus struct {
	sync.Mutex
	subs map[chan *Event]bool
}

// New creates a simple event bus that manages a list of
// subscribers to which events are published.
func New() Bus {
	return newEventbus()
}

func newEventbus() *eventbus {
	return &eventbus{
		subs: make(map[chan *Event]bool),
	}
}

func (b *eventbus) Subscribe(c chan *Event) {
	b.Lock()
	b.subs[c] = true
	b.Unlock()
}

func (b *eventbus) Unsubscribe(c chan *Event) {
	b.Lock()
	delete(b.subs, c)
	b.Unlock()
}

func (b *eventbus) Publish(event *Event) {
	b.Lock()
	defer b.Unlock()

	for s := range b.subs {
		go func(c chan *Event) {
			defer recover()
			c <- event
		}(s)
	}
}
