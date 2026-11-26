//  Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"encoding/hex"
	"encoding/xml"
	"io"
	"math"
	"strings"
	"testing"

	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"

	"github.com/stretchr/testify/require"
)

func TestW2rNewFileBackedBufferValidatesSize(t *testing.T) {
	for _, size := range []int{-1, math.MaxInt32 + 1} {
		_, err := NewFileBackedBuffer(size)
		require.ErrorIs(t, err, ErrInvalidMemorySize)
	}
	b, err := NewFileBackedBuffer(0)
	require.NoError(t, err)
	require.Zero(t, b.Size())
}

func TestW2rFileBackedBufferInMemory(t *testing.T) {
	b, err := NewFileBackedBuffer(64)
	require.NoError(t, err)

	n, err := b.Write([]byte("hello "))
	require.NoError(t, err)
	require.Equal(t, 6, n)
	_, err = b.Write([]byte("world"))
	require.NoError(t, err)
	require.Equal(t, int64(11), b.Size())

	data, err := io.ReadAll(b)
	require.NoError(t, err)
	require.Equal(t, "hello world", string(data))

	pos, err := b.Seek(6, io.SeekStart)
	require.NoError(t, err)
	require.Equal(t, int64(6), pos)

	buf := make([]byte, 5)
	_, err = b.ReadAt(buf, 6)
	require.NoError(t, err)
	require.Equal(t, "world", string(buf))

	_, err = b.Write([]byte("x"))
	require.ErrorIs(t, err, ErrWriteAfterRead)
	require.NoError(t, b.Close())
}

func TestW2rFileBackedBufferSpillsToFile(t *testing.T) {
	b, err := NewFileBackedBuffer(4)
	require.NoError(t, err)

	_, err = b.Write([]byte("abc"))
	require.NoError(t, err)
	_, err = b.Write([]byte("defgh")) // exceeds the memory limit and moves to a temp file.
	require.NoError(t, err)
	_, err = b.Write([]byte("ij")) // appended directly to the file.
	require.NoError(t, err)
	require.Equal(t, int64(10), b.Size())

	data, err := io.ReadAll(b)
	require.NoError(t, err)
	require.Equal(t, "abcdefghij", string(data))

	buf := make([]byte, 3)
	_, err = b.ReadAt(buf, 2)
	require.NoError(t, err)
	require.Equal(t, "cde", string(buf))

	_, err = b.Seek(0, io.SeekStart)
	require.NoError(t, err)
	require.NoError(t, b.Close())
	// Closing twice is safe.
	require.NoError(t, b.Close())
}

func TestW2rPrimaryPackageMarshalsWithRPMPrefix(t *testing.T) {
	pkg := PrimaryPackage{
		Type:         "rpm",
		Name:         "demo",
		Architecture: "x86_64",
		Version:      PrimaryVersion{Epoch: "0", Version: "1.0", Release: "1"},
		Format: PrimaryFormat{
			License:  "MIT",
			Provides: PrimaryEntryList{Entries: []*rpmmetadata.Entry{{Name: "demo", Version: "1.0"}}},
			Requires: PrimaryEntryList{Entries: []*rpmmetadata.Entry{{Name: "libc"}}},
			Files:    []*rpmmetadata.File{{Path: "/usr/bin/demo"}},
		},
	}

	out, err := xml.Marshal(pkg)
	require.NoError(t, err)
	s := string(out)

	require.Contains(t, s, "<name>demo</name>")
	require.Contains(t, s, "<rpm:license>MIT</rpm:license>")
	require.Contains(t, s, `<rpm:entry name="demo" ver="1.0"></rpm:entry>`)
	require.Contains(t, s, `<rpm:entry name="libc"></rpm:entry>`)
	require.Contains(t, s, "<file>/usr/bin/demo</file>")
}

func TestW2rHashedBufferComputesSums(t *testing.T) {
	b, err := CreateHashedBufferFromReader(strings.NewReader("abc"))
	require.NoError(t, err)
	defer b.Close()

	md5Sum, sha1Sum, sha256Sum, sha512Sum := b.Sums()
	require.Equal(t, "900150983cd24fb0d6963f7d28e17f72", hex.EncodeToString(md5Sum))
	require.Equal(t, "a9993e364706816aba3e25717850c26c9cd0d89d", hex.EncodeToString(sha1Sum))
	require.Equal(t, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", hex.EncodeToString(sha256Sum))
	require.Len(t, sha512Sum, 64)
	require.Equal(t, int64(3), b.Size())
}

func TestW2rHashedBufferWithSizeSpillsAndValidates(t *testing.T) {
	_, err := NewHashedBufferWithSize(-1)
	require.ErrorIs(t, err, ErrInvalidMemorySize)
	_, err = CreateHashedBufferFromReaderWithSize(strings.NewReader("x"), -1)
	require.ErrorIs(t, err, ErrInvalidMemorySize)

	b, err := CreateHashedBufferFromReaderWithSize(strings.NewReader("0123456789"), 2)
	require.NoError(t, err)
	defer b.Close()
	require.Equal(t, int64(10), b.Size())
}

func TestW2rMultiHasherMarshalBinary(t *testing.T) {
	h := NewMultiHasher()
	_, err := h.Write([]byte("hello"))
	require.NoError(t, err)

	state, err := h.MarshalBinary()
	require.NoError(t, err)
	require.NotEmpty(t, state)

	md5Sum, sha1Sum, sha256Sum, sha512Sum := h.Sums()
	require.Len(t, md5Sum, 16)
	require.Len(t, sha1Sum, 20)
	require.Len(t, sha256Sum, 32)
	require.Len(t, sha512Sum, 64)
}

type w2rFailingReader struct{}

func (w2rFailingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestW2rHashedBufferFromFailingReader(t *testing.T) {
	_, err := CreateHashedBufferFromReader(w2rFailingReader{})
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}
