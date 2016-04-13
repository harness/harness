package stream

import "io"

// Reader is a concurrent-safe Stream Reader.
type Reader struct {
	s    *Stream
	file File
}

// Name returns the name of the underlying File in the FileSystem.
func (r *Reader) Name() string { return r.file.Name() }

// ReadAt lets you Read from specific offsets in the Stream.
// ReadAt blocks while waiting for the requested section of the Stream to be written,
// unless the Stream is closed in which case it will always return immediately.
func (r *Reader) ReadAt(p []byte, off int64) (n int, err error) {
	r.s.b.RLock()
	defer r.s.b.RUnlock()

	var m int

	for {

		m, err = r.file.ReadAt(p[n:], off+int64(n))
		n += m

		if r.s.b.IsOpen() {

			switch {
			case n != 0 && err == nil:
				return n, err
			case err == io.EOF:
				r.s.b.Wait()
			case err != nil:
				return n, err
			}

		} else {
			return n, err
		}

	}
}

// Read reads from the Stream. If the end of an open Stream is reached, Read
// blocks until more data is written or the Stream is Closed.
func (r *Reader) Read(p []byte) (n int, err error) {
	r.s.b.RLock()
	defer r.s.b.RUnlock()

	var m int

	for {

		m, err = r.file.Read(p[n:])
		n += m

		if r.s.b.IsOpen() {

			switch {
			case n != 0 && err == nil:
				return n, err
			case err == io.EOF:
				r.s.b.Wait()
			case err != nil:
				return n, err
			}

		} else {
			return n, err
		}

	}
}

// Close closes this Reader on the Stream. This must be called when done with the
// Reader or else the Stream cannot be Removed.
func (r *Reader) Close() error {
	defer r.s.dec()
	return r.file.Close()
}
