package blobstore

import (
	"io"

	"code.google.com/p/go.net/context"
)

type Blobstore interface {
	// Del removes an object from the blobstore.
	Del(path string) error

	// Get retrieves an object from the blobstore.
	Get(path string) ([]byte, error)

	// GetReader retrieves an object from the blobstore.
	// It is the caller's responsibility to call Close on
	// the ReadCloser when finished reading.
	GetReader(path string) (io.ReadCloser, error)

	// Put inserts an object into the blobstore.
	Put(path string, data []byte) error

	// PutReader inserts an object into the blobstore by
	// consuming data from r until EOF.
	PutReader(path string, r io.Reader) error
}

// Del removes an object from the blobstore.
func Del(c context.Context, path string) error {
	return FromContext(c).Del(path)
}

// Get retrieves an object from the blobstore.
func Get(c context.Context, path string) ([]byte, error) {
	return FromContext(c).Get(path)
}

// GetReader retrieves an object from the blobstore.
// It is the caller's responsibility to call Close on
// the ReadCloser when finished reading.
func GetReader(c context.Context, path string) (io.ReadCloser, error) {
	return FromContext(c).GetReader(path)
}

// Put inserts an object into the blobstore.
func Put(c context.Context, path string, data []byte) error {
	return FromContext(c).Put(path, data)
}

// PutReader inserts an object into the blobstore by
// consuming data from r until EOF.
func PutReader(c context.Context, path string, r io.Reader) error {
	return FromContext(c).PutReader(path, r)
}
