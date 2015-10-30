package engine

import (
	"sync"
)

type eventbus struct {
	sync.Mutex
	subs map[chan *Event]bool
}

// New creates a new eventbus that manages a list of
// subscribers to which events are published.
func newEventbus() *eventbus {
	return &eventbus{
		subs: make(map[chan *Event]bool),
	}
}

// Subscribe adds the channel to the list of
// subscribers. Each subscriber in the list will
// receive broadcast events.
func (b *eventbus) subscribe(c chan *Event) {
	b.Lock()
	b.subs[c] = true
	b.Unlock()
}

// Unsubscribe removes the channel from the
// list of subscribers.
func (b *eventbus) unsubscribe(c chan *Event) {
	b.Lock()
	delete(b.subs, c)
	b.Unlock()
}

// Send dispatches a message to all subscribers.
func (b *eventbus) send(event *Event) {
	b.Lock()
	defer b.Unlock()

	for s := range b.subs {
		go func(c chan *Event) {
			defer recover()
			c <- event
		}(s)
	}
}
