package builtin

import (
	"sync"

	"github.com/drone/drone/pkg/bus"
)

type Bus struct {
	sync.Mutex
	subs map[chan *bus.Event]bool
}

// New creates a new Bus that manages a list of
// subscribers to which events are published.
func New() *Bus {
	return &Bus{
		subs: make(map[chan *bus.Event]bool),
	}
}

// Subscribe adds the channel to the list of
// subscribers. Each subscriber in the list will
// receive broadcast events.
func (b *Bus) Subscribe(c chan *bus.Event) {
	b.Lock()
	b.subs[c] = true
	b.Unlock()
}

// Unsubscribe removes the channel from the
// list of subscribers.
func (b *Bus) Unsubscribe(c chan *bus.Event) {
	b.Lock()
	delete(b.subs, c)
	b.Unlock()
}

// Send dispatches a message to all subscribers.
func (b *Bus) Send(event *bus.Event) {
	b.Lock()
	defer b.Unlock()

	for s := range b.subs {
		go func(c chan *bus.Event) {
			defer recover()
			c <- event
		}(s)
	}
}
