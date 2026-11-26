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
	"net/http"
	"testing"

	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRsg_GlobalBlobStoreOCIPutGet(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGlobalBlobStore(t, rsgNewFilesystemDriver(t))
	payload := []byte(rsgPayload)
	dgst := rsgHelloDigest(t)

	desc, err := store.Put(ctx, rsgAcctPrefix, payload)
	require.NoError(t, err)
	assert.Equal(t, dgst, desc.Digest)

	got, err := store.Get(ctx, rsgAcctPrefix, dgst)
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	stat, err := store.Stat(ctx, rsgAcctPrefix, dgst)
	require.NoError(t, err)
	assert.Equal(t, dgst, stat.Digest)
}

func TestRsg_GlobalBlobStoreUploadOCI(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGlobalBlobStore(t, rsgNewFilesystemDriver(t))
	payload := []byte(rsgPayload)
	dgst := rsgHelloDigest(t)

	w, err := store.Create(ctx)
	require.NoError(t, err)
	_, err = w.Write(payload)
	require.NoError(t, err)

	desc, err := w.Commit(ctx, rsgAcctPrefix, manifest.Descriptor{Digest: dgst, Size: int64(len(payload))})
	require.NoError(t, err)
	assert.Equal(t, dgst, desc.Digest)
}

func TestRsg_GlobalBlobStoreGenericFlow(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGlobalBlobStore(t, rsgNewFilesystemDriver(t))
	payload := []byte(rsgPayload)

	w, err := store.CreateGeneric(ctx, rsgGenericRoot)
	require.NoError(t, err)
	_, err = w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.PlainCommit(ctx, rsgGenericSHA))

	size, err := store.StatByDigest(ctx, rsgGenericRoot, rsgGenericSHA)
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), size)

	encoded := "sha256:" + rsgGenericSHA
	fr, err := store.GetV2NoRedirect(ctx, rsgGenericRoot, encoded, int64(len(payload)))
	require.NoError(t, err)
	defer fr.Close()
}

func TestRsg_GlobalBlobStoreServeAndOpen(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGlobalBlobStore(t, rsgNewFilesystemDriver(t))
	dgst := rsgHelloDigest(t)
	_, err := store.Put(ctx, rsgAcctPrefix, []byte(rsgPayload))
	require.NoError(t, err)

	headers := map[string]string{}
	fr, redirect, size, err := store.ServeBlobInternal(ctx, rsgAcctPrefix, dgst, headers, http.MethodGet)
	require.NoError(t, err)
	assert.Empty(t, redirect)
	assert.Equal(t, int64(len(rsgPayload)), size)
	require.NotNil(t, fr)
	defer fr.Close()

	rc, err := store.Open(ctx, rsgAcctPrefix, dgst)
	require.NoError(t, err)
	defer rc.Close()

	_, size2, err := store.GetBlobInternal(ctx, rsgAcctPrefix, dgst)
	require.NoError(t, err)
	assert.Equal(t, int64(len(rsgPayload)), size2)
}

func TestRsg_GlobalPathFn(t *testing.T) {
	dgst := rsgHelloDigest(t)
	p, err := GlobalPathFn(dgst)
	require.NoError(t, err)
	assert.Contains(t, p, "/blobs/sha256")
	assert.Contains(t, p, "data")
}

func TestRsg_GlobalBlobStoreResume(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGlobalBlobStore(t, rsgNewFilesystemDriver(t))

	w, err := store.Create(ctx)
	require.NoError(t, err)
	id := w.ID()
	require.NoError(t, w.Cancel(ctx))

	resumed, err := store.Resume(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, resumed)
}

func TestRsg_GlobalBlobStoreMetadata(t *testing.T) {
	store := rsgNewGlobalBlobStore(t, rsgNewFilesystemDriver(t))
	assert.Equal(t, types.BucketKey(rsgBucketCustom), store.BucketKey())
	assert.Empty(t, store.Path())
}
