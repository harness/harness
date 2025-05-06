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
	"encoding/binary"
	"errors"
	"io"
)

// We're not interested in the central directory's data, we just want to skip over it,
// clearing the stream of the current zip, in case anything else needs to be sent over
// the same stream.
func discardCentralDirectory(br *bufio.Reader) error {
	for {
		sigBytes, err := br.Peek(4)
		if err != nil {
			return err
		}
		switch sig := binary.LittleEndian.Uint32(sigBytes); sig {
		case directoryHeaderSignature:
			if err := discardDirectoryHeaderRecord(br); err != nil {
				return err
			}
		case directoryEndSignature:
			if err := discardDirectoryEndRecord(br); err != nil {
				return err
			}
			return io.EOF
		case directory64EndSignature:
			return errors.New("Zip64 not yet supported")
		case directory64LocSignature: // Not sure what this is yet
			return errors.New("Zip64 not yet supported")
		default:
			return zip.ErrFormat
		}
	}
}

func discardDirectoryHeaderRecord(br *bufio.Reader) error {
	if _, err := br.Discard(28); err != nil {
		return err
	}
	lb, err := br.Peek(6)
	if err != nil {
		return err
	}
	lengths := int(binary.LittleEndian.Uint16(lb[:2])) + // File name length
		int(binary.LittleEndian.Uint16(lb[2:4])) + // Extra field length
		int(binary.LittleEndian.Uint16(lb[4:])) // File comment length
	_, err = br.Discard(18 + lengths)
	return err
}

func discardDirectoryEndRecord(br *bufio.Reader) error {
	if _, err := br.Discard(20); err != nil {
		return err
	}
	commentLength, err := br.Peek(2)
	if err != nil {
		return err
	}
	_, err = br.Discard(2 + int(binary.LittleEndian.Uint16(commentLength)))
	return err
}
