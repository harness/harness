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

	"github.com/harness/gitness/registry/app/manifest"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRsg_PathSpecMarkerMethods(t *testing.T) {
	var specs []pathSpec
	specs = append(specs,
		blobsPathSpec{},
		blobPathSpec{},
		blobDataPathSpec{},
		uploadDataPathSpec{},
		uploadFilePathSpec{},
		genericUploadDataPathSpec{},
		genericDataPathSpec{},
		uploadHashStatePathSpec{},
		globalBlobPathSpec{},
		globalUploadDataPathSpec{},
		globalUploadHashStatePathSpec{},
		repositoriesRootPathSpec{},
	)
	for _, spec := range specs {
		spec.pathSpec()
	}
}

func TestRsg_GenericBlobStoreDelete(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGenericBlobStore(t, rsgNewFilesystemDriver(t))
	path := "/rsg/to-delete"
	require.NoError(t, store.driver.PutContent(ctx, path, []byte("x")))
	require.NoError(t, store.Delete(ctx, path))
}

func TestRsg_GenericBlobStoreGetGenericRedirectFallback(t *testing.T) {
	ctx := context.Background()
	store := &genericBlobStore{
		driver:        rsgNewFilesystemDriver(t),
		rootParentRef: rsgAcctPrefix,
		redirect:      true,
	}
	w, err := store.CreateGeneric(ctx, rsgGenericRoot)
	require.NoError(t, err)
	require.NoError(t, w.PlainCommit(ctx, rsgGenericSHA))

	fr, url, err := store.GetGeneric(ctx, 1, "f", rsgGenericRoot, rsgGenericSHA)
	require.NoError(t, err)
	assert.Empty(t, url)
	require.NotNil(t, fr)
	fr.Close()
}

func TestRsg_GlobalBlobStoreMultipartWrite(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGlobalBlobStore(t, rsgNewFilesystemDriver(t))
	store.multipartEnabled = true

	w, err := store.Create(ctx)
	require.NoError(t, err)
	info, err := store.Write(ctx, w, nil, strings.NewReader(rsgPayload))
	require.NoError(t, err)
	assert.NotEmpty(t, info.Sha256)

	dgst := rsgHelloDigest(t)
	desc, err := w.Commit(ctx, rsgAcctPrefix, manifest.Descriptor{Digest: dgst, Size: int64(len(rsgPayload))})
	require.NoError(t, err)
	assert.Equal(t, dgst, desc.Digest)
}

func TestRsg_GlobalBlobStoreGetGeneric(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGlobalBlobStore(t, rsgNewFilesystemDriver(t))
	payload := []byte(rsgPayload)

	w, err := store.CreateGeneric(ctx, rsgGenericRoot)
	require.NoError(t, err)
	require.NoError(t, w.PlainCommit(ctx, rsgGenericSHA))
	err = store.driver.PutContent(ctx, "/"+rsgGenericRoot+"/files/"+rsgGenericSHA, payload)
	require.NoError(t, err)

	fr, redirect, err := store.GetGeneric(ctx, int64(len(payload)), "f.bin", rsgGenericRoot, rsgGenericSHA)
	require.NoError(t, err)
	assert.Empty(t, redirect)
	require.NotNil(t, fr)
	fr.Close()
}

func TestRsg_BlobWriterInvalidCommitDigest(t *testing.T) {
	ctx := context.Background()
	store := rsgNewOciBlobStore(t, rsgNewFilesystemDriver(t))
	w, err := store.Create(ctx)
	require.NoError(t, err)
	_, err = w.Write([]byte(rsgPayload))
	require.NoError(t, err)

	bad := digest.FromBytes([]byte("other"))
	_, err = w.Commit(ctx, rsgAcctPrefix, manifest.Descriptor{Digest: bad, Size: int64(len(rsgPayload))})
	require.Error(t, err)
}

func TestRsg_OciResumableDigestEnabled(t *testing.T) {
	ctx := context.Background()
	store := rsgNewOciBlobStore(t, rsgNewFilesystemDriver(t))
	store.resumableDigestEnabled = true

	w, err := store.Create(ctx)
	require.NoError(t, err)
	_, err = w.Write([]byte(rsgPayload))
	require.NoError(t, err)

	dgst := rsgHelloDigest(t)
	_, err = w.Commit(ctx, rsgAcctPrefix, manifest.Descriptor{Digest: dgst, Size: int64(len(rsgPayload))})
	require.NoError(t, err)
}

func TestRsg_GlobalBlobWriterPlainCommit(t *testing.T) {
	ctx := context.Background()
	store := rsgNewGlobalBlobStore(t, rsgNewFilesystemDriver(t))

	w, err := store.CreateGeneric(ctx, rsgGenericRoot)
	require.NoError(t, err)
	_, err = w.Write([]byte(rsgPayload))
	require.NoError(t, err)
	require.NoError(t, w.PlainCommit(ctx, rsgGenericSHA))
}

func TestRsg_BlobWriterReaderMethod(t *testing.T) {
	ctx := context.Background()
	store := rsgNewOciBlobStore(t, rsgNewFilesystemDriver(t))
	w, err := store.Create(ctx)
	require.NoError(t, err)
	_, err = w.Write([]byte(rsgPayload))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	bw, ok := w.(*blobWriter)
	require.True(t, ok)
	rc, err := bw.Reader()
	require.NoError(t, err)
	defer rc.Close()
	_, err = io.Copy(io.Discard, rc)
	require.NoError(t, err)
}

func TestRsg_FileReaderSeekEnd(t *testing.T) {
	ctx := context.Background()
	drv := rsgNewFilesystemDriver(t)
	payload := []byte(rsgPayload)
	require.NoError(t, drv.PutContent(ctx, "/rsg/seek", payload))

	fr, err := NewFileReader(ctx, drv, "/rsg/seek", int64(len(payload)))
	require.NoError(t, err)
	defer fr.Close()

	_, err = fr.Seek(-2, io.SeekEnd)
	require.NoError(t, err)
	buf := make([]byte, 2)
	n, err := fr.Read(buf)
	require.ErrorIs(t, err, io.EOF)
	assert.Equal(t, 2, n)
	assert.Equal(t, payload[len(payload)-2:], buf)
}

func TestRsg_LimitedReaderStickyError(t *testing.T) {
	r, ok := limitReader(bytes.NewReader([]byte("ab")), 1).(*limitedReader)
	require.True(t, ok)
	_, errAll := io.ReadAll(r)
	require.Error(t, errAll)
	_, err := r.Read(make([]byte, 1))
	require.Error(t, err)
}
