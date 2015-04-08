package eventbus

import (
	"sync"
)

type EventBus struct {
	sync.Mutex
	subs map[chan *Event]bool
}

// New creates a new EventBus that manages a list of
// subscribers to which events are published.
func New() *EventBus {
	return &EventBus{
		subs: make(map[chan *Event]bool),
	}
}

// Subscribe adds the channel to the list of
// subscribers. Each subscriber in the list will
// receive broadcast events.
func (b *EventBus) Subscribe(c chan *Event) {
	b.Lock()
	b.subs[c] = true
	b.Unlock()
}

// Unsubscribe removes the channel from the
// list of subscribers.
func (b *EventBus) Unsubscribe(c chan *Event) {
	b.Lock()
	delete(b.subs, c)
	b.Unlock()
}

// Send dispatches a message to all subscribers.
func (b *EventBus) Send(event *Event) {
	b.Lock()
	defer b.Unlock()

	for s, _ := range b.subs {
		go func(c chan *Event) {
			defer recover()
			c <- event
		}(s)
	}
}
