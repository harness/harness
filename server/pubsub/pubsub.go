package pubsub

import (
	"sync"

	"code.google.com/p/go.net/context"
)

type PubSub struct {
	sync.Mutex

	// In-memory list of all channels being managed by the broker.
	channels map[interface{}]*Channel
}

// NewPubSub creates a new instance of the PubSub type
// and returns a pointer.
func NewPubSub() *PubSub {
	return &PubSub{
		channels: make(map[interface{}]*Channel),
	}
}

// Lookup performs a thread safe operation to return a pointer
// to an existing Channel object with the given key. If the
// Channel does not exist a nil value is returned.
func (b *PubSub) Lookup(key interface{}) *Channel {
	b.Lock()
	defer b.Unlock()

	// find the channel in the existing list
	return b.channels[key]
}

// Register performs a thread safe operation to return a pointer
// to a Channel object for the given key. The Channel is created
// if it does not yet exist.
func (b *PubSub) Register(key interface{}) *Channel {
	return b.RegisterOpts(key, DefaultOpts)
}

// Register performs a thread safe operation to return a pointer
// to a Channel object for the given key. The Channel is created
// if it does not yet exist using custom options.
func (b *PubSub) RegisterOpts(key interface{}, opts *Opts) *Channel {
	b.Lock()
	defer b.Unlock()

	// find the channel in the existing list
	c, ok := b.channels[key]
	if ok {
		return c
	}

	// create the channel and register
	// with the pubsub server
	c = NewChannel(opts)
	b.channels[key] = c
	go c.start()
	return c
}

// Unregister performs a thread safe operation to delete the
// Channel with the given key.
func (b *PubSub) Unregister(key interface{}) {
	b.Lock()
	defer b.Unlock()

	// find the channel in the existing list
	c, ok := b.channels[key]
	if !ok {
		return
	}
	c.Close()
	delete(b.channels, key)
	return
}

// Lookup performs a thread safe operation to return a pointer
// to an existing Channel object with the given key. If the
// Channel does not exist a nil value is returned.
func Lookup(c context.Context, key interface{}) *Channel {
	return FromContext(c).Lookup(key)
}

// Register performs a thread safe operation to return a pointer
// to a Channel object for the given key. The Channel is created
// if it does not yet exist.
func Register(c context.Context, key interface{}) *Channel {
	return FromContext(c).Register(key)
}

// Register performs a thread safe operation to return a pointer
// to a Channel object for the given key. The Channel is created
// if it does not yet exist using custom options.
func RegisterOpts(c context.Context, key interface{}, opts *Opts) *Channel {
	return FromContext(c).RegisterOpts(key, opts)
}

// Unregister performs a thread safe operation to delete the
// Channel with the given key.
func Unregister(c context.Context, key interface{}) {
	FromContext(c).Unregister(key)
}
