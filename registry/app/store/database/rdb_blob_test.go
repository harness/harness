// Copyright 2026 Harness, Inc.
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

package database

import (
	"testing"

	"github.com/harness/gitness/registry/types"
	gstore "github.com/harness/gitness/store"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func TestRdbMediaTypes(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx

	id, err := e.mediaTypes.MapMediaType(ctx, "application/octet-stream")
	require.NoError(t, err)
	require.NotZero(t, id)

	_, err = e.mediaTypes.MapMediaType(ctx, "application/x-unknown")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	exists, err := e.mediaTypes.MediaTypeExists(ctx, "application/octet-stream")
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = e.mediaTypes.MediaTypeExists(ctx, "application/x-unknown")
	require.NoError(t, err)
	require.False(t, exists)

	mt, err := e.mediaTypes.GetMediaTypeByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "application/octet-stream", mt.MediaType)
	require.True(t, mt.IsRunnable)

	cosignID, err := e.mediaTypes.MapMediaType(ctx, "application/vnd.dsse.envelope.v1+json")
	require.NoError(t, err)
	cosign, err := e.mediaTypes.GetMediaTypeByID(ctx, cosignID)
	require.NoError(t, err)
	require.False(t, cosign.IsRunnable)

	_, err = e.mediaTypes.GetMediaTypeByID(ctx, 99999)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
}

func TestRdbBlob_CreateOrFindAndFind(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx

	dgst := rdbDigest(1)
	b := &types.Blob{
		RootParentID: rdbRootSpaceID,
		MediaType:    "application/octet-stream",
		Digest:       dgst,
		Size:         123,
	}
	created, isNew, err := e.blobs.CreateOrFind(ctx, b)
	require.NoError(t, err)
	require.True(t, isNew)
	require.NotZero(t, created.ID)
	require.Equal(t, dgst, created.Digest)
	require.Equal(t, int64(123), created.Size)
	require.Equal(t, "application/octet-stream", created.MediaType)

	// Creating the same blob again finds the existing record.
	again, isNew, err := e.blobs.CreateOrFind(ctx, &types.Blob{
		RootParentID: rdbRootSpaceID,
		MediaType:    "application/octet-stream",
		Digest:       dgst,
		Size:         999,
	})
	require.NoError(t, err)
	require.False(t, isNew)
	require.Equal(t, created.ID, again.ID)
	require.Equal(t, int64(123), again.Size)

	byID, err := e.blobs.FindByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, dgst, byID.Digest)

	byDigest, err := e.blobs.FindByDigestAndRootParentID(ctx, dgst, rdbRootSpaceID)
	require.NoError(t, err)
	require.Equal(t, created.ID, byDigest.ID)

	_, err = e.blobs.FindByDigestAndRootParentID(ctx, dgst, rdbRootSpaceID+1)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.blobs.FindByID(ctx, 12345)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// Invalid digests are rejected before hitting the database.
	_, err = e.blobs.FindByDigestAndRootParentID(ctx, digest.Digest("bogus"), 1)
	require.Error(t, err)
	_, err = e.blobs.FindByDigestAndRepoID(ctx, digest.Digest("bogus"), 1, "img")
	require.Error(t, err)
	_, err = e.blobs.ExistsBlob(ctx, 1, digest.Digest("bogus"), "img")
	require.Error(t, err)
	_, _, err = e.blobs.CreateOrFind(ctx, &types.Blob{
		RootParentID: 1, MediaType: "application/octet-stream", Digest: "bogus",
	})
	require.Error(t, err)

	// Unknown media types cannot be mapped.
	_, _, err = e.blobs.CreateOrFind(ctx, &types.Blob{
		RootParentID: 1, MediaType: "application/x-unknown", Digest: rdbDigest(2),
	})
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
}

func TestRdbBlob_TotalSizeAndDelete(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx

	size, err := e.blobs.TotalSizeByRootParentID(ctx, rdbRootSpaceID)
	require.NoError(t, err)
	require.Zero(t, size)

	b1 := e.newBlob(t, 100)
	e.newBlob(t, 250)

	size, err = e.blobs.TotalSizeByRootParentID(ctx, rdbRootSpaceID)
	require.NoError(t, err)
	require.Equal(t, int64(350), size)

	require.NoError(t, e.blobs.DeleteByID(ctx, b1.ID))
	size, err = e.blobs.TotalSizeByRootParentID(ctx, rdbRootSpaceID)
	require.NoError(t, err)
	require.Equal(t, int64(250), size)

	// Deleting a missing blob is a no-op.
	require.NoError(t, e.blobs.DeleteByID(ctx, b1.ID))
}

func TestRdbRegistryBlobs_LinkUnlinkAndLookup(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	rbDao := NewRegistryBlobDao(e.db)

	reg := e.newDockerRegistry(t, "docker")
	blob := e.newBlob(t, 10)

	exists, err := e.blobs.ExistsBlob(ctx, reg.ID, blob.Digest, "img")
	require.NoError(t, err)
	require.False(t, exists)

	require.NoError(t, rbDao.LinkBlob(ctx, "img", reg, blob.ID))
	// Linking twice is idempotent.
	require.NoError(t, rbDao.LinkBlob(ctx, "img", reg, blob.ID))

	exists, err = e.blobs.ExistsBlob(ctx, reg.ID, blob.Digest, "img")
	require.NoError(t, err)
	require.True(t, exists)

	found, err := e.blobs.FindByDigestAndRepoID(ctx, blob.Digest, reg.ID, "img")
	require.NoError(t, err)
	require.Equal(t, blob.ID, found.ID)

	_, err = e.blobs.FindByDigestAndRepoID(ctx, blob.Digest, reg.ID, "other")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// Linking a blob that does not exist violates the foreign key.
	err = rbDao.LinkBlob(ctx, "img", reg, blob.ID+1000)
	require.ErrorIs(t, err, gstore.ErrForeignKeyViolation)

	unlinked, err := rbDao.UnlinkBlob(ctx, "img", reg, blob.ID)
	require.NoError(t, err)
	require.True(t, unlinked)
	unlinked, err = rbDao.UnlinkBlob(ctx, "img", reg, blob.ID)
	require.NoError(t, err)
	require.False(t, unlinked)

	require.NoError(t, rbDao.LinkBlob(ctx, "img", reg, blob.ID))
	require.NoError(t, rbDao.LinkBlob(ctx, "img2", reg, blob.ID))
	removed, err := rbDao.UnlinkBlobByImageName(ctx, reg.ID, "img")
	require.NoError(t, err)
	require.True(t, removed)
	removed, err = rbDao.UnlinkBlobByImageName(ctx, reg.ID, "img")
	require.NoError(t, err)
	require.False(t, removed)
}
