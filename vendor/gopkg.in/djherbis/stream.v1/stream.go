// Package stream provides a way to read and write to a synchronous buffered pipe, with multiple reader support.
package stream

import (
	"errors"
	"sync"
)

// ErrRemoving is returned when requesting a Reader on a Stream which is being Removed.
var ErrRemoving = errors.New("cannot open a new reader while removing file")

// Stream is used to concurrently Write and Read from a File.
type Stream struct {
	grp      sync.WaitGroup
	b        *broadcaster
	file     File
	fs       FileSystem
	removing chan struct{}
}

// New creates a new Stream from the StdFileSystem with Name "name".
func New(name string) (*Stream, error) {
	return NewStream(name, StdFileSystem)
}

// NewStream creates a new Stream with Name "name" in FileSystem fs.
func NewStream(name string, fs FileSystem) (*Stream, error) {
	f, err := fs.Create(name)
	sf := &Stream{
		file:     f,
		fs:       fs,
		b:        newBroadcaster(),
		removing: make(chan struct{}),
	}
	sf.inc()
	return sf, err
}

// Name returns the name of the underlying File in the FileSystem.
func (s *Stream) Name() string { return s.file.Name() }

// Write writes p to the Stream. It's concurrent safe to be called with Stream's other methods.
func (s *Stream) Write(p []byte) (int, error) {
	defer s.b.Broadcast()
	s.b.Lock()
	defer s.b.Unlock()
	return s.file.Write(p)
}

// Close will close the active stream. This will cause Readers to return EOF once they have
// read the entire stream.
func (s *Stream) Close() error {
	defer s.dec()
	defer s.b.Close()
	s.b.Lock()
	defer s.b.Unlock()
	return s.file.Close()
}

// Remove will block until the Stream and all its Readers have been Closed,
// at which point it will delete the underlying file. NextReader() will return
// ErrRemoving if called after Remove.
func (s *Stream) Remove() error {
	close(s.removing)
	s.grp.Wait()
	return s.fs.Remove(s.file.Name())
}

// NextReader will return a concurrent-safe Reader for this stream. Each Reader will
// see a complete and independent view of the stream, and can Read will the stream
// is written to.
func (s *Stream) NextReader() (*Reader, error) {
	s.inc()

	select {
	case <-s.removing:
		s.dec()
		return nil, ErrRemoving
	default:
	}

	file, err := s.fs.Open(s.file.Name())
	if err != nil {
		s.dec()
		return nil, err
	}

	return &Reader{file: file, s: s}, nil
}

func (s *Stream) inc() { s.grp.Add(1) }
func (s *Stream) dec() { s.grp.Done() }
