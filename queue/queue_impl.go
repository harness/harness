package queue

import "sync"

type queue struct {
	sync.Mutex

	items map[*Work]struct{}
	itemc chan *Work
}

func New() Queue {
	return newQueue()
}

func newQueue() *queue {
	return &queue{
		items: make(map[*Work]struct{}),
		itemc: make(chan *Work, 999),
	}
}

func (q *queue) Publish(work *Work) error {
	q.Lock()
	q.items[work] = struct{}{}
	q.Unlock()
	q.itemc <- work
	return nil
}

func (q *queue) Remove(work *Work) error {
	q.Lock()
	defer q.Unlock()

	_, ok := q.items[work]
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
		if item == work {
			delete(q.items, work)
			continue
		}
		q.itemc <- item
	}
	return nil
}

func (q *queue) Pull() *Work {
	work := <-q.itemc
	q.Lock()
	delete(q.items, work)
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
			delete(q.items, work)
			q.Unlock()
			return work
		}
	}
}
