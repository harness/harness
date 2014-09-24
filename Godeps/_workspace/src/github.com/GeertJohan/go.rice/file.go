package rice

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// File abstracts file methods so the user doesn't see the difference between rice.virtualFile, rice.virtualDir and os.File
// This type implements the io.Reader, io.Seeker, io.Closer and http.File interfaces
type File struct {
	realF *os.File

	// when embedded (go)
	virtualF *virtualFile
	virtualD *virtualDir

	// when appended (zip)
	appendedF  *appendedFile
	appendedRC io.ReadCloser
}

// Close is like (*os.File).Close()
// Visit http://golang.org/pkg/os/#File.Close for more information
func (f *File) Close() error {
	if f.appendedF != nil {
		return f.appendedRC.Close()
	}
	if f.virtualF != nil {
		return f.virtualF.close()
	}
	if f.virtualD != nil {
		return f.virtualD.close()
	}
	return f.realF.Close()
}

// Stat is like (*os.File).Stat()
// Visit http://golang.org/pkg/os/#File.Stat for more information
func (f *File) Stat() (os.FileInfo, error) {
	if f.appendedF != nil {
		if f.appendedF.dir {
			return f.appendedF.dirInfo, nil
		}
		return f.appendedF.zipFile.FileInfo(), nil
	}
	if f.virtualF != nil {
		return f.virtualF.stat()
	}
	if f.virtualD != nil {
		return f.virtualD.stat()
	}
	return f.realF.Stat()
}

// Readdir is like (*os.File).Readdir()
// Visit http://golang.org/pkg/os/#File.Readdir for more information
func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	if f.appendedF != nil {
		if f.appendedF.dir {
			fi := make([]os.FileInfo, 0, len(f.appendedF.children))
			for _, childAppendedFile := range f.appendedF.children {
				if childAppendedFile.dir {
					fi = append(fi, childAppendedFile.dirInfo)
				} else {
					fi = append(fi, childAppendedFile.zipFile.FileInfo())
				}
			}
			return fi, nil
		}
		//++ TODO: is os.ErrInvalid the correct error for Readdir on file?
		return nil, os.ErrInvalid
	}
	if f.virtualF != nil {
		return f.virtualF.readdir(count)
	}
	if f.virtualD != nil {
		return f.virtualD.readdir(count)
	}
	return f.realF.Readdir(count)
}

// Read is like (*os.File).Read()
// Visit http://golang.org/pkg/os/#File.Read for more information
func (f *File) Read(bts []byte) (int, error) {
	if f.appendedF != nil {
		if f.appendedF.dir {
			return 0, &os.PathError{
				Op:   "read",
				Path: filepath.Base(f.appendedF.zipFile.Name),
				Err:  errors.New("is a directory"),
			}
		}
		return f.appendedRC.Read(bts)
	}
	if f.virtualF != nil {
		return f.virtualF.read(bts)
	}
	if f.virtualD != nil {
		return f.virtualD.read(bts)
	}
	return f.realF.Read(bts)
}

// Seek is like (*os.File).Seek()
// Visit http://golang.org/pkg/os/#File.Seek for more information
func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f.appendedF != nil {
		return 0, ErrNotImplemented
	}
	if f.virtualF != nil {
		return f.virtualF.seek(offset, whence)
	}
	if f.virtualD != nil {
		return f.virtualD.seek(offset, whence)
	}
	return f.realF.Seek(offset, whence)
}
