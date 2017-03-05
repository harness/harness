package pubsub

import "sync"

type topic struct {
	sync.Mutex

	name string
	done chan bool
	subs map[*subscriber]struct{}
}

func newTopic(dest string) *topic {
	return &topic{
		name: dest,
		done: make(chan bool),
		subs: make(map[*subscriber]struct{}),
	}
}

func (t *topic) subscribe(s *subscriber) {
	t.Lock()
	t.subs[s] = struct{}{}
	t.Unlock()
}

func (t *topic) unsubscribe(s *subscriber) {
	t.Lock()
	delete(t.subs, s)
	t.Unlock()
}

func (t *topic) publish(m Message) {
	t.Lock()
	for s := range t.subs {
		go s.receiver(m)
	}
	t.Unlock()
}

func (t *topic) close() {
	t.Lock()
	close(t.done)
	t.Unlock()
}
