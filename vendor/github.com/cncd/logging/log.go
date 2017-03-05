package logging

import (
	"context"
	"io"
	"sync"
)

// TODO (bradrydzewski) writing to subscribers is currently a blocking
// operation and does not protect against slow clients from locking
// the stream. This should be resolved.

// TODO (bradrydzewski) implement a mux.Info to fetch information and
// statistics for the multiplexier. Streams, subscribers, etc
// mux.Info()

// TODO (bradrydzewski) refactor code to place publisher and subscriber
// operations in separate files with more encapsulated logic.
// sub.push()
// sub.join()
// sub.start()... event loop

type subscriber struct {
	handler Handler
}

type stream struct {
	sync.Mutex

	path string
	hist []*Entry
	subs map[*subscriber]struct{}
	done chan struct{}
	wait sync.WaitGroup
}

type log struct {
	sync.Mutex

	streams map[string]*stream
}

// New returns a new logger.
func New() Log {
	return &log{
		streams: map[string]*stream{},
	}
}

func (l *log) Open(c context.Context, path string) error {
	l.Lock()
	_, ok := l.streams[path]
	if !ok {
		l.streams[path] = &stream{
			path: path,
			subs: make(map[*subscriber]struct{}),
			done: make(chan struct{}),
		}
	}
	l.Unlock()
	return nil
}

func (l *log) Write(c context.Context, path string, entry *Entry) error {
	l.Lock()
	s, ok := l.streams[path]
	l.Unlock()
	if !ok {
		return ErrNotFound
	}
	s.Lock()
	s.hist = append(s.hist, entry)
	for sub := range s.subs {
		go sub.handler(entry)
	}
	s.Unlock()
	return nil
}

func (l *log) Tail(c context.Context, path string, handler Handler) error {
	l.Lock()
	s, ok := l.streams[path]
	l.Unlock()
	if !ok {
		return ErrNotFound
	}

	sub := &subscriber{
		handler: handler,
	}
	s.Lock()
	if len(s.hist) != 0 {
		sub.handler(s.hist...)
	}
	s.subs[sub] = struct{}{}
	s.Unlock()

	select {
	case <-c.Done():
	case <-s.done:
	}

	s.Lock()
	delete(s.subs, sub)
	s.Unlock()
	return nil
}

func (l *log) Close(c context.Context, path string) error {
	l.Lock()
	s, ok := l.streams[path]
	l.Unlock()
	if !ok {
		return ErrNotFound
	}

	s.Lock()
	close(s.done)
	s.Unlock()

	l.Lock()
	delete(l.streams, path)
	l.Unlock()
	return nil
}

func (l *log) Snapshot(c context.Context, path string, w io.Writer) error {
	l.Lock()
	s, ok := l.streams[path]
	l.Unlock()
	if !ok {
		return ErrNotFound
	}
	s.Lock()
	for _, entry := range s.hist {
		w.Write(entry.Data)
		w.Write(cr)
	}
	s.Unlock()
	return nil
}

var cr = []byte{'\n'}
