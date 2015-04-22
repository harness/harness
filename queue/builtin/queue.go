package builtin

import (
	"sync"

	"github.com/drone/drone/queue"
)

type Queue struct {
	sync.Mutex

	acks  map[*queue.Work]struct{}
	items map[*queue.Work]struct{}
	itemc chan *queue.Work
}

func New() *Queue {
	return &Queue{
		acks:  make(map[*queue.Work]struct{}),
		items: make(map[*queue.Work]struct{}),
		itemc: make(chan *queue.Work, 999),
	}
}

// Publish inserts work at the tail of this queue, waiting for
// space to become available if the queue is full.
func (q *Queue) Publish(work *queue.Work) error {
	q.Lock()
	q.items[work] = struct{}{}
	q.Unlock()
	q.itemc <- work
	return nil
}

// Remove removes the specified work item from this queue,
// if it is present.
func (q *Queue) Remove(work *queue.Work) error {
	return nil
}

// Pull retrieves and removes the head of this queue, waiting
// if necessary until work becomes available.
func (q *Queue) Pull() *queue.Work {
	work := <-q.itemc
	q.Lock()
	delete(q.items, work)
	q.Unlock()
	return work
}

// PullAct retrieves and removes the head of this queue, waiting
// if necessary until work becomes available. Items pull from the
// queue that aren't acknowledged will be pushed back to the queue
// again when the default acknowledgement deadline is reached.
func (q *Queue) PullAck() *queue.Work {
	work := q.Pull()
	q.Lock()
	q.acks[work] = struct{}{}
	q.Unlock()
	return work
}

// Ack acknowledges an item in the queue was processed.
func (q *Queue) Ack(work *queue.Work) error {
	q.Lock()
	delete(q.acks, work)
	q.Unlock()
	return nil
}

// Items returns a slice containing all of the work in this
// queue, in proper sequence.
func (q *Queue) Items() []*queue.Work {
	q.Lock()
	defer q.Unlock()
	items := []*queue.Work{}
	for work, _ := range q.items {
		items = append(items, work)
	}
	return items
}
