package stream

import (
	"bufio"
	"io"
	"strconv"

	"golang.org/x/net/context"
)

// Stream manages the stream of build logs.
type Stream interface {
	Create(string) error
	Delete(string) error
	Reader(string) (io.ReadCloser, error)
	Writer(string) (io.WriteCloser, error)
}

// Create creates a new stream.
func Create(c context.Context, key string) error {
	return FromContext(c).Create(key)
}

// Reader opens the stream for reading.
func Reader(c context.Context, key string) (io.ReadCloser, error) {
	return FromContext(c).Reader(key)
}

// Writer opens the stream for writing.
func Writer(c context.Context, key string) (io.WriteCloser, error) {
	return FromContext(c).Writer(key)
}

// Delete deletes the stream by key.
func Delete(c context.Context, key string) error {
	return FromContext(c).Delete(key)
}

// ToKey is a helper function that converts a unique identifier
// of type int64 into a string.
func ToKey(i int64) string {
	return strconv.FormatInt(i, 10)
}

// Copy copies the stream from the source to the destination in valid JSON
// format. This converts the logs, which are per-line JSON objects, to a
// proper JSON array.
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
