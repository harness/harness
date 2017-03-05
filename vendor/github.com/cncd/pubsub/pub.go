package pubsub

import (
	"context"
	"sync"
)

type subscriber struct {
	receiver Receiver
}

type publisher struct {
	sync.Mutex

	topics map[string]*topic
}

// New creates an in-memory publisher.
func New() Publisher {
	return &publisher{
		topics: make(map[string]*topic),
	}
}

func (p *publisher) Create(c context.Context, dest string) error {
	p.Lock()
	t, ok := p.topics[dest]
	if !ok {
		t = newTopic(dest)
		p.topics[dest] = t
	}
	p.Unlock()
	return nil
}

func (p *publisher) Publish(c context.Context, dest string, message Message) error {
	p.Lock()
	t, ok := p.topics[dest]
	p.Unlock()
	if !ok {
		return ErrNotFound
	}
	t.publish(message)
	return nil
}

func (p *publisher) Subscribe(c context.Context, dest string, receiver Receiver) error {
	p.Lock()
	t, ok := p.topics[dest]
	p.Unlock()
	if !ok {
		return ErrNotFound
	}
	s := &subscriber{
		receiver: receiver,
	}
	t.subscribe(s)
	select {
	case <-c.Done():
	case <-t.done:
	}
	t.unsubscribe(s)
	return nil
}

func (p *publisher) Remove(c context.Context, dest string) error {
	p.Lock()
	t, ok := p.topics[dest]
	if ok {
		delete(p.topics, dest)
		t.close()
	}
	p.Unlock()
	return nil
}
