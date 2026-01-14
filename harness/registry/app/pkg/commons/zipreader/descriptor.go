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
	"io"
)

type descriptorReader struct {
	br         *bufio.Reader
	size       uint64
	eof        bool
	fileHeader *zip.FileHeader
}

var (
	sigBytes = []byte{0x50, 0x4b}
)

func (r *descriptorReader) Read(p []byte) (n int, err error) {
	if r.eof {
		return 0, io.EOF
	}

	if n = len(p); n > maxRead {
		n = maxRead
	}

	z, err := r.br.Peek(n + readAhead)
	if err != nil {
		if err == io.EOF && len(z) < 46+22 { // Min length of Central directory + End of central directory
			return 0, err
		}
		n = len(z)
	}

	// Look for header of next file or central directory
	discard := n
	s := 16
	for !r.eof && s < n {
		i := bytes.Index(z[s:len(z)-4], sigBytes) + s
		if i == -1 {
			break
		}

		// If directoryHeaderSignature or fileHeaderSignature file could be finished
		//nolint: nestif
		if sig := binary.LittleEndian.Uint32(z[i : i+4]); sig == fileHeaderSignature ||
			sig == directoryHeaderSignature {
			// Now check for compressed file sizes to ensure not false positive and if zip64.

			if i < len(z)-8 { // Zip32
				// Zip32 optional dataDescriptorSignature
				offset := 0
				if binary.LittleEndian.Uint32(z[i-16:i-12]) == dataDescriptorSignature {
					offset = 4
				}

				// Zip32 compressed file size
				//nolint: gosec
				if binary.LittleEndian.Uint32(z[i-8:i-4]) == uint32(r.size)+uint32(i-12-offset) {
					n, discard = i-12-offset, i
					r.eof = true
					r.fileHeader.CRC32 = binary.LittleEndian.Uint32(z[i-12 : i-8])
					break
				}
			}

			if i > 24 {
				// Zip64 optional dataDescriptorSignature
				offset := 0
				if binary.LittleEndian.Uint32(z[i-24:i-20]) == dataDescriptorSignature {
					offset = 4
				}

				// Zip64 compressed file size
				//nolint: gosec
				if i >= 8 && binary.LittleEndian.Uint64(z[i-16:i-8]) == r.size+uint64(i-20-offset) {
					n, discard = i-20-offset, i
					r.eof = true
					break
				}
			}
		}

		s = i + 2
	}
	copy(p, z[:n])
	//nolint: errcheck
	r.br.Discard(discard)
	//nolint: gosec
	r.size += uint64(n)
	return n, err
}
