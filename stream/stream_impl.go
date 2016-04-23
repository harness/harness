package stream

import (
	"fmt"
	"io"
	"sync"
)

type stream struct {
	sync.Mutex
	writers map[string]*writer
}

// New returns a new in-memory implementation of Stream.
func New() Stream {
	return &stream{
		writers: map[string]*writer{},
	}
}

// Reader returns an io.Reader for reading from to the stream.
func (s *stream) Reader(name string) (io.ReadCloser, error) {
	s.Lock()
	defer s.Unlock()

	if !s.exists(name) {
		return nil, fmt.Errorf("stream: cannot read stream %s, not found", name)
	}
	return s.writers[name].Reader()
}

// Writer returns an io.WriteCloser for writing to the stream.
func (s *stream) Writer(name string) (io.WriteCloser, error) {
	s.Lock()
	defer s.Unlock()

	if !s.exists(name) {
		return nil, fmt.Errorf("stream: cannot write stream %s, not found", name)
	}
	return s.writers[name], nil
}

// Create creates a new stream.
func (s *stream) Create(name string) error {
	s.Lock()
	defer s.Unlock()

	if s.exists(name) {
		return fmt.Errorf("stream: cannot create stream %s, already exists", name)
	}

	s.writers[name] = newWriter()
	return nil
}

// Delete deletes the stream by key.
func (s *stream) Delete(name string) error {
	s.Lock()
	defer s.Unlock()

	if !s.exists(name) {
		return fmt.Errorf("stream: cannot delete stream %s, not found", name)
	}
	w := s.writers[name]
	delete(s.writers, name)
	return w.Close()
}

func (s *stream) exists(name string) bool {
	_, exists := s.writers[name]
	return exists
}
