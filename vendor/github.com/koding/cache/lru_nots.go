package cache

import (
	"container/list"
)

// LRUNoTS Discards the least recently used items first. This algorithm
// requires keeping track of what was used when.
type LRUNoTS struct {
	// list holds all items in a linked list, for finding the `tail` of the list
	list *list.List

	// cache holds the all cache values
	cache Cache

	// size holds the limit of the LRU cache
	size int
}

// kv is an helper struct for keeping track of the key for the list item. Only
// place where we need the key of a value is while removing the last item from
// linked list, for other cases, all operations alread have the key
type kv struct {
	k string
	v interface{}
}

// NewLRUNoTS creates a new LRU cache struct for further cache operations. Size
// is used for limiting the upper bound of the cache
func NewLRUNoTS(size int) Cache {
	if size < 1 {
		panic("invalid cache size")
	}

	return &LRUNoTS{
		list:  list.New(),
		cache: NewMemoryNoTS(),
		size:  size,
	}
}

// Get returns the value of a given key if it exists, every get item will be
// moved to the head of the linked list for keeping track of least recent used
// item
func (l *LRUNoTS) Get(key string) (interface{}, error) {
	res, err := l.cache.Get(key)
	if err != nil {
		return nil, err
	}

	elem := res.(*list.Element)
	// move found item to the head
	l.list.MoveToFront(elem)

	return elem.Value.(*kv).v, nil
}

// Set sets or overrides the given key with the given value, every set item will
// be moved or prepended to the head of the linked list for keeping track of
// least recent used item. When the cache is full, last item of the linked list
// will be evicted from the cache
func (l *LRUNoTS) Set(key string, val interface{}) error {
	// try to get item
	res, err := l.cache.Get(key)
	if err != nil && err != ErrNotFound {
		return err
	}

	var elem *list.Element

	// if elem is not in the cache, push it to front of the list
	if err == ErrNotFound {
		elem = l.list.PushFront(&kv{k: key, v: val})
	} else {
		// if elem is in the cache, update the data and move it the front
		elem = res.(*list.Element)

		// update the  data
		elem.Value.(*kv).v = val

		// item already exists, so move it to the front of the list
		l.list.MoveToFront(elem)
	}

	// in any case, set the item to the cache
	err = l.cache.Set(key, elem)
	if err != nil {
		return err
	}

	// if the cache is full, evict last entry
	if l.list.Len() > l.size {
		// remove last element from cache
		return l.removeElem(l.list.Back())
	}

	return nil
}

// Delete deletes the given key-value pair from cache, this function doesnt
// return an error if item is not in the cache
func (l *LRUNoTS) Delete(key string) error {
	res, err := l.cache.Get(key)
	if err != nil && err != ErrNotFound {
		return err
	}

	// item already deleted
	if err == ErrNotFound {
		// surpress not found errors
		return nil
	}

	elem := res.(*list.Element)

	return l.removeElem(elem)
}

func (l *LRUNoTS) removeElem(e *list.Element) error {
	l.list.Remove(e)
	return l.cache.Delete(e.Value.(*kv).k)
}
