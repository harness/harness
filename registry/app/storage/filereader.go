// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/harness/gitness/registry/app/driver"
)

const fileReaderBufferSize = 4 * 1024 * 1024

// remoteFileReader provides a read seeker interface to files stored in
// storagedriver. Used to implement part of layer interface and will be used
// to implement read side of LayerUpload.
type FileReader struct {
	driver driver.StorageDriver

	ctx context.Context

	// identifying fields
	path string
	size int64 // size is the total size, must be set.

	// mutable fields
	rc     io.ReadCloser // remote read closer
	brd    *bufio.Reader // internal buffered io
	offset int64         // offset is the current read offset
	err    error         // terminal error, if set, reader is closed
}

// NewFileReader initializes a file reader for the remote file. The reader
// takes on the size and path that must be determined externally with a stat
// call. The reader operates optimistically, assuming that the file is already
// there.
func NewFileReader(ctx context.Context, driver driver.StorageDriver, path string, size int64) (*FileReader, error) {
	return &FileReader{
		ctx:    ctx,
		driver: driver,
		path:   path,
		size:   size,
	}, nil
}

func (fr *FileReader) Read(p []byte) (n int, err error) {
	if fr.err != nil {
		return 0, fr.err
	}

	rd, err := fr.reader()
	if err != nil {
		return 0, err
	}

	n, err = rd.Read(p)
	fr.offset += int64(n)

	// Simulate io.EOR error if we reach filesize.
	if err == nil && fr.offset >= fr.size {
		err = io.EOF
	}

	return n, err
}

func (fr *FileReader) Seek(offset int64, whence int) (int64, error) {
	if fr.err != nil {
		return 0, fr.err
	}

	var err error
	newOffset := fr.offset

	switch whence {
	case io.SeekCurrent:
		newOffset += offset
	case io.SeekEnd:
		newOffset = fr.size + offset
	case io.SeekStart:
		newOffset = offset
	}

	if newOffset < 0 {
		err = fmt.Errorf("cannot seek to negative position")
	} else {
		if fr.offset != newOffset {
			fr.reset()
		}

		// No problems, set the offset.
		fr.offset = newOffset
	}

	return fr.offset, err
}

func (fr *FileReader) Close() error {
	return fr.closeWithErr(fmt.Errorf("FileReader: closed"))
}

// reader prepares the current reader at the lrs offset, ensuring its buffered
// and ready to go.
func (fr *FileReader) reader() (io.Reader, error) {
	if fr.err != nil {
		return nil, fr.err
	}

	if fr.rc != nil {
		return fr.brd, nil
	}

	// If we don't have a reader, open one up.
	rc, err := fr.driver.Reader(fr.ctx, fr.path, fr.offset)
	if err != nil {
		if errors.As(err, &driver.PathNotFoundError{}) {
			return io.NopCloser(bytes.NewReader([]byte{})), nil
		}
		return nil, err
	}

	fr.rc = rc

	if fr.brd == nil {
		fr.brd = bufio.NewReaderSize(fr.rc, fileReaderBufferSize)
	} else {
		fr.brd.Reset(fr.rc)
	}

	return fr.brd, nil
}

// resetReader resets the reader, forcing the read method to open up a new
// connection and rebuild the buffered reader. This should be called when the
// offset and the reader will become out of sync, such as during a seek
// operation.
func (fr *FileReader) reset() {
	if fr.err != nil {
		return
	}
	if fr.rc != nil {
		fr.rc.Close()
		fr.rc = nil
	}
}

func (fr *FileReader) closeWithErr(err error) error {
	if fr.err != nil {
		return fr.err
	}

	fr.err = err

	// close and release reader chain
	if fr.rc != nil {
		fr.rc.Close()
	}

	fr.rc = nil
	fr.brd = nil

	return fr.err
}
