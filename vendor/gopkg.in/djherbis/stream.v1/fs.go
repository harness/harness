package stream

import (
	"io"
	"os"
)

// File is a backing data-source for a Stream.
type File interface {
	Name() string // The name used to Create/Open the File
	io.Reader     // Reader must continue reading after EOF on subsequent calls after more Writes.
	io.ReaderAt   // Similarly to Reader
	io.Writer     // Concurrent reading/writing must be supported.
	io.Closer     // Close should do any cleanup when done with the File.
}

// FileSystem is used to manage Files
type FileSystem interface {
	Create(name string) (File, error) // Create must return a new File for Writing
	Open(name string) (File, error)   // Open must return an existing File for Reading
	Remove(name string) error         // Remove deletes an existing File
}

// StdFileSystem is backed by the os package.
var StdFileSystem FileSystem = stdFS{}

type stdFS struct{}

func (fs stdFS) Create(name string) (File, error) {
	return os.Create(name)
}

func (fs stdFS) Open(name string) (File, error) {
	return os.Open(name)
}

func (fs stdFS) Remove(name string) error {
	return os.Remove(name)
}
