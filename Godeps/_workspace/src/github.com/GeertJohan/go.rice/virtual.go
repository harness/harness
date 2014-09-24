package rice

import (
	"errors"
	"github.com/GeertJohan/go.rice/embedded"
	"os"
	"syscall"
)

//++ TODO: IDEA: merge virtualFile and virtualDir, this decreases work done by rice.File

// Error indicating some function is not implemented yet (but available to satisfy an interface)
var ErrNotImplemented = errors.New("not implemented yet")

// virtualFile is a 'stateful' virtual file.
// virtualFile wraps an *EmbeddedFile for a call to Box.Open() and virtualizes 'read cursor' (offset) and 'closing'.
// virtualFile is only internally visible and should be exposed through rice.File
type virtualFile struct {
	*embedded.EmbeddedFile       // the actual embedded file, embedded to obtain methods
	offset                 int64 // read position on the virtual file
	closed                 bool  // closed when true
}

// create a new virtualFile for given EmbeddedFile
func newVirtualFile(ef *embedded.EmbeddedFile) *virtualFile {
	vf := &virtualFile{
		EmbeddedFile: ef,
		offset:       0,
		closed:       false,
	}
	return vf
}

//++ TODO check for nil pointers in all these methods. When so: return os.PathError with Err: os.ErrInvalid

func (vf *virtualFile) close() error {
	if vf.closed {
		return &os.PathError{
			Op:   "close",
			Path: vf.EmbeddedFile.Filename,
			Err:  errors.New("already closed"),
		}
	}
	vf.EmbeddedFile = nil
	vf.closed = true
	return nil
}

func (vf *virtualFile) stat() (os.FileInfo, error) {
	if vf.closed {
		return nil, &os.PathError{
			Op:   "stat",
			Path: vf.EmbeddedFile.Filename,
			Err:  errors.New("bad file descriptor"),
		}
	}
	return (*embeddedFileInfo)(vf.EmbeddedFile), nil
}

func (vf *virtualFile) readdir(count int) ([]os.FileInfo, error) {
	if vf.closed {
		return nil, &os.PathError{
			Op:   "readdir",
			Path: vf.EmbeddedFile.Filename,
			Err:  errors.New("bad file descriptor"),
		}
	}
	//TODO: return proper error for a readdir() call on a file
	return nil, ErrNotImplemented
}

func (vf *virtualFile) read(bts []byte) (int, error) {
	if vf.closed {
		return 0, &os.PathError{
			Op:   "read",
			Path: vf.EmbeddedFile.Filename,
			Err:  errors.New("bad file descriptor"),
		}
	}
	end := vf.offset + int64(len(bts))
	n := copy(bts, vf.Content[vf.offset:end])
	vf.offset += int64(n)
	return n, nil
}

func (vf *virtualFile) seek(offset int64, whence int) (int64, error) {
	if vf.closed {
		return 0, &os.PathError{
			Op:   "seek",
			Path: vf.EmbeddedFile.Filename,
			Err:  errors.New("bad file descriptor"),
		}
	}
	var e error

	//++ TODO: check if this is correct implementation for seek
	switch whence {
	case os.SEEK_SET:
		//++ check if new offset isn't out of bounds, set e when it is, then break out of switch
		vf.offset = offset
	case os.SEEK_CUR:
		//++ check if new offset isn't out of bounds, set e when it is, then break out of switch
		vf.offset += offset
	case os.SEEK_END:
		//++ check if new offset isn't out of bounds, set e when it is, then break out of switch
		vf.offset = int64(len(vf.EmbeddedFile.Content)) - offset
	}

	if e != nil {
		return 0, &os.PathError{
			Op:   "seek",
			Path: vf.Filename,
			Err:  e,
		}
	}

	return vf.offset, nil
}

// vritualDir is a 'stateful' virtual directory.
// vritualDir wraps an *EmbeddedDir for a call to Box.Open() and virtualizes 'closing'.
// vritualDir is only internally visible and should be exposed through rice.File
type virtualDir struct {
	*embedded.EmbeddedDir
	closed bool
}

// create a new virtualDir for given EmbeddedDir
func newVirtualDir(ed *embedded.EmbeddedDir) *virtualDir {
	vd := &virtualDir{
		EmbeddedDir: ed,
		closed:      false,
	}
	return vd
}

func (vd *virtualDir) close() error {
	//++ TODO: needs sync mutex?
	if vd.closed {
		return &os.PathError{
			Op:   "close",
			Path: vd.EmbeddedDir.Filename,
			Err:  errors.New("already closed"),
		}
	}
	vd.closed = true
	return nil
}

func (vd *virtualDir) stat() (os.FileInfo, error) {
	if vd.closed {
		return nil, &os.PathError{
			Op:   "stat",
			Path: vd.EmbeddedDir.Filename,
			Err:  errors.New("bad file descriptor"),
		}
	}
	return (*embeddedDirInfo)(vd.EmbeddedDir), nil
}

func (vd *virtualDir) readdir(count int) ([]os.FileInfo, error) {
	if vd.closed {
		return nil, &os.PathError{
			Op:   "readdir",
			Path: vd.EmbeddedDir.Filename,
			Err:  errors.New("bad file descriptor"),
		}
	}
	//++ TODO: what should happen on closed dir? return an error here?
	//++ read ChildDirs and ChildFiles from vd.EmbeddedDir
	//++ keep track of n in virtualDir field to remember what the the last pos was
	return nil, ErrNotImplemented
}

func (vd *virtualDir) read(bts []byte) (int, error) {
	if vd.closed {
		return 0, &os.PathError{
			Op:   "read",
			Path: vd.EmbeddedDir.Filename,
			Err:  errors.New("bad file descriptor"),
		}
	}
	return 0, &os.PathError{
		Op:   "read",
		Path: vd.EmbeddedDir.Filename,
		Err:  errors.New("is a directory"),
	}
}

func (vd *virtualDir) seek(offset int64, whence int) (int64, error) {
	if vd.closed {
		return 0, &os.PathError{
			Op:   "seek",
			Path: vd.EmbeddedDir.Filename,
			Err:  errors.New("bad file descriptor"),
		}
	}
	return 0, &os.PathError{
		Op:   "seek",
		Path: vd.Filename,
		Err:  syscall.EISDIR,
	}
}
