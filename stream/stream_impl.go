package stream

import (
	"io"
	"sync"

	"github.com/djherbis/fscache"
)

var noexp fscache.Reaper

// New creates a new Mux using an in-memory filesystem.
func New() Mux {
	fs := fscache.NewMemFs()
	c, err := fscache.NewCache(fs, noexp)
	if err != nil {
		panic(err)
	}
	return &mux{c}
}

// New creates a new Mux using a persistent filesystem.
func NewFileSystem(path string) Mux {
	fs, err := fscache.NewFs(path, 0777)
	if err != nil {
		panic(err)
	}
	c, err := fscache.NewCache(fs, noexp)
	if err != nil {
		panic(err)
	}
	return &mux{c}
}

// mux wraps the default fscache.Cache to match the
// defined interface and to wrap the ReadCloser and
// WriteCloser to avoid panics when we over-aggressively
// close streams.
type mux struct {
	cache fscache.Cache
}

func (m *mux) Create(key string) (io.ReadCloser, io.WriteCloser, error) {
	rc, wc, err := m.cache.Get(key)
	if rc != nil {
		rc = &closeOnceReader{ReadCloser: rc}
	}
	if wc != nil {
		wc = &closeOnceWriter{WriteCloser: wc}
	}
	return rc, wc, err
}

func (m *mux) Open(key string) (io.ReadCloser, io.WriteCloser, error) {
	return m.Create(key)
}

func (m *mux) Exists(key string) bool {
	return m.cache.Exists(key)
}
func (m *mux) Remove(key string) error {
	return m.cache.Remove(key)
}

// closeOnceReader is a helper function that ensures
// the reader is only closed once. This is because
// attempting to close the fscache reader more than
// once results in a panic.
type closeOnceReader struct {
	io.ReadCloser
	once sync.Once
}

func (c *closeOnceReader) Close() error {
	c.once.Do(func() {
		c.ReadCloser.Close()
	})
	return nil
}

// closeOnceWriter is a helper function that ensures
// the writer is only closed once. This is because
// attempting to close the fscache writer more than
// once results in a panic.
type closeOnceWriter struct {
	io.WriteCloser
	once sync.Once
}

func (c *closeOnceWriter) Close() error {
	c.once.Do(func() {
		c.WriteCloser.Close()
	})
	return nil
}
