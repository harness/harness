package event

import (
	"container/list"
	"strings"
	"sync"
	"sync/atomic"
)

type HandlerID uint32

var hidCounter uint32 = 0

func NewHandlerID() HandlerID {
	return HandlerID(atomic.AddUint32(&hidCounter, 1))
}

type Handler interface {
	Run(...interface{})
	Id() HandlerID
}

type basicHandler struct {
	fn func(...interface{})
	id HandlerID
}

func (h *basicHandler) Run(ev ...interface{}) {
	h.fn(ev...)
}

func (h *basicHandler) Id() HandlerID {
	return h.id
}

func NewHandler(h func(...interface{})) Handler {
	return &basicHandler{h, NewHandlerID()}
}

type EventDispatcher interface {
	Dispatch(name string, ev ...interface{})
}

type EventRegistry interface {
	AddHandler(h Handler, names ...string)
	DelHandler(h Handler, names ...string)
	Dispatch(name string, ev ...interface{})
	ClearEvents(name string)
}

type registry struct {
	// Event registry as a lockable map of linked-lists
	sync.RWMutex
	events     map[string]*list.List
	dispatcher func(r *registry, name string, ev ...interface{})
}

func NewRegistry() *registry {
	r := &registry{events: make(map[string]*list.List)}
	r.Parallel()
	return r
}

func (r *registry) AddHandler(h Handler, names ...string) {
	if len(names) == 0 {
		return
	}
	r.Lock()
	defer r.Unlock()
N:
	for _, name := range names {
		name = strings.ToLower(name)
		if _, ok := r.events[name]; !ok {
			r.events[name] = list.New()
		}
		for e := r.events[name].Front(); e != nil; e = e.Next() {
			// Check we're not adding a duplicate handler to this event
			if e.Value.(Handler).Id() == h.Id() {
				continue N
			}
		}
		r.events[name].PushBack(h)
	}
}

func _del(l *list.List, id HandlerID) bool {
	for e := l.Front(); e != nil; e = e.Next() {
		if e.Value.(Handler).Id() == id {
			l.Remove(e)
		}
	}
	return l.Len() == 0
}

func (r *registry) DelHandler(h Handler, names ...string) {
	r.Lock()
	defer r.Unlock()
	if len(names) == 0 {
		for name, l := range r.events {
			if _del(l, h.Id()) {
				delete(r.events, name)
			}
		}
	} else {
		for _, name := range names {
			name = strings.ToLower(name)
			if l, ok := r.events[name]; ok {
				if _del(l, h.Id()) {
					delete(r.events, name)
				}
			}
		}
	}
}

func (r *registry) Dispatch(name string, ev ...interface{}) {
	r.dispatcher(r, strings.ToLower(name), ev...)
}

func (r *registry) ClearEvents(name string) {
	name = strings.ToLower(name)
	r.Lock()
	defer r.Unlock()
	if l, ok := r.events[name]; ok {
		l.Init() // I hope this is enough to GC all list elements.
		delete(r.events, name)
	}
}

func (r *registry) Parallel() {
	r.dispatcher = (*registry).parallelDispatch
}

func (r *registry) Serial() {
	r.dispatcher = (*registry).serialDispatch
}

func (r *registry) parallelDispatch(name string, ev ...interface{}) {
	r.RLock()
	defer r.RUnlock()
	if l, ok := r.events[name]; ok {
		for e := l.Front(); e != nil; e = e.Next() {
			h := e.Value.(Handler)
			go h.Run(ev...)
		}
	}
}

func (r *registry) serialDispatch(name string, ev ...interface{}) {
	r.RLock()
	defer r.RUnlock()
	if l, ok := r.events[name]; ok {
		hlist := make([]Handler, l.Len())
		for e, i := l.Front(), 0; e != nil; e, i = e.Next(), i+1 {
			hlist[i] = e.Value.(Handler)
		}
		go func() {
			for _, h := range hlist {
				h.Run(ev...)
			}
		}()
	}
}
