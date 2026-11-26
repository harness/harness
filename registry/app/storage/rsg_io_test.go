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
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRsg_LimitedReaderWithinLimit(t *testing.T) {
	r := limitReader(strings.NewReader("abcd"), 10)
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "abcd", string(out))
}

func TestRsg_LimitedReaderExceedsLimit(t *testing.T) {
	r := limitReader(strings.NewReader("abcdefgh"), 3)
	_, err := io.ReadAll(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read exceeds limit")
}

func TestRsg_LimitedReaderEmptyRead(t *testing.T) {
	r, ok := limitReader(strings.NewReader("x"), 5).(*limitedReader)
	require.True(t, ok)
	n, err := r.Read([]byte{})
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestRsg_GetContentViaDriver(t *testing.T) {
	ctx := context.Background()
	drv := rsgNewFilesystemDriver(t)
	path := "/data/rsg/test.bin"
	payload := []byte(rsgPayload)
	require.NoError(t, drv.PutContent(ctx, path, payload))

	got, err := getContent(ctx, drv, path)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestRsg_GetContentNotFound(t *testing.T) {
	ctx := context.Background()
	drv := rsgNewFilesystemDriver(t)
	_, err := getContent(ctx, drv, "/missing/object")
	require.Error(t, err)
}

func TestRsg_GetContentTooLarge(t *testing.T) {
	ctx := context.Background()
	drv := rsgNewFilesystemDriver(t)
	path := "/data/rsg/big.bin"
	big := bytes.Repeat([]byte("x"), maxBlobGetSize+1)
	require.NoError(t, drv.PutContent(ctx, path, big))

	_, err := getContent(ctx, drv, path)
	require.Error(t, err)
}
