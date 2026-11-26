// Copyright 2023 Harness, Inc.
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

package storage

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRsg_FileReaderReadSeek(t *testing.T) {
	ctx := context.Background()
	drv := rsgNewFilesystemDriver(t)
	path := "/files/rsg/readme"
	payload := []byte(rsgPayload)
	require.NoError(t, drv.PutContent(ctx, path, payload))

	fr, err := NewFileReader(ctx, drv, path, int64(len(payload)))
	require.NoError(t, err)
	defer fr.Close()

	buf := make([]byte, 3)
	n, err := fr.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "rsg", string(buf))

	off, err := fr.Seek(4, io.SeekStart)
	require.NoError(t, err)
	assert.Equal(t, int64(4), off)

	n, err = fr.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "blo", string(buf[:n]))
}

func TestRsg_FileReaderSeekNegative(t *testing.T) {
	ctx := context.Background()
	drv := rsgNewFilesystemDriver(t)
	fr, err := NewFileReader(ctx, drv, "/noop", 0)
	require.NoError(t, err)
	defer fr.Close()

	_, err = fr.Seek(-1, io.SeekStart)
	require.Error(t, err)
}

func TestRsg_FileReaderMissingPath(t *testing.T) {
	ctx := context.Background()
	drv := rsgNewFilesystemDriver(t)
	fr, err := NewFileReader(ctx, drv, "/missing", 0)
	require.NoError(t, err)
	defer fr.Close()

	buf := make([]byte, 4)
	n, err := fr.Read(buf)
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, 0, n)
}

func TestRsg_FileReaderCloseSticky(t *testing.T) {
	ctx := context.Background()
	drv := rsgNewFilesystemDriver(t)
	fr, err := NewFileReader(ctx, drv, "/x", 0)
	require.NoError(t, err)

	err = fr.Close()
	require.Error(t, err)

	_, err = fr.Read(make([]byte, 1))
	require.Error(t, err)
}
