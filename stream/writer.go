package stream

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"
)

type writer struct {
	sync.RWMutex
	*sync.Cond

	buffer bytes.Buffer
	closed uint32
}

func newWriter() *writer {
	var w writer
	w.Cond = sync.NewCond(w.RWMutex.RLocker())
	return &w
}

func (w *writer) Write(p []byte) (n int, err error) {
	defer w.Broadcast()
	w.Lock()
	defer w.Unlock()
	if w.Closed() {
		return 0, io.EOF
	}
	return w.buffer.Write(p)
}

func (w *writer) Reader() (io.ReadCloser, error) {
	return &reader{w: w}, nil
}

func (w *writer) Wait() {
	if !w.Closed() {
		w.Cond.Wait()
	}
}

func (w *writer) Close() error {
	atomic.StoreUint32(&w.closed, 1)
	w.Cond.Broadcast()
	return nil
}

func (w *writer) Closed() bool {
	return atomic.LoadUint32(&w.closed) != 0
}
