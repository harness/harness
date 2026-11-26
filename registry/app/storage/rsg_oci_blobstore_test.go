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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRsg_OciBlobStorePutGetStat(t *testing.T) {
	ctx := context.Background()
	store := rsgNewOciBlobStore(t, rsgNewFilesystemDriver(t))
	payload := []byte(rsgPayload)
	dgst := rsgHelloDigest(t)

	_, err := store.Stat(ctx, rsgAcctPrefix, dgst)
	require.ErrorIs(t, err, ErrBlobUnknown)

	desc, err := store.Put(ctx, rsgAcctPrefix, payload)
	require.NoError(t, err)
	assert.Equal(t, dgst, desc.Digest)
	assert.Equal(t, int64(len(payload)), desc.Size)

	again, err := store.Put(ctx, rsgAcctPrefix, payload)
	require.NoError(t, err)
	assert.Equal(t, desc.Digest, again.Digest)

	stat, err := store.Stat(ctx, rsgAcctPrefix, dgst)
	require.NoError(t, err)
	assert.Equal(t, dgst, stat.Digest)

	got, err := store.Get(ctx, rsgAcctPrefix, dgst)
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	rc, err := store.Open(ctx, rsgAcctPrefix, dgst)
	require.NoError(t, err)
	defer rc.Close()
}

func TestRsg_OciBlobStoreUploadCommit(t *testing.T) {
	ctx := context.Background()
	store := rsgNewOciBlobStore(t, rsgNewFilesystemDriver(t))
	payload := []byte(rsgPayload)
	dgst := rsgHelloDigest(t)

	w, err := store.Create(ctx)
	require.NoError(t, err)
	_, err = w.Write(payload)
	require.NoError(t, err)

	desc, err := w.Commit(ctx, rsgAcctPrefix, manifest.Descriptor{Digest: dgst, Size: int64(len(payload))})
	require.NoError(t, err)
	assert.Equal(t, dgst, desc.Digest)

	_, err = store.Stat(ctx, rsgAcctPrefix, dgst)
	require.NoError(t, err)
}

func TestRsg_OciBlobStoreUploadCancel(t *testing.T) {
	ctx := context.Background()
	store := rsgNewOciBlobStore(t, rsgNewFilesystemDriver(t))

	w, err := store.Create(ctx)
	require.NoError(t, err)
	_, err = w.Write([]byte("partial"))
	require.NoError(t, err)
	require.NoError(t, w.Cancel(ctx))
}

func TestRsg_OciBlobStoreServeBlob(t *testing.T) {
	ctx := context.Background()
	store := rsgNewOciBlobStore(t, rsgNewFilesystemDriver(t))
	payload := []byte(rsgPayload)
	dgst := rsgHelloDigest(t)
	_, err := store.Put(ctx, rsgAcctPrefix, payload)
	require.NoError(t, err)

	headers := map[string]string{}
	fr, redirect, size, err := store.ServeBlobInternal(ctx, rsgAcctPrefix, dgst, headers, http.MethodGet)
	require.NoError(t, err)
	assert.Empty(t, redirect)
	assert.Equal(t, int64(len(payload)), size)
	require.NotNil(t, fr)
	defer fr.Close()
	assert.NotEmpty(t, headers[HeaderDockerContentDigest])
}

func TestRsg_OciBlobStoreGetBlobInternal(t *testing.T) {
	ctx := context.Background()
	store := rsgNewOciBlobStore(t, rsgNewFilesystemDriver(t))
	dgst := rsgHelloDigest(t)
	_, err := store.Put(ctx, rsgAcctPrefix, []byte(rsgPayload))
	require.NoError(t, err)

	fr, size, err := store.GetBlobInternal(ctx, rsgAcctPrefix, dgst)
	require.NoError(t, err)
	require.NotNil(t, fr)
	defer fr.Close()
	assert.Equal(t, int64(len(rsgPayload)), size)
}

func TestRsg_OciBlobStoreResume(t *testing.T) {
	ctx := context.Background()
	store := rsgNewOciBlobStore(t, rsgNewFilesystemDriver(t))

	w, err := store.Create(ctx)
	require.NoError(t, err)
	id := w.ID()
	_, err = w.Write([]byte("part1"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	w2, err := store.Resume(ctx, id)
	require.NoError(t, err)
	_, err = w2.Write([]byte("part2"))
	require.NoError(t, err)
	require.NoError(t, w2.Cancel(ctx))
}

func TestRsg_OciBlobStorePathAndBucket(t *testing.T) {
	store := rsgNewOciBlobStore(t, rsgNewFilesystemDriver(t))
	assert.Equal(t, rsgAcctPrefix, store.Path())
	assert.Equal(t, DefaultBucketKey, store.BucketKey())
	assert.ErrorIs(t, store.Delete(context.Background(), rsgAcctPrefix, rsgHelloDigest(t)), ErrUnsupported)
}
