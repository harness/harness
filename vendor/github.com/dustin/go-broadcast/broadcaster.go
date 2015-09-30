/*
Package broadcast provides pubsub of messages over channels.

A provider has a Broadcaster into which it Submits messages and into
which subscribers Register to pick up those messages.

*/
package broadcast

type broadcaster struct {
	input chan interface{}
	reg   chan chan<- interface{}
	unreg chan chan<- interface{}

	outputs map[chan<- interface{}]bool
}

// The Broadcaster interface describes the main entry points to
// broadcasters.
type Broadcaster interface {
	// Register a new channel to receive broadcasts
	Register(chan<- interface{})
	// Unregister a channel so that it no longer receives broadcasts.
	Unregister(chan<- interface{})
	// Shut this broadcaster down.
	Close() error
	// Submit a new object to all subscribers
	Submit(interface{})
}

func (b *broadcaster) broadcast(m interface{}) {
	for ch := range b.outputs {
		ch <- m
	}
}

func (b *broadcaster) run() {
	for {
		select {
		case m := <-b.input:
			b.broadcast(m)
		case ch, ok := <-b.reg:
			if ok {
				b.outputs[ch] = true
			} else {
				return
			}
		case ch := <-b.unreg:
			delete(b.outputs, ch)
		}
	}
}

// NewBroadcaster creates a new broadcaster with the given input
// channel buffer length.
func NewBroadcaster(buflen int) Broadcaster {
	b := &broadcaster{
		input:   make(chan interface{}, buflen),
		reg:     make(chan chan<- interface{}),
		unreg:   make(chan chan<- interface{}),
		outputs: make(map[chan<- interface{}]bool),
	}

	go b.run()

	return b
}

func (b *broadcaster) Register(newch chan<- interface{}) {
	b.reg <- newch
}

func (b *broadcaster) Unregister(newch chan<- interface{}) {
	b.unreg <- newch
}

func (b *broadcaster) Close() error {
	close(b.reg)
	return nil
}

func (b *broadcaster) Submit(m interface{}) {
	if b != nil {
		b.input <- m
	}
}
