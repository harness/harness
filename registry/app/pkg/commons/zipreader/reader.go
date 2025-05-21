// Major portions copied from the GO std lib, copyright below:

// Copyright (c) 2012 The Go Authors. All rights reserved.

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:

// * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
// * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.

// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Code enabling ZipStreaming, copyright below:

// Copyright (c) 2015 Richard Warburton. All rights reserved.

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:

// * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
// * The name Richard Warburton may not be used to endorse or promote
// products derived from this software without specific prior written
// permission.

// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package zipreader

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"
)

const (
	readAhead  = 28
	maxRead    = 4096
	bufferSize = maxRead + readAhead
)

// A Reader provides sequential access to the contents of a zip archive.
// A zip archive consists of a sequence of files,
// The Next method advances to the next file in the archive (including the first),
// and then it can be treated as an io.Reader to access the file's data.
// The Buffered method recovers any bytes read beyond the end of the zip file,
// necessary if you plan to process anything after it that is not another zip file.
type Reader struct {
	io.Reader
	br *bufio.Reader
}

// NewReader creates a new Reader reading from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{br: bufio.NewReaderSize(r, bufferSize)}
}

// Next advances to the next entry in the zip archive.
//
// io.EOF is returned when the end of the zip file has been reached.
// If Next is called again, it will presume another zip file immediately follows
// and it will advance into it.
func (r *Reader) Next() (*zip.FileHeader, error) {
	if r.Reader != nil {
		if _, err := io.Copy(io.Discard, r.Reader); err != nil {
			return nil, err
		}
	}

	for {
		sigData, err := r.br.Peek(4096)
		if err != nil {
			if err == io.EOF && len(sigData) < 46+22 { // Min length of Central directory + End of central directory
				return nil, err
			}
		}

		switch sig := binary.LittleEndian.Uint32(sigData); sig {
		case fileHeaderSignature:
			break
		case directoryHeaderSignature: // Directory appears at end of file so we are finished
			return nil, discardCentralDirectory(r.br)
		default:
			index := bytes.Index(sigData[1:], sigBytes)
			if index == -1 {
				//nolint: errcheck
				r.br.Discard(len(sigData) - len(sigBytes) + 1)
				continue
			}
			//nolint: errcheck
			r.br.Discard(index + 1)
		}
		break
	}

	headBuf := make([]byte, fileHeaderLen)
	if _, err := io.ReadFull(r.br, headBuf); err != nil {
		return nil, err
	}
	b := readBuf(headBuf[4:])

	f := &zip.FileHeader{
		ReaderVersion:    b.uint16(),
		Flags:            b.uint16(),
		Method:           b.uint16(),
		ModifiedTime:     b.uint16(),
		ModifiedDate:     b.uint16(),
		CRC32:            b.uint32(),
		CompressedSize:   b.uint32(), // TODO handle zip64
		UncompressedSize: b.uint32(), // TODO handle zip64
	}

	filenameLen := b.uint16()
	extraLen := b.uint16()

	d := make([]byte, filenameLen+extraLen)
	if _, err := io.ReadFull(r.br, d); err != nil {
		return nil, err
	}
	f.Name = string(d[:filenameLen])
	f.Extra = d[filenameLen : filenameLen+extraLen]

	dcomp := decompressor(f.Method)
	if dcomp == nil {
		return nil, zip.ErrAlgorithm
	}

	// TODO handle encryption here
	crc := &crcReader{
		hash: crc32.NewIEEE(),
		crc:  &f.CRC32,
	}
	if f.Flags&0x8 != 0 { // If has dataDescriptor
		crc.Reader = dcomp(&descriptorReader{br: r.br, fileHeader: f})
	} else {
		//nolint: gosec
		crc.Reader = dcomp(io.LimitReader(r.br, int64(f.CompressedSize)))
		crc.crc = &f.CRC32
	}
	r.Reader = crc
	return f, nil
}

// Buffered returns any bytes beyond the end of the zip file that it may have
// read. These are necessary if you plan to process anything after it,
// that isn't another zip file.
func (r *Reader) Buffered() io.Reader { return r.br }
