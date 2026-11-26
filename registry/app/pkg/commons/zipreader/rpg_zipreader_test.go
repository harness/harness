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

package zipreader

import (
	"archive/zip"
	"bytes"
	"hash/crc32"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

const rpgCustomMethod uint16 = 0x7777

var rpgRegisterOnce sync.Once

func rpgZip(t *testing.T, entries map[string]string, order []string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, name := range order {
		method := zip.Deflate
		if strings.HasPrefix(name, "stored") {
			method = zip.Store
		}
		f, err := w.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		require.NoError(t, err)
		_, err = f.Write([]byte(entries[name]))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// rpgRawZip writes entries with known sizes and no data descriptor.
func rpgRawZip(t *testing.T, name string, method uint16, data string, crc uint32) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.CreateRaw(&zip.FileHeader{
		Name: name, Method: method, CRC32: crc,
		CompressedSize64: uint64(len(data)), UncompressedSize64: uint64(len(data)),
	})
	require.NoError(t, err)
	_, err = f.Write([]byte(data))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestRpgReaderStreamsEntriesInOrder(t *testing.T) {
	long := strings.Repeat("compressible content ", 2000)
	entries := map[string]string{"a.txt": "alpha", "stored.txt": "stored bytes", "long.txt": long, "empty.txt": ""}
	order := []string{"a.txt", "stored.txt", "long.txt", "empty.txt"}

	r := NewReader(bytes.NewReader(rpgZip(t, entries, order)))
	for _, name := range order {
		header, err := r.Next()
		require.NoError(t, err, name)
		require.Equal(t, name, header.Name)
		body, err := io.ReadAll(r)
		require.NoError(t, err, name)
		require.Equal(t, entries[name], string(body), name)
	}

	_, err := r.Next()
	require.Error(t, err)
	require.NotNil(t, r.Buffered())
}

func TestRpgReaderSkipsUnreadEntries(t *testing.T) {
	entries := map[string]string{"first": "one", "second": "two"}
	r := NewReader(bytes.NewReader(rpgZip(t, entries, []string{"first", "second"})))

	header, err := r.Next()
	require.NoError(t, err)
	require.Equal(t, "first", header.Name)

	// The first entry is discarded when it was not read.
	header, err = r.Next()
	require.NoError(t, err)
	require.Equal(t, "second", header.Name)
	body, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "two", string(body))
}

func TestRpgReaderEntriesWithKnownSizes(t *testing.T) {
	data := "raw stored data"
	good := rpgRawZip(t, "raw.txt", zip.Store, data, crc32.ChecksumIEEE([]byte(data)))

	r := NewReader(bytes.NewReader(good))
	header, err := r.Next()
	require.NoError(t, err)
	require.Equal(t, "raw.txt", header.Name)
	body, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, data, string(body))

	bad := rpgRawZip(t, "raw.txt", zip.Store, data, crc32.ChecksumIEEE([]byte(data))+1)
	r = NewReader(bytes.NewReader(bad))
	_, err = r.Next()
	require.NoError(t, err)
	_, err = io.ReadAll(r)
	require.ErrorIs(t, err, zip.ErrChecksum)
}

func TestRpgReaderUnsupportedMethod(t *testing.T) {
	archive := rpgRawZip(t, "odd.bin", 99, "data", 0)
	_, err := NewReader(bytes.NewReader(archive)).Next()
	require.ErrorIs(t, err, zip.ErrAlgorithm)
}

func TestRpgRegisterDecompressor(t *testing.T) {
	rpgRegisterOnce.Do(func() { RegisterDecompressor(rpgCustomMethod, io.NopCloser) })
	require.NotNil(t, decompressor(rpgCustomMethod))
	require.Panics(t, func() { RegisterDecompressor(rpgCustomMethod, io.NopCloser) })

	archive := rpgRawZip(t, "custom.bin", rpgCustomMethod, "custom data", 0)
	r := NewReader(bytes.NewReader(archive))
	_, err := r.Next()
	require.NoError(t, err)
	body, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "custom data", string(body))
}

func TestRpgReaderInvalidInput(t *testing.T) {
	_, err := NewReader(bytes.NewReader(nil)).Next()
	require.ErrorIs(t, err, io.EOF)

	_, err = NewReader(strings.NewReader("not a zip file")).Next()
	require.ErrorIs(t, err, io.EOF)

	// A header cut short in the middle is reported as an unexpected EOF.
	archive := rpgRawZip(t, "cut.bin", zip.Store, strings.Repeat("x", 200), 0)
	_, err = NewReader(bytes.NewReader(archive[:20])).Next()
	require.Error(t, err)

	// Garbage in front of the archive is skipped.
	padded := append(bytes.Repeat([]byte{1}, 5000), rpgZip(t, map[string]string{"a": "b"}, []string{"a"})...)
	header, err := NewReader(bytes.NewReader(padded)).Next()
	require.NoError(t, err)
	require.Equal(t, "a", header.Name)
}
