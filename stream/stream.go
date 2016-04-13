package stream

//go:generate mockery -name Mux -output mock -case=underscore

import (
	"bufio"
	"io"
	"strconv"

	"golang.org/x/net/context"
)

// Mux defines a stream multiplexer
type Mux interface {
	// Create creates and returns a new stream identified by
	// the specified key.
	Create(key string) (io.ReadCloser, io.WriteCloser, error)

	// Open returns the existing stream by key. If the stream
	// does not exist an error is returned.
	Open(key string) (io.ReadCloser, io.WriteCloser, error)

	// Remove deletes the stream by key.
	Remove(key string) error

	// Exists return true if the stream exists.
	Exists(key string) bool
}

// Create creates and returns a new stream identified
// by the specified key.
func Create(c context.Context, key string) (io.ReadCloser, io.WriteCloser, error) {
	return FromContext(c).Create(key)
}

// Open returns the existing stream by key. If the stream does
// not exist an error is returned.
func Open(c context.Context, key string) (io.ReadCloser, io.WriteCloser, error) {
	return FromContext(c).Open(key)
}

// Exists return true if the stream exists.
func Exists(c context.Context, key string) bool {
	return FromContext(c).Exists(key)
}

// Remove deletes the stream by key.
func Remove(c context.Context, key string) error {
	return FromContext(c).Remove(key)
}

// ToKey is a helper function that converts a unique identifier
// of type int64 into a string.
func ToKey(i int64) string {
	return strconv.FormatInt(i, 10)
}

// Copy copies the stream from the source to the destination in
// valid JSON format. This converts the logs, which are per-line
// JSON objects, to a JSON array.
func Copy(dest io.Writer, src io.Reader) error {
	io.WriteString(dest, "[")

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		io.WriteString(dest, scanner.Text())
		io.WriteString(dest, ",\n")
	}

	io.WriteString(dest, "{}]")

	return nil
}
