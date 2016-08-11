package queue

import "sync"

type queue struct {
	sync.Mutex

	items map[int64]struct{}
	itemc chan *Work
}

// New creates a Queue instance.
func New() Queue {
	return newQueue()
}

func newQueue() *queue {
	return &queue{
		items: make(map[int64]struct{}),
		itemc: make(chan *Work, 999),
	}
}

func (q *queue) Publish(work *Work) error {
	q.Lock()
	q.items[work.Job.ID] = struct{}{}
	q.Unlock()
	q.itemc <- work
	return nil
}

func (q *queue) Remove(id int64) error {
	q.Lock()
	defer q.Unlock()

	_, ok := q.items[id]
	if !ok {
		return ErrNotFound
	}
	var items []*Work

	// loop through and drain all items
	// from the
drain:
	for {
		select {
		case item := <-q.itemc:
			items = append(items, item)
		default:
			break drain
		}
	}

	// re-add all items to the queue except
	// the item we're trying to remove
	for _, item := range items {
		if item.Job.ID == id {
			delete(q.items, id)
			continue
		}
		q.itemc <- item
	}
	return nil
}

func (q *queue) Pull() *Work {
	work := <-q.itemc
	q.Lock()
	delete(q.items, work.Job.ID)
	q.Unlock()
	return work
}

func (q *queue) PullClose(cn CloseNotifier) *Work {
	for {
		select {
		case <-cn.CloseNotify():
			return nil
		case work := <-q.itemc:
			q.Lock()
			delete(q.items, work.Job.ID)
			q.Unlock()
			return work
		}
	}
}

func (q *queue) IndexOf(id int64) int {
	var items []*Work

	// loop through and drain all items
	// from the queue
	//
	// Cant just range over the channel as
	// it will consume the channel
drain:
	for {
		select {
		case item := <-q.itemc:
			items = append(items, item)
		default:
			break drain
		}
	}

	index := -1

	// re-add all items to the queue
	for i, item := range items {
		if item.Job.ID == id {
			index = i
		}
		q.itemc <- item
	}
	return index
}

func (q *queue) Length() int {
	return len(q.itemc)
}
