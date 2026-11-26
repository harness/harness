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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRsg_GenericBlobStorePlainCommit(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGenericBlobStore(t, rsgNewFilesystemDriver(t))
	payload := []byte(rsgPayload)

	w, err := store.CreateGeneric(ctx, rsgGenericRoot)
	require.NoError(t, err)
	_, err = w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.PlainCommit(ctx, rsgGenericSHA))

	size, err := store.StatByDigest(ctx, rsgGenericRoot, rsgGenericSHA)
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), size)
}

func TestRsg_GenericBlobStoreGetV2(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGenericBlobStore(t, rsgNewFilesystemDriver(t))
	payload := []byte(rsgPayload)

	w, err := store.CreateGeneric(ctx, rsgGenericRoot)
	require.NoError(t, err)
	_, err = w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.PlainCommit(ctx, rsgGenericSHA))

	fr, err := store.GetV2NoRedirect(ctx, rsgGenericRoot, rsgGenericSHA, int64(len(payload)))
	require.NoError(t, err)
	defer fr.Close()
}

func TestRsg_GenericBlobStoreGetGeneric(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGenericBlobStore(t, rsgNewFilesystemDriver(t))
	payload := []byte(rsgPayload)

	w, err := store.CreateGeneric(ctx, rsgGenericRoot)
	require.NoError(t, err)
	_, err = w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.PlainCommit(ctx, rsgGenericSHA))

	fr, redirect, err := store.GetGeneric(ctx, int64(len(payload)), "file.bin", rsgGenericRoot, rsgGenericSHA)
	require.NoError(t, err)
	assert.Empty(t, redirect)
	require.NotNil(t, fr)
	defer fr.Close()
}

func TestRsg_GenericBlobStoreWriteReader(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGenericBlobStore(t, rsgNewFilesystemDriver(t))

	w, err := store.CreateGeneric(ctx, rsgGenericRoot)
	require.NoError(t, err)

	info, err := store.Write(ctx, w, nil, strings.NewReader(rsgPayload))
	require.NoError(t, err)
	assert.NotEmpty(t, info.Sha256)
}

func TestRsg_GenericBlobStoreWriteBytes(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGenericBlobStore(t, rsgNewFilesystemDriver(t))

	w, err := store.CreateGeneric(ctx, rsgGenericRoot)
	require.NoError(t, err)

	_, err = store.Write(ctx, w, nil, bytes.NewReader([]byte(rsgPayload)))
	require.NoError(t, err)
}
