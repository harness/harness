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

package docker

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg"
	refcache "github.com/harness/gitness/registry/app/services/refcache"
	registrystorage "github.com/harness/gitness/registry/app/storage"
	registrytypes "github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"

	"github.com/distribution/distribution/v3"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

// rpgBucket hands out a fixed blob store (nil falls back to the default store).
type rpgBucket struct {
	store *BlobStore
	calls int
}

func (b *rpgBucket) GetBlobStore(context.Context, string, string, any, string, string) *BlobStore {
	b.calls++
	return b.store
}

// rpgOciStore overrides the blob store calls made when serving blobs.
type rpgOciStoreFake struct {
	registrystorage.OciBlobStore
	serve func(method string) (*registrystorage.FileReader, string, int64, error)
}

func (s *rpgOciStoreFake) ServeBlobInternal(
	_ context.Context, _ string, _ digest.Digest, headers map[string]string, method string,
) (*registrystorage.FileReader, string, int64, error) {
	headers["X-Served"] = "yes"
	return s.serve(method)
}

func (s *rpgOciStoreFake) BucketKey() registrytypes.BucketKey { return "bucket" }

func rpgStateFromLocation(t *testing.T, location string) string {
	t.Helper()
	u, err := url.Parse(location)
	require.NoError(t, err)
	state := u.Query().Get("_state")
	require.NotEmpty(t, state)
	return state
}

// rpgUpload initiates an upload and returns its upload id and state token.
func rpgUpload(ctx context.Context, t *testing.T, env *rpgEnv, info pkg.RegistryInfo) (string, string) {
	t.Helper()
	headers, errs := env.local.InitBlobUpload(ctx, info, "", "")
	require.Empty(t, errs)
	require.Equal(t, http.StatusAccepted, headers.Code)
	return headers.Headers["Docker-Upload-UUID"], rpgStateFromLocation(t, headers.Headers["Location"])
}

func rpgUploadContext(
	ctx context.Context, t *testing.T, env *rpgEnv, info pkg.RegistryInfo, uuid, token string,
) *Context {
	t.Helper()
	info.Reference = uuid
	blobCtx, err := env.app.GetBlobsContext(ctx, info, registrytypes.BlobLocator{})
	require.NoError(t, err)
	require.Empty(t, ResumeBlobUpload(blobCtx, token)) //nolint:contextcheck
	return blobCtx
}

func TestRpgBlobLifecycle(t *testing.T) {
	ctx := context.Background()
	env := rpgNewEnv(t)
	env.app.bucketService = &rpgBucket{}
	info := rpgInfo(t)
	content := []byte("hello blob content")
	dgst := digest.FromBytes(content)

	// Chunked upload: initiate, patch, then complete with the digest.
	uploadID, token := rpgUpload(ctx, t, env, info)
	require.NotEmpty(t, uploadID)

	blobCtx := rpgUploadContext(ctx, t, env, info, uploadID, token)
	headers, errs := env.local.PushBlobChunk(
		blobCtx, info, "application/octet-stream", "0-8", "9", io.NopCloser(bytes.NewReader(content[:9])), 9,
	)
	require.Empty(t, errs)
	require.Equal(t, http.StatusAccepted, headers.Code)
	require.Equal(t, "0-8", headers.Headers["Range"])
	token = rpgStateFromLocation(t, headers.Headers["Location"])

	info.Reference = uploadID
	info.Digest = dgst.String()
	headers, errs = env.local.PushBlob(
		ctx, info, io.NopCloser(bytes.NewReader(content[9:])), int64(len(content)-9), token,
	)
	require.Empty(t, errs)
	require.Equal(t, http.StatusCreated, headers.Code)
	require.Equal(t, dgst.String(), headers.Headers["Docker-Content-Digest"])
	require.Contains(t, headers.Headers["Location"], "/blobs/")
	require.Equal(t, []int64{77}, env.regBlobs.linked)
	require.Equal(t, 1, env.hook.commits)
	require.Equal(t, 1, env.repl.calls)

	// The committed blob can be inspected and downloaded.
	info.Digest = dgst.String()
	head, errs := env.local.HeadBlob(ctx, info)
	require.Empty(t, errs)
	require.Equal(t, "123", head.Headers["Content-Length"])
	require.Equal(t, `"`+dgst.String()+`"`, head.Headers["Etag"])

	getHeaders, reader, size, _, redirect, errs := env.local.GetBlob(ctx, info)
	require.Empty(t, errs)
	require.Empty(t, redirect)
	require.Equal(t, int64(len(content)), size)
	require.Equal(t, dgst.String(), getHeaders.Headers["Docker-Content-Digest"])
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, content, body)
}

// rpgWriter is a blob writer whose behaviour is scripted by the test.
type rpgWriter struct {
	size      int64
	writeErr  error
	commitErr error
	cancelErr error
	canceled  int
}

func (w *rpgWriter) Write(p []byte) (int, error) {
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	w.size += int64(len(p))
	return len(p), nil
}

func (w *rpgWriter) Close() error { return nil }
func (w *rpgWriter) Size() int64  { return w.size }
func (w *rpgWriter) ID() string   { return "scripted-upload" }

func (w *rpgWriter) Commit(context.Context, string, manifest.Descriptor) (manifest.Descriptor, error) {
	return manifest.Descriptor{}, w.commitErr
}

func (w *rpgWriter) PlainCommit(context.Context, string) error { return nil }

func (w *rpgWriter) Cancel(context.Context) error {
	w.canceled++
	return w.cancelErr
}

func TestRpgPushBlobChunkValidation(t *testing.T) {
	env := rpgNewEnv(t)
	info := rpgInfo(t)
	newCtx := func(w *rpgWriter) *Context {
		blobCtx, err := env.app.GetBlobsContext(context.Background(), info, registrytypes.BlobLocator{})
		require.NoError(t, err)
		if w != nil {
			blobCtx.Upload = w
		}
		return blobCtx
	}
	body := func() io.ReadCloser { return io.NopCloser(strings.NewReader("12345")) }

	tests := []struct {
		name        string
		writer      *rpgWriter
		contentType string
		cr, cl      string
		want        errcode.CodeError
	}{
		{"no upload", nil, "", "", "", errcode.ErrCodeBlobUploadUnknown},
		{"bad content type", &rpgWriter{}, "text/plain", "", "", errcode.ErrCodeUnknown},
		{"bad range", &rpgWriter{}, "", "abc", "5", errcode.ErrCodeUnknown},
		{"start after end", &rpgWriter{}, "", "5-1", "5", errcode.ErrCodeRangeInvalid},
		{"start not at size", &rpgWriter{size: 2}, "", "0-4", "5", errcode.ErrCodeRangeInvalid},
		{"bad length", &rpgWriter{}, "", "0-4", "x", errcode.ErrCodeUnknown},
		{"length mismatch", &rpgWriter{}, "", "0-4", "9", errcode.ErrCodeSizeInvalid},
		{"write failure", &rpgWriter{writeErr: errRpgBoom}, "", "0-4", "5", errcode.ErrCodeUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers, errs := env.local.PushBlobChunk(newCtx(tt.writer), info, tt.contentType, tt.cr, tt.cl, body(), 5)
			require.Equal(t, []errcode.CodeError{tt.want}, rpgCodes(errs))
			require.Zero(t, headers.Code)
		})
	}

	t.Run("accepted without range headers", func(t *testing.T) {
		blobCtx := newCtx(nil)
		blobCtx.Upload = &rpgWriter{}
		headers, errs := env.local.PushBlobChunk(blobCtx, info, "", "", "", body(), 5)
		require.Empty(t, errs)
		require.Equal(t, http.StatusAccepted, headers.Code)
		require.Equal(t, "0-4", headers.Headers["Range"])
	})
}

func TestRpgPushBlobErrors(t *testing.T) {
	ctx := context.Background()
	content := []byte("some blob bytes")

	t.Run("no upload id", func(t *testing.T) {
		env := rpgNewEnv(t)
		_, errs := env.local.PushBlob(ctx, rpgInfo(t), io.NopCloser(bytes.NewReader(content)), 1, "")
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeBlobUploadUnknown}, rpgCodes(errs))
	})
	t.Run("storage unavailable", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.app.storageService, _ = registrystorage.NewStorageService(rpgFailingResolver{})
		_, errs := env.local.PushBlob(ctx, rpgInfo(t), io.NopCloser(bytes.NewReader(content)), 1, "")
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
	})
	t.Run("invalid state token", func(t *testing.T) {
		env := rpgNewEnv(t)
		info := rpgInfo(t)
		uploadID, _ := rpgUpload(ctx, t, env, info)
		info.Reference = uploadID
		_, errs := env.local.PushBlob(ctx, info, io.NopCloser(bytes.NewReader(content)), 1, "garbage")
		require.Contains(t, rpgCodes(errs), errcode.ErrCodeBlobUploadInvalid)
	})
	t.Run("missing and malformed digest", func(t *testing.T) {
		env := rpgNewEnv(t)
		info := rpgInfo(t)
		uploadID, token := rpgUpload(ctx, t, env, info)
		info.Reference = uploadID

		_, errs := env.local.PushBlob(ctx, info, io.NopCloser(bytes.NewReader(content)), 1, token)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeDigestInvalid}, rpgCodes(errs))

		info.Digest = "not-a-digest"
		_, errs = env.local.PushBlob(ctx, info, io.NopCloser(bytes.NewReader(content)), 1, token)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeDigestInvalid}, rpgCodes(errs))
	})
	t.Run("body failure", func(t *testing.T) {
		env := rpgNewEnv(t)
		info := rpgInfo(t)
		uploadID, token := rpgUpload(ctx, t, env, info)
		info.Reference = uploadID
		info.Digest = digest.FromBytes(content).String()
		_, errs := env.local.PushBlob(ctx, info, io.NopCloser(&rpgFailingReader{}), 1, token)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
	})
	t.Run("digest mismatch cancels the upload", func(t *testing.T) {
		env := rpgNewEnv(t)
		info := rpgInfo(t)
		uploadID, token := rpgUpload(ctx, t, env, info)
		info.Reference = uploadID
		info.Digest = digest.FromString("something else").String()
		_, errs := env.local.PushBlob(ctx, info, io.NopCloser(bytes.NewReader(content)), int64(len(content)), token)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeDigestInvalid}, rpgCodes(errs))
	})
	t.Run("database failure", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.blobs.createOrFind = func(*registrytypes.Blob) (*registrytypes.Blob, bool, error) {
			return nil, false, errRpgBoom
		}
		info := rpgInfo(t)
		uploadID, token := rpgUpload(ctx, t, env, info)
		info.Reference = uploadID
		info.Digest = digest.FromBytes(content).String()
		_, errs := env.local.PushBlob(ctx, info, io.NopCloser(bytes.NewReader(content)), int64(len(content)), token)
		require.Len(t, errs, 1)
		require.ErrorIs(t, errs[0], errRpgBoom)
	})
}

type rpgFailingResolver struct{}

func (rpgFailingResolver) Resolve(context.Context, registrytypes.StorageLookup) (registrystorage.StorageTarget, error) {
	return registrystorage.StorageTarget{}, errRpgBoom
}

func TestRpgInitBlobUpload(t *testing.T) {
	ctx := context.Background()
	mountDigest := digest.FromString("mounted").String()

	t.Run("storage unavailable", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.app.storageService, _ = registrystorage.NewStorageService(rpgFailingResolver{})
		headers, errs := env.local.InitBlobUpload(ctx, rpgInfo(t), "", "")
		require.Nil(t, headers)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
	})
	t.Run("mount", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.local.registryFinder = &rpgRegFinder{reg: &registrytypes.Registry{ID: 6}}
		headers, errs := env.local.InitBlobUpload(ctx, rpgInfo(t), "root/other/imported", mountDigest)
		require.Empty(t, errs)
		require.Equal(t, http.StatusCreated, headers.Code)
		require.Equal(t, mountDigest, headers.Headers["Docker-Content-Digest"])
		require.Equal(t, []int64{9}, env.regBlobs.linked)
	})
	t.Run("mount failures", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(env *rpgEnv)
			from  string
			want  errcode.CodeError
		}{
			{"destination missing", func(env *rpgEnv) { env.regDao.err = gitnessstore.ErrResourceNotFound },
				"root/other/imported", errcode.ErrCodeRegNotFound},
			{"destination nil", func(env *rpgEnv) { env.regDao.reg = nil }, "root/other/imported", errcode.ErrCodeUnknown},
			{"bad source ref", func(*rpgEnv) {}, "/", errcode.ErrCodeUnknown},
			{"bad source image", func(*rpgEnv) {}, "root/", errcode.ErrCodeUnknown},
			{"source lookup fails", func(env *rpgEnv) { env.local.registryFinder = &rpgRegFinder{err: errRpgBoom} },
				"root/other/imported", errcode.ErrCodeUnknown},
			{"source nil", func(env *rpgEnv) { env.local.registryFinder = &rpgRegFinder{} },
				"root/other/imported", errcode.ErrCodeUnknown},
			{"blob missing", func(env *rpgEnv) {
				env.local.registryFinder = &rpgRegFinder{reg: &registrytypes.Registry{ID: 6}}
				env.blobs.byRepo = func(digest.Digest) (*registrytypes.Blob, error) {
					return nil, gitnessstore.ErrResourceNotFound
				}
			}, "root/other/imported", errcode.ErrCodeUnknown},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				env := rpgNewEnv(t)
				tt.setup(env)
				headers, errs := env.local.InitBlobUpload(ctx, rpgInfo(t), tt.from, mountDigest)
				require.NotNil(t, headers)
				require.Equal(t, tt.want, rpgCodes(errs)[0])
			})
		}
	})
}

type rpgRegFinder struct {
	refcache.RegistryFinder
	reg *registrytypes.Registry
	err error
}

func (f *rpgRegFinder) FindByRootParentID(
	context.Context, int64, string, ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return f.reg, f.err
}

func TestRpgGetBlobUploadStatus(t *testing.T) {
	ctx := context.Background()
	env := rpgNewEnv(t)
	info := rpgInfo(t)
	uploadID, token := rpgUpload(ctx, t, env, info)

	// Resumes the upload from storage.
	info.Reference = uploadID
	blobCtx, err := env.app.GetBlobsContext(ctx, info, registrytypes.BlobLocator{})
	require.NoError(t, err)
	headers, errs := env.local.GetBlobUploadStatus(blobCtx, info, token)
	require.Empty(t, errs)
	require.Equal(t, http.StatusNoContent, headers.Code)
	require.Equal(t, "0-0", headers.Headers["Range"])

	// Uses an upload that is already loaded.
	headers, errs = env.local.GetBlobUploadStatus(blobCtx, info, token)
	require.Empty(t, errs)
	require.Equal(t, http.StatusNoContent, headers.Code)

	// Resume failures are mapped to registry errors.
	for resumeErr, want := range map[error]errcode.CodeError{
		distribution.ErrBlobUploadUnknown: errcode.ErrCodeBlobUploadUnknown,
		errRpgBoom:                        errcode.ErrCodeUnknown,
	} {
		failing := &Context{
			App: env.app, Context: ctx, OciBlobStore: &rpgResumeStore{err: resumeErr}, UUID: "x", URLBuilder: info.URLBuilder,
		}
		_, errs = env.local.GetBlobUploadStatus(failing, info, token)
		require.Equal(t, []errcode.CodeError{want}, rpgCodes(errs))
	}
}

type rpgResumeStore struct {
	registrystorage.OciBlobStore
	err error
}

func (s *rpgResumeStore) Resume(context.Context, string) (registrystorage.BlobWriter, error) {
	return nil, s.err
}

func TestRpgDeleteBlob(t *testing.T) {
	ctx := context.Background()
	info := rpgInfo(t)
	info.Digest = digest.FromString("blob").String()

	tests := []struct {
		name  string
		setup func(env *rpgEnv)
		want  []errcode.CodeError
	}{
		{"success", func(*rpgEnv) {}, nil},
		{"deletion disabled", func(env *rpgEnv) { env.app.Config.Registry.Storage.S3Storage.Delete = false },
			[]errcode.CodeError{errcode.ErrCodeUnsupported}},
		{"unknown blob", func(env *rpgEnv) {
			env.blobs.byRepo = func(digest.Digest) (*registrytypes.Blob, error) {
				return nil, gitnessstore.ErrResourceNotFound
			}
		}, []errcode.CodeError{errcode.ErrCodeBlobUnknown}},
		{"lookup failure", func(env *rpgEnv) {
			env.blobs.byRepo = func(digest.Digest) (*registrytypes.Blob, error) { return nil, errRpgBoom }
		}, []errcode.CodeError{errcode.ErrCodeUnknown}},
		{"not linked", func(env *rpgEnv) { env.regBlobs.unlink = false }, []errcode.CodeError{errcode.ErrCodeBlobUnknown}},
		{"unlink failure", func(env *rpgEnv) { env.regBlobs.unlinkErr = errRpgBoom },
			[]errcode.CodeError{errcode.ErrCodeUnknown}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := rpgNewEnv(t)
			tt.setup(env)
			headers, errs := env.local.DeleteBlob(&Context{App: env.app, Context: ctx}, info) //nolint:contextcheck
			require.Equal(t, append([]errcode.CodeError{}, tt.want...), append([]errcode.CodeError{}, rpgCodes(errs)...))
			if tt.want == nil {
				require.Equal(t, http.StatusAccepted, headers.Code)
				require.Equal(t, "0", headers.Headers["Content-Length"])
			}
		})
	}
}

func TestRpgFetchBlobInternalBranches(t *testing.T) {
	ctx := context.Background()
	info := rpgInfo(t)
	info.Digest = digest.FromString("blob").String()

	serving := func(env *rpgEnv, serve func(string) (*registrystorage.FileReader, string, int64, error)) {
		env.app.bucketService = &rpgBucket{store: &BlobStore{OciStore: &rpgOciStoreFake{serve: serve}}}
	}

	t.Run("redirect emits a read event", func(t *testing.T) {
		env := rpgNewEnv(t)
		serving(env, func(string) (*registrystorage.FileReader, string, int64, error) {
			return nil, "https://cdn.example.com/blob", 0, nil
		})
		headers, reader, _, _, redirect, errs := env.local.GetBlob(ctx, info)
		require.Empty(t, errs)
		require.Nil(t, reader)
		require.Equal(t, "https://cdn.example.com/blob", redirect)
		require.Empty(t, headers.Headers)
		require.Equal(t, 1, env.hook.reads)
	})
	t.Run("headers are copied", func(t *testing.T) {
		env := rpgNewEnv(t)
		serving(env, func(method string) (*registrystorage.FileReader, string, int64, error) {
			require.Equal(t, http.MethodGet, method)
			return nil, "", 4, nil
		})
		headers, _, size, _, _, errs := env.local.GetBlob(ctx, info)
		require.Empty(t, errs)
		require.Equal(t, int64(4), size)
		require.Equal(t, "yes", headers.Headers["X-Served"])
	})
	t.Run("serve errors", func(t *testing.T) {
		env := rpgNewEnv(t)
		serving(env, func(string) (*registrystorage.FileReader, string, int64, error) {
			return nil, "", 0, registrystorage.ErrBlobUnknown
		})
		_, _, _, _, _, errs := env.local.GetBlob(ctx, info)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeBlobUnknown}, rpgCodes(errs))

		serving(env, func(string) (*registrystorage.FileReader, string, int64, error) {
			return nil, "", 0, errRpgBoom
		})
		_, _, _, _, _, errs = env.local.GetBlob(ctx, info)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
	})
	t.Run("blob link lookup fails", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.blobs.byRoot = func(digest.Digest) (*registrytypes.Blob, error) {
			return nil, gitnessstore.ErrResourceNotFound
		}
		_, _, _, _, _, errs := env.local.GetBlob(ctx, info)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeBlobUnknown}, rpgCodes(errs))
		_, errs = env.local.HeadBlob(ctx, info)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeBlobUnknown}, rpgCodes(errs))
	})
	t.Run("blob store root missing", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.app.spaceFinder = rpgSpaceFinder(errRpgBoom)
		_, _, _, _, _, errs := env.local.GetBlob(ctx, info)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
	})
	t.Run("head blob lookup fails", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.blobs.byID = func(int64) (*registrytypes.Blob, error) { return nil, errRpgBoom }
		_, errs := env.local.HeadBlob(ctx, info)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
	})
}

func TestRpgDBBlobLinkExists(t *testing.T) {
	ctx := context.Background()
	info := rpgInfo(t)
	dgst := digest.FromString("blob")

	t.Run("found", func(t *testing.T) {
		env := rpgNewEnv(t)
		id, err := env.local.dbBlobLinkExists(ctx, dgst, info)
		require.NoError(t, err)
		require.Equal(t, int64(9), id)
	})
	t.Run("lookup failure", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.blobs.byRoot = func(digest.Digest) (*registrytypes.Blob, error) { return nil, errRpgBoom }
		_, err := env.local.dbBlobLinkExists(ctx, dgst, info)
		require.ErrorIs(t, err, errRpgBoom)
	})
	t.Run("gc lock failure", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.gc.blobErr = errRpgBoom
		_, err := env.local.dbBlobLinkExists(ctx, dgst, info)
		require.ErrorIs(t, err, errRpgBoom)
	})
	t.Run("gc has no review row", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.gc.blobErr = sql.ErrNoRows
		_, err := env.local.dbBlobLinkExists(ctx, dgst, info)
		require.NoError(t, err)
	})
	t.Run("pending review is rescheduled", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.gc.blobTask = &registrytypes.GCBlobTask{BlobID: 9}
		_, err := env.local.dbBlobLinkExists(ctx, dgst, info)
		require.NoError(t, err)
		require.Equal(t, 1, env.gc.rescheduled)

		env.gc.rescErr = errRpgBoom
		_, err = env.local.dbBlobLinkExists(ctx, dgst, info)
		require.ErrorIs(t, err, errRpgBoom)
	})
	t.Run("not linked to the image", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.blobs.exists = func() (bool, error) { return false, nil }
		_, err := env.local.dbBlobLinkExists(ctx, dgst, info)
		require.Error(t, err)
		env.blobs.exists = func() (bool, error) { return false, errRpgBoom }
		_, err = env.local.dbBlobLinkExists(ctx, dgst, info)
		require.ErrorIs(t, err, errRpgBoom)
	})
}

func TestRpgDBPutBlobUploadComplete(t *testing.T) {
	ctx := context.Background()
	info := rpgInfo(t)
	run := func(env *rpgEnv, callback func(context.Context) error) error {
		return env.local.dbPutBlobUploadComplete(ctx, "application/octet-stream", "sha256:x", 3, "path", info, callback)
	}

	env := rpgNewEnv(t)
	require.NoError(t, run(env, nil))
	require.Equal(t, 1, env.repl.calls)

	// Existing blobs do not trigger replication.
	env = rpgNewEnv(t)
	env.blobs.createOrFind = func(b *registrytypes.Blob) (*registrytypes.Blob, bool, error) { return b, false, nil }
	require.NoError(t, run(env, nil))
	require.Zero(t, env.repl.calls)

	// A missing row is tolerated by CreateOrFind.
	env = rpgNewEnv(t)
	env.blobs.createOrFind = func(b *registrytypes.Blob) (*registrytypes.Blob, bool, error) {
		return b, true, gitnessstore.ErrResourceNotFound
	}
	require.NoError(t, run(env, nil))

	env = rpgNewEnv(t)
	env.blobs.createOrFind = func(*registrytypes.Blob) (*registrytypes.Blob, bool, error) { return nil, false, errRpgBoom }
	require.ErrorIs(t, run(env, nil), errRpgBoom)

	env = rpgNewEnv(t)
	env.regBlobs.linkErr = errRpgBoom
	require.ErrorIs(t, run(env, nil), errRpgBoom)

	env = rpgNewEnv(t)
	env.images.getErr = errRpgBoom
	require.ErrorIs(t, run(env, nil), errRpgBoom)

	env = rpgNewEnv(t)
	err := run(env, func(context.Context) error { return errRpgBoom })
	require.ErrorContains(t, err, "failed to execute commit callback")
}

func TestRpgWriteBlobCreatedHeadersErrors(t *testing.T) {
	env := rpgNewEnv(t)
	info := rpgInfo(t)
	blobCtx := &Context{App: env.app, Context: context.Background(), URLBuilder: info.URLBuilder}
	headers := rpgHeaders()
	require.NoError(t, writeBlobCreatedHeaders(blobCtx, digest.FromString("x"), headers, info))
	require.Equal(t, http.StatusCreated, headers.Code)

	info.RegIdentifier = "UPPER CASE"
	require.Error(t, writeBlobCreatedHeaders(blobCtx, digest.FromString("x"), rpgHeaders(), info))

	info = rpgInfo(t)
	require.Error(t, writeBlobCreatedHeaders(blobCtx, "bad", rpgHeaders(), info))
}
