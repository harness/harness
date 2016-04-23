package stream

import (
	"bytes"
	"io"
)

type reader struct {
	w   *writer
	off int
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

		r.w.Wait()
	}

	return
}

func (r *reader) Close() error {
	// TODO close should remove reader from the parent!
	return nil
}
