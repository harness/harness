package fscache

import (
	"errors"
	"io"
	"sync"
)

type layeredCache struct {
	layers []Cache
}

// NewLayered returns a Cache which stores its data in all the passed
// caches, when a key is requested it is loaded into all the caches above the first hit.
func NewLayered(caches ...Cache) Cache {
	return &layeredCache{layers: caches}
}

func (l *layeredCache) Get(key string) (r ReadAtCloser, w io.WriteCloser, err error) {
	var last ReadAtCloser
	var writers []io.WriteCloser

	for i, layer := range l.layers {
		r, w, err = layer.Get(key)
		if err != nil {
			if len(writers) > 0 {
				last.Close()
				multiWC(writers...).Close()
			}
			return nil, nil, err
		}

		// hit
		if w == nil {
			if len(writers) > 0 {
				go func(r io.ReadCloser) {
					wc := multiWC(writers...)
					defer r.Close()
					defer wc.Close()
					io.Copy(wc, r)
				}(r)
				return last, nil, nil
			}
			return r, nil, nil
		}

		// miss
		writers = append(writers, w)

		if i == len(l.layers)-1 {
			if last != nil {
				last.Close()
			}
			return r, multiWC(writers...), nil
		}

		if last != nil {
			last.Close()
		}
		last = r
	}

	return nil, nil, errors.New("no caches")
}

func (l *layeredCache) Remove(key string) error {
	var grp sync.WaitGroup
	// walk upwards so that lower layers don't
	// restore upper layers on Get()
	for i := len(l.layers) - 1; i >= 0; i-- {
		grp.Add(1)
		go func(layer Cache) {
			defer grp.Done()
			layer.Remove(key)
		}(l.layers[i])
	}
	grp.Wait()
	return nil
}

func (l *layeredCache) Exists(key string) bool {
	for _, layer := range l.layers {
		if layer.Exists(key) {
			return true
		}
	}
	return false
}

func (l *layeredCache) Clean() (err error) {
	for _, layer := range l.layers {
		er := layer.Clean()
		if er != nil {
			err = er
		}
	}
	return nil
}

func multiWC(wc ...io.WriteCloser) io.WriteCloser {
	if len(wc) == 0 {
		return nil
	}

	return &multiWriteCloser{
		writers: wc,
	}
}

type multiWriteCloser struct {
	writers []io.WriteCloser
}

func (t *multiWriteCloser) Write(p []byte) (n int, err error) {
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
	}
	return len(p), nil
}

func (t *multiWriteCloser) Close() error {
	for _, w := range t.writers {
		w.Close()
	}
	return nil
}
