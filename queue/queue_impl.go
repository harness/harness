package queue

import (
	"reflect"
	"sync"
)

type queue struct {
	sync.Mutex

	items map[*Work]struct{}
	itemc map[string]chan *Work
}

func New() Queue {
	return newQueue()
}

func newQueue() *queue {
	return &queue{
		items: make(map[*Work]struct{}),
		itemc: make(map[string]chan *Work),
	}
}

func (q *queue) Publish(work *Work) error {
	q.Lock()
	if work.Label == "" {
		work.Label = DefaultLabel
	}
	if _, ok := q.itemc[work.Label]; !ok {
		q.itemc[work.Label] = make(chan *Work, 999)
	}
	q.items[work] = struct{}{}
	q.Unlock()
	q.itemc[work.Label] <- work
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
		case item := <-q.itemc[work.Label]:
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
		q.itemc[work.Label] <- item
	}
	return nil
}

func (q *queue) PullWithLabels(labels []string) *Work {
	cases := make([]reflect.SelectCase, 0)
	for _, label := range labels {
		if _, ok := q.itemc[label]; ok {
			cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(q.itemc[label])})
		}
	}
	_, value, _ := reflect.Select(cases)
	work := value.Interface().(*Work)
	return work
}

func (q *queue) Pull() *Work {
	return q.PullWithLabels([]string{DefaultLabel})
}

func (q *queue) PullClose(cn CloseNotifier) *Work {
	return q.PullCloseWithLabels([]string{DefaultLabel}, cn)
}

func (q *queue) PullCloseWithLabels(labels []string, cn CloseNotifier) *Work {
	cases := make([]reflect.SelectCase, 0)
	for _, label := range labels {
		if _, ok := q.itemc[label]; ok {
			cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(q.itemc[label])})
		}
	}
	cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cn.CloseNotify())})
	sel, value, _ := reflect.Select(cases)
	if sel == len(cases)-1 {
		return nil
	} else {
		work := value.Interface().(*Work)
		return work
	}
}
