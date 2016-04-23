package stream

import (
	"bytes"
	"io"
	"sync/atomic"
)

type reader struct {
	w      *writer
	off    int
	closed uint32
}

// Read reads from the Buffer
func (r *reader) Read(p []byte) (n int, err error) {
	r.w.RLock()
	defer r.w.RUnlock()

	var m int

	for len(p) > 0 {

		m, _ = bytes.NewReader(r.w.buffer.Bytes()[r.off:]).Read(p)
		n += m
		r.off += n

		if n > 0 {
			break
		}

		if r.w.Closed() {
			err = io.EOF
			break
		}
		if r.Closed() {
			err = io.EOF
			break
		}

		r.w.Wait()
	}

	return
}

func (r *reader) Close() error {
	atomic.StoreUint32(&r.closed, 1)
	return nil
}

func (r *reader) Closed() bool {
	return atomic.LoadUint32(&r.closed) != 0
}
