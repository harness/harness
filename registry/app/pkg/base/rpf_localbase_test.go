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

package base

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	commonstype "github.com/harness/gitness/registry/app/pkg/types/commons"
	registryrefcache "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"
	gtypes "github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type rpfMeta struct {
	Files []metadata.File `json:"files"`
	Size  int64           `json:"size"`
}

func (m *rpfMeta) GetFiles() []metadata.File  { return m.Files }
func (m *rpfMeta) SetFiles(f []metadata.File) { m.Files = f }
func (m *rpfMeta) GetSize() int64             { return m.Size }
func (m *rpfMeta) UpdateSize(s int64)         { m.Size += s }

type rpfRegistryFinder struct {
	registryrefcache.RegistryFinder
	err error
}

func (f rpfRegistryFinder) FindByRootParentID(
	context.Context, int64, string, ...types.QueryOption,
) (*types.Registry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.Registry{ID: 3, Name: "reg"}, nil
}

type rpfFM struct {
	filemanager.FileManager
	headErr       error
	uploadErr     error
	postErr       error
	downloadErr   error
	deleteErr     error
	filePathErr   error
	saveNodesErr  error
	posted        []string
	saved         []string
	deleted       []string
	uploadedPaths []string
}

func (f *rpfFM) HeadFile(context.Context, string, int64) (string, int64, error) {
	if f.headErr != nil {
		return "", 0, f.headErr
	}
	return "sha-head", 12, nil
}

func (f *rpfFM) UploadFile(
	_ context.Context, path string, _ int64, _ int64, _ string, _ multipart.File, _ io.Reader, _ int64,
) (types.FileInfo, error) {
	f.uploadedPaths = append(f.uploadedPaths, path)
	return types.FileInfo{Sha256: "sha-new", Size: 7}, f.uploadErr
}

func (f *rpfFM) PostFileUpload(
	_ context.Context, path string, _ int64, _ int64, _ string, _ types.FileInfo, _ int64,
) error {
	f.posted = append(f.posted, path)
	return f.postErr
}

func (f *rpfFM) DownloadFileByPath(
	_ context.Context, _ string, _ int64, _ string, _ string, _ bool,
) (*storage.FileReader, int64, string, error) {
	if f.downloadErr != nil {
		return nil, 0, "", f.downloadErr
	}
	return nil, 0, "https://redirect.test/file", nil
}

func (f *rpfFM) DeleteLeafNode(_ context.Context, _ int64, path string) error {
	f.deleted = append(f.deleted, path)
	return f.deleteErr
}

func (f *rpfFM) GetFilePath(context.Context, string, int64, int64) (string, error) {
	return "/existing", f.filePathErr
}

func (f *rpfFM) SaveNodes(_ context.Context, path string, _ int64, _ int64, _ int64, _ string) error {
	f.saved = append(f.saved, path)
	return f.saveNodesErr
}

type rpfTx struct{ err error }

func (t rpfTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	if t.err != nil {
		return t.err
	}
	return fn(ctx)
}

type rpfImageRepo struct {
	store.ImageRepository
	image     *types.Image
	getErr    error
	upsertErr error
	deleteErr error
}

func (r *rpfImageRepo) GetByName(context.Context, int64, string, ...types.QueryOption) (*types.Image, error) {
	return r.image, r.getErr
}

func (r *rpfImageRepo) CreateOrUpdate(_ context.Context, image *types.Image) error {
	image.ID = 5
	image.UUID = "img-uuid"
	return r.upsertErr
}

func (r *rpfImageRepo) DeleteByImageNameAndRegID(context.Context, int64, string) error {
	return r.deleteErr
}

type rpfArtifactRepo struct {
	store.ArtifactRepository
	existing  *types.Artifact
	getErr    error
	upsertErr error
	deleteErr error
	versionOK bool
	saved     *types.Artifact
}

func (r *rpfArtifactRepo) GetByName(context.Context, int64, string, ...types.QueryOption) (*types.Artifact, error) {
	return r.existing, r.getErr
}

func (r *rpfArtifactRepo) CreateOrUpdate(_ context.Context, a *types.Artifact) (int64, error) {
	a.UUID = "art-uuid"
	r.saved = a
	return 42, r.upsertErr
}

func (r *rpfArtifactRepo) GetByRegistryImageAndVersion(
	context.Context, int64, string, string,
) (*types.Artifact, error) {
	if !r.versionOK {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return &types.Artifact{}, nil
}

func (r *rpfArtifactRepo) DeleteByImageNameAndRegistryID(context.Context, int64, string) error {
	return r.deleteErr
}

func (r *rpfArtifactRepo) DeleteByVersionAndImageName(context.Context, string, string, int64) error {
	return r.deleteErr
}

type rpfNodesRepo struct {
	store.NodesRepository
	err error
}

func (r rpfNodesRepo) DeleteByNodePathAndRegistryID(context.Context, string, int64) error {
	return r.err
}

type rpfAuthorizer struct {
	authz.Authorizer
	allowed bool
}

func (a rpfAuthorizer) CheckAll(context.Context, *auth.Session, ...gtypes.PermissionCheck) (bool, error) {
	return a.allowed, nil
}

type rpfSpaceCache struct{}

func (rpfSpaceCache) Stats() (int64, int64) { return 0, 0 }

func (rpfSpaceCache) Get(_ context.Context, key int64) (*gtypes.SpaceCore, error) {
	if key != 7 {
		return nil, errors.New("space not found")
	}
	return &gtypes.SpaceCore{ID: 7, Path: "acme/team"}, nil
}

func (rpfSpaceCache) Evict(context.Context, int64) {}

type rpfAudit struct {
	audit.Service
	actions []audit.Action
}

func (a *rpfAudit) Log(
	_ context.Context, _ gtypes.Principal, _ audit.Resource, action audit.Action, _ string, _ ...audit.Option,
) error {
	a.actions = append(a.actions, action)
	return nil
}

func rpfAuthedContext() context.Context {
	return request.WithAuthSession(context.Background(), &auth.Session{Principal: gtypes.Principal{ID: 9}})
}

type rpfDeps struct {
	finder   rpfRegistryFinder
	fm       *rpfFM
	tx       rpfTx
	images   *rpfImageRepo
	versions *rpfArtifactRepo
	nodes    rpfNodesRepo
	allowed  bool
}

func rpfNewBase(d rpfDeps) (*localBase, *rpfAudit) {
	if d.fm == nil {
		d.fm = &rpfFM{}
	}
	if d.images == nil {
		d.images = &rpfImageRepo{}
	}
	if d.versions == nil {
		d.versions = &rpfArtifactRepo{}
	}
	auditSvc := &rpfAudit{}
	finder := refcache.NewSpaceFinder(rpfSpaceCache{}, nil, nil, cache.Evictor[*gtypes.SpaceCore]{})
	lb := NewLocalBase(nil, d.finder, d.fm, d.tx, d.images, d.versions, d.nodes, nil,
		rpfAuthorizer{allowed: d.allowed}, finder, auditSvc)
	base, ok := lb.(*localBase)
	if !ok {
		return nil, nil
	}
	return base, auditSvc
}

func rpfArtifactInfo() pkg.ArtifactInfo {
	return pkg.ArtifactInfo{
		BaseInfo:      &pkg.BaseInfo{ParentID: 7, RootIdentifier: "root", RootParentID: 2},
		RegIdentifier: "reg",
		RegistryID:    3,
		Image:         "demo",
	}
}

func rpfPackageInfo() pkg.PackageArtifactInfo {
	return commonstype.ArtifactInfo{ArtifactInfo: rpfArtifactInfo(), Version: "1.0"}
}

func rpfStoredMeta(t *testing.T, names ...string) *types.Artifact {
	t.Helper()
	m := rpfMeta{}
	for _, n := range names {
		m.Files = append(m.Files, metadata.File{Filename: n})
	}
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	return &types.Artifact{Version: "1.0", Metadata: raw}
}

func TestRpfUploadHappyPaths(t *testing.T) {
	ctx := rpfAuthedContext()

	t.Run("first upload of a package", func(t *testing.T) {
		images := &rpfImageRepo{getErr: gitnessstore.ErrResourceNotFound}
		versions := &rpfArtifactRepo{getErr: gitnessstore.ErrResourceNotFound}
		fm := &rpfFM{}
		lb, auditSvc := rpfNewBase(rpfDeps{fm: fm, images: images, versions: versions})
		meta := &rpfMeta{}
		headers, sha, err := lb.Upload(ctx, rpfArtifactInfo(), "demo.bin", "1.0", "demo/1.0/demo.bin",
			io.NopCloser(strings.NewReader("x")), meta)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, headers.Code)
		require.Equal(t, "sha-new", sha)
		require.Equal(t, []string{"demo/1.0/demo.bin"}, fm.uploadedPaths)
		require.Len(t, meta.Files, 1)
		require.Equal(t, "demo.bin", meta.Files[0].Filename)
		require.Equal(t, int64(7), meta.Size)
		require.Equal(t, "art-uuid", versions.saved.UUID)
		require.Len(t, auditSvc.actions, 1)
	})

	t.Run("new file is merged into an existing version", func(t *testing.T) {
		images := &rpfImageRepo{image: &types.Image{ID: 5}}
		versions := &rpfArtifactRepo{existing: rpfStoredMeta(t, "other.bin")}
		lb, _ := rpfNewBase(rpfDeps{images: images, versions: versions})
		meta := &rpfMeta{}
		_, _, err := lb.UploadFile(ctx, rpfArtifactInfo(), "demo.bin", "1.0", "demo/1.0/demo.bin", nil, meta)
		require.NoError(t, err)
		require.Len(t, meta.Files, 2)
		require.Equal(t, "other.bin", meta.Files[0].Filename)
		require.Equal(t, "demo.bin", meta.Files[1].Filename)
	})

	t.Run("re-uploading a known filename that no longer exists on disk", func(t *testing.T) {
		images := &rpfImageRepo{image: &types.Image{ID: 5}}
		versions := &rpfArtifactRepo{existing: rpfStoredMeta(t, "demo.bin")}
		fm := &rpfFM{headErr: gitnessstore.ErrResourceNotFound}
		lb, _ := rpfNewBase(rpfDeps{fm: fm, images: images, versions: versions})
		meta := &rpfMeta{}
		_, _, err := lb.Upload(ctx, rpfArtifactInfo(), "demo.bin", "1.0", "demo/1.0/demo.bin",
			io.NopCloser(strings.NewReader("x")), meta)
		require.NoError(t, err)
		require.Len(t, meta.Files, 1)
	})

	t.Run("a nil metadata stores an empty document", func(t *testing.T) {
		versions := &rpfArtifactRepo{getErr: gitnessstore.ErrResourceNotFound}
		lb, _ := rpfNewBase(rpfDeps{images: &rpfImageRepo{getErr: gitnessstore.ErrResourceNotFound}, versions: versions})
		_, _, err := lb.Upload(ctx, rpfArtifactInfo(), "demo.bin", "1.0", "p", io.NopCloser(strings.NewReader("x")), nil)
		require.NoError(t, err)
		require.JSONEq(t, "{}", string(versions.saved.Metadata))
	})
}

func TestRpfUploadConflicts(t *testing.T) {
	ctx := rpfAuthedContext()
	existing := func() (*rpfImageRepo, *rpfArtifactRepo) {
		return &rpfImageRepo{image: &types.Image{ID: 5}}, &rpfArtifactRepo{existing: rpfStoredMeta(t, "demo.bin")}
	}

	t.Run("overwrite is allowed with delete permission", func(t *testing.T) {
		images, versions := existing()
		lb, _ := rpfNewBase(rpfDeps{images: images, versions: versions, allowed: true})
		_, sha, err := lb.Upload(ctx, rpfArtifactInfo(), "demo.bin", "1.0", "p", io.NopCloser(strings.NewReader("x")),
			&rpfMeta{})
		require.NoError(t, err)
		require.Equal(t, "sha-new", sha)
	})

	t.Run("overwrite without delete permission is forbidden", func(t *testing.T) {
		images, versions := existing()
		lb, _ := rpfNewBase(rpfDeps{images: images, versions: versions, allowed: false})
		_, _, err := lb.Upload(ctx, rpfArtifactInfo(), "demo.bin", "1.0", "p", io.NopCloser(strings.NewReader("x")),
			&rpfMeta{})
		require.Error(t, err)
		var userErr *usererror.Error
		require.ErrorAs(t, err, &userErr)
		require.Equal(t, http.StatusForbidden, userErr.Status)
	})

	t.Run("lookup failures abort the upload", func(t *testing.T) {
		upload := func(images *rpfImageRepo, versions *rpfArtifactRepo) error {
			lb, _ := rpfNewBase(rpfDeps{images: images, versions: versions, allowed: true})
			_, _, err := lb.Upload(ctx, rpfArtifactInfo(), "demo.bin", "1.0", "p",
				io.NopCloser(strings.NewReader("x")), &rpfMeta{})
			return err
		}
		require.ErrorContains(t, upload(&rpfImageRepo{getErr: errors.New("db down")}, &rpfArtifactRepo{}),
			"failed to fetch the image")
		require.ErrorContains(t, upload(&rpfImageRepo{image: &types.Image{ID: 5}},
			&rpfArtifactRepo{getErr: errors.New("db down")}), "failed to fetch artifact")
		require.ErrorContains(t, upload(&rpfImageRepo{image: &types.Image{ID: 5}},
			&rpfArtifactRepo{existing: &types.Artifact{Metadata: []byte("{")}}), "failed to get metadata")
	})
}

func TestRpfUploadFailures(t *testing.T) {
	ctx := rpfAuthedContext()
	fresh := func() (*rpfImageRepo, *rpfArtifactRepo) {
		return &rpfImageRepo{getErr: gitnessstore.ErrResourceNotFound},
			&rpfArtifactRepo{getErr: gitnessstore.ErrResourceNotFound}
	}
	upload := func(d rpfDeps) error {
		lb, _ := rpfNewBase(d)
		_, _, err := lb.Upload(ctx, rpfArtifactInfo(), "demo.bin", "1.0", "p",
			io.NopCloser(strings.NewReader("x")), &rpfMeta{})
		return err
	}

	images, versions := fresh()
	require.Error(t, upload(rpfDeps{finder: rpfRegistryFinder{err: errors.New("x")}, images: images, versions: versions}))

	images, versions = fresh()
	require.Error(t, upload(rpfDeps{fm: &rpfFM{uploadErr: errors.New("x")}, images: images, versions: versions}))

	images, versions = fresh()
	require.Error(t, upload(rpfDeps{tx: rpfTx{err: errors.New("x")}, images: images, versions: versions}))

	images, versions = fresh()
	images.upsertErr = errors.New("x")
	require.ErrorContains(t, upload(rpfDeps{images: images, versions: versions}), "failed to create image")

	// The artifact lookup inside the transaction fails for a reason other than a missing record.
	boom := &rpfArtifactRepo{getErr: errors.New("db down")}
	lb, _ := rpfNewBase(rpfDeps{images: &rpfImageRepo{getErr: gitnessstore.ErrResourceNotFound}, versions: boom})
	_, err := lb.postUploadArtifact(ctx, rpfArtifactInfo(), &types.Registry{ID: 3}, "1.0", &rpfMeta{}, types.FileInfo{})
	require.ErrorContains(t, err, "failed to fetch artifact")

	// A stored document that cannot be decoded into the metadata type.
	bad := &rpfArtifactRepo{existing: &types.Artifact{Metadata: []byte("{")}}
	lb, _ = rpfNewBase(rpfDeps{images: &rpfImageRepo{getErr: gitnessstore.ErrResourceNotFound}, versions: bad})
	_, err = lb.postUploadArtifact(ctx, rpfArtifactInfo(), &types.Registry{ID: 3}, "1.0", &rpfMeta{}, types.FileInfo{})
	require.ErrorContains(t, err, "failed to update metadata")

	images, versions = fresh()
	versions.upsertErr = errors.New("x")
	require.ErrorContains(t, upload(rpfDeps{images: images, versions: versions}), "failed to create artifact")
}

func TestRpfUploadRawFile(t *testing.T) {
	ctx := rpfAuthedContext()
	body := func() io.ReadCloser { return io.NopCloser(strings.NewReader("x")) }

	t.Run("success", func(t *testing.T) {
		lb, _ := rpfNewBase(rpfDeps{fm: &rpfFM{headErr: gitnessstore.ErrResourceNotFound}})
		headers, sha, err := lb.UploadRawFile(ctx, rpfArtifactInfo(), "dir/a.txt", body(), true)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, headers.Code)
		require.Equal(t, "sha-new", sha)
	})

	t.Run("conflict", func(t *testing.T) {
		lb, _ := rpfNewBase(rpfDeps{})
		_, _, err := lb.UploadRawFile(ctx, rpfArtifactInfo(), "dir/a.txt", body(), true)
		require.Error(t, err)
		var userErr *usererror.Error
		require.ErrorAs(t, err, &userErr)
		require.Equal(t, http.StatusConflict, userErr.Status)

		// Without failOnConflict an existing file is simply replaced.
		_, _, err = lb.UploadRawFile(ctx, rpfArtifactInfo(), "dir/a.txt", body(), false)
		require.NoError(t, err)
	})

	t.Run("existence check failure", func(t *testing.T) {
		lb, _ := rpfNewBase(rpfDeps{fm: &rpfFM{headErr: errors.New("db down")}})
		_, _, err := lb.UploadRawFile(ctx, rpfArtifactInfo(), "dir/a.txt", body(), true)
		require.Error(t, err)
	})

	t.Run("anonymous callers are rejected", func(t *testing.T) {
		lb, _ := rpfNewBase(rpfDeps{fm: &rpfFM{headErr: gitnessstore.ErrResourceNotFound}})
		_, _, err := lb.UploadRawFile(context.Background(), rpfArtifactInfo(), "dir/a.txt", body(), true)
		require.ErrorIs(t, err, usererror.ErrUnauthorized)
	})

	t.Run("storage failure", func(t *testing.T) {
		fm := &rpfFM{headErr: gitnessstore.ErrResourceNotFound, uploadErr: errors.New("disk full")}
		lb, _ := rpfNewBase(rpfDeps{fm: fm})
		_, _, err := lb.UploadRawFile(ctx, rpfArtifactInfo(), "dir/a.txt", body(), true)
		require.Error(t, err)
	})
}

func TestRpfDownloads(t *testing.T) {
	ctx := context.Background()

	lb, _ := rpfNewBase(rpfDeps{})
	headers, _, redirect, err := lb.DownloadRawFile(ctx, rpfArtifactInfo(), "/dir/a.txt")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, "https://redirect.test/file", redirect)
	headers, _, redirect, err = lb.Download(ctx, rpfArtifactInfo(), "1.0", "demo.bin")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, "https://redirect.test/file", redirect)

	failing, _ := rpfNewBase(rpfDeps{fm: &rpfFM{downloadErr: errors.New("x")}})
	_, _, _, err = failing.DownloadRawFile(ctx, rpfArtifactInfo(), "/dir/a.txt")
	require.Error(t, err)
	_, _, _, err = failing.Download(ctx, rpfArtifactInfo(), "1.0", "demo.bin")
	require.Error(t, err)

	noRegistry, _ := rpfNewBase(rpfDeps{finder: rpfRegistryFinder{err: errors.New("x")}})
	_, _, _, err = noRegistry.Download(ctx, rpfArtifactInfo(), "1.0", "demo.bin")
	require.Error(t, err)
}

func TestRpfUpdateFileManagerAndCreateArtifact(t *testing.T) {
	ctx := rpfAuthedContext()
	fileInfo := types.FileInfo{Sha256: "sha-file", Size: 3, Filename: "demo.bin"}
	fresh := func() (*rpfImageRepo, *rpfArtifactRepo) {
		return &rpfImageRepo{getErr: gitnessstore.ErrResourceNotFound},
			&rpfArtifactRepo{getErr: gitnessstore.ErrResourceNotFound}
	}

	t.Run("new artifact", func(t *testing.T) {
		images, versions := fresh()
		fm := &rpfFM{}
		lb, auditSvc := rpfNewBase(rpfDeps{fm: fm, images: images, versions: versions})
		headers, sha, id, existent, err := lb.UpdateFileManagerAndCreateArtifact(ctx, rpfArtifactInfo(), "1.0",
			"demo/1.0/demo.bin", &rpfMeta{}, fileInfo, true)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, headers.Code)
		require.Equal(t, "sha-file", sha)
		require.Equal(t, int64(42), id)
		require.False(t, existent)
		require.Equal(t, []string{"demo/1.0/demo.bin"}, fm.posted)
		require.Len(t, auditSvc.actions, 1)
	})

	conflicting := func() (*rpfImageRepo, *rpfArtifactRepo) {
		return &rpfImageRepo{image: &types.Image{ID: 5}}, &rpfArtifactRepo{existing: rpfStoredMeta(t, "demo.bin")}
	}

	t.Run("conflicting file fails when requested", func(t *testing.T) {
		images, versions := conflicting()
		lb, _ := rpfNewBase(rpfDeps{images: images, versions: versions})
		headers, _, _, existent, err := lb.UpdateFileManagerAndCreateArtifact(ctx, rpfArtifactInfo(), "1.0",
			"p", &rpfMeta{}, fileInfo, true)
		require.Error(t, err)
		require.True(t, existent)
		require.Equal(t, http.StatusConflict, headers.Code)
	})

	t.Run("conflicting file is reused otherwise", func(t *testing.T) {
		images, versions := conflicting()
		lb, _ := rpfNewBase(rpfDeps{images: images, versions: versions})
		headers, sha, id, existent, err := lb.UpdateFileManagerAndCreateArtifact(ctx, rpfArtifactInfo(), "1.0",
			"p", &rpfMeta{}, fileInfo, false)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, headers.Code)
		require.Equal(t, "sha-head", sha)
		require.Zero(t, id)
		require.True(t, existent)
	})

	t.Run("failures", func(t *testing.T) {
		call := func(d rpfDeps) error {
			lb, _ := rpfNewBase(d)
			_, _, _, _, err := lb.UpdateFileManagerAndCreateArtifact(ctx, rpfArtifactInfo(), "1.0", "p",
				&rpfMeta{}, fileInfo, true)
			return err
		}
		images, versions := fresh()
		require.Error(t, call(rpfDeps{images: &rpfImageRepo{getErr: errors.New("db")}, versions: versions}))
		require.Error(t, call(rpfDeps{finder: rpfRegistryFinder{err: errors.New("x")}, images: images, versions: versions}))
		require.Error(t, call(rpfDeps{fm: &rpfFM{postErr: errors.New("x")}, images: images, versions: versions}))
		bad := &rpfArtifactRepo{getErr: gitnessstore.ErrResourceNotFound, upsertErr: errors.New("x")}
		require.Error(t, call(rpfDeps{images: images, versions: bad}))
	})
}

func TestRpfMoveMultipleTempFiles(t *testing.T) {
	ctx := rpfAuthedContext()
	files := []types.FileInfo{
		{Sha256: "sha-a", Filename: "a.bin", Size: 1},
		{Sha256: "sha-b", Filename: "b.bin", Size: 2},
	}
	info := rpfArtifactInfo()

	t.Run("new files are moved and registered", func(t *testing.T) {
		fm := &rpfFM{filePathErr: errors.New("unknown")}
		versions := &rpfArtifactRepo{getErr: gitnessstore.ErrResourceNotFound}
		lb, auditSvc := rpfNewBase(rpfDeps{fm: fm, versions: versions})
		meta := &rpfMeta{}
		err := lb.MoveMultipleTempFilesAndCreateArtifact(ctx, &info, "demo/1.0", meta, &files, "1.0")
		require.NoError(t, err)
		require.Equal(t, []string{"demo/1.0/a.bin", "demo/1.0/b.bin"}, fm.posted)
		require.Len(t, meta.Files, 2)
		require.Equal(t, int64(3), meta.Size)
		require.Len(t, auditSvc.actions, 1)
	})

	t.Run("files that already exist are only linked", func(t *testing.T) {
		fm := &rpfFM{}
		existing := rpfStoredMeta(t, "a.bin")
		lb, _ := rpfNewBase(rpfDeps{fm: fm, versions: &rpfArtifactRepo{existing: existing}})
		err := lb.MoveMultipleTempFilesAndCreateArtifact(ctx, &info, "demo/1.0", &rpfMeta{}, &files, "1.0")
		require.NoError(t, err)
		require.Equal(t, []string{"sha-a", "sha-b"}, fm.saved)
		require.Empty(t, fm.posted)
	})

	t.Run("failures", func(t *testing.T) {
		move := func(d rpfDeps) error {
			lb, _ := rpfNewBase(d)
			return lb.MoveMultipleTempFilesAndCreateArtifact(ctx, &info, "demo/1.0", &rpfMeta{}, &files, "1.0")
		}
		notFound := &rpfArtifactRepo{getErr: gitnessstore.ErrResourceNotFound}
		require.Error(t, move(rpfDeps{fm: &rpfFM{saveNodesErr: errors.New("x")}, versions: notFound}))
		postFails := &rpfFM{filePathErr: errors.New("x"), postErr: errors.New("x")}
		require.Error(t, move(rpfDeps{fm: postFails, versions: notFound}))
		missing := &rpfFM{filePathErr: errors.New("x")}
		require.ErrorContains(t,
			move(rpfDeps{fm: missing, images: &rpfImageRepo{upsertErr: errors.New("x")}, versions: notFound}),
			"failed to create image")
		require.ErrorContains(t,
			move(rpfDeps{fm: missing, versions: &rpfArtifactRepo{getErr: errors.New("db down")}}),
			"failed to fetch artifact")
		require.ErrorContains(t,
			move(rpfDeps{fm: missing, versions: &rpfArtifactRepo{existing: &types.Artifact{Metadata: []byte("{")}}}),
			"failed to update metadata")
		require.ErrorContains(t,
			move(rpfDeps{fm: missing, versions: &rpfArtifactRepo{
				getErr: gitnessstore.ErrResourceNotFound, upsertErr: errors.New("x"),
			}}),
			"failed to create artifact")
		require.Error(t, move(rpfDeps{fm: missing, tx: rpfTx{err: errors.New("x")}, versions: notFound}))
	})
}

func TestRpfExistence(t *testing.T) {
	ctx := context.Background()
	info := rpfPackageInfo()

	lb, _ := rpfNewBase(rpfDeps{})
	require.True(t, lb.Exists(ctx, rpfArtifactInfo(), "demo/1.0/a"))
	headers, err := lb.ExistsE(ctx, info, "a.bin")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, "sha-head", headers.Headers[storage.HeaderContentDigest])
	require.Equal(t, "12", headers.Headers[storage.HeaderContentLength])
	require.Equal(t, "sha256:sha-head", headers.Headers[storage.HeaderEtag])
	exists, err := lb.ExistsByFilePath(ctx, 3, "demo/1.0/a")
	require.NoError(t, err)
	require.True(t, exists)
	exists, sha, err := lb.GetSHA256ByPath(ctx, 3, "demo/1.0/a")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "sha-head", sha)

	missing, _ := rpfNewBase(rpfDeps{fm: &rpfFM{headErr: gitnessstore.ErrResourceNotFound}})
	require.False(t, missing.Exists(ctx, rpfArtifactInfo(), "demo/1.0/a"))
	headers, err = missing.ExistsE(ctx, info, "a.bin")
	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, headers.Code)
	exists, err = missing.ExistsByFilePath(ctx, 3, "demo/1.0/a")
	require.NoError(t, err)
	require.False(t, exists)
	_, _, err = missing.GetSHA256ByPath(ctx, 3, "demo/1.0/a")
	require.Error(t, err)

	broken, _ := rpfNewBase(rpfDeps{fm: &rpfFM{headErr: errors.New("db down")}})
	_, err = broken.ExistsByFilePath(ctx, 3, "demo/1.0/a")
	require.ErrorContains(t, err, "db down")
}

func TestRpfDeleteFile(t *testing.T) {
	ctx := context.Background()
	info := rpfPackageInfo()

	fm := &rpfFM{}
	lb, _ := rpfNewBase(rpfDeps{fm: fm})
	headers, err := lb.DeleteFile(ctx, info, "a.bin")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, headers.Code)
	require.Equal(t, []string{"demo/1.0/a.bin"}, fm.deleted)

	missing, _ := rpfNewBase(rpfDeps{fm: &rpfFM{headErr: gitnessstore.ErrResourceNotFound}})
	headers, err = missing.DeleteFile(ctx, info, "a.bin")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, headers.Code)

	failing, _ := rpfNewBase(rpfDeps{fm: &rpfFM{deleteErr: errors.New("x")}})
	_, err = failing.DeleteFile(ctx, info, "a.bin")
	require.Error(t, err)
}

func TestRpfCheckIfVersionExists(t *testing.T) {
	ctx := context.Background()
	lb, _ := rpfNewBase(rpfDeps{versions: &rpfArtifactRepo{versionOK: true}})
	exists, err := lb.CheckIfVersionExists(ctx, rpfPackageInfo())
	require.NoError(t, err)
	require.True(t, exists)

	lb, _ = rpfNewBase(rpfDeps{})
	exists, err = lb.CheckIfVersionExists(ctx, rpfPackageInfo())
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	require.False(t, exists)
}

func TestRpfDeletePackageAndVersion(t *testing.T) {
	ctx := context.Background()
	info := rpfPackageInfo()
	boom := errors.New("boom")

	lb, _ := rpfNewBase(rpfDeps{})
	require.NoError(t, lb.DeletePackage(ctx, info))
	require.NoError(t, lb.DeleteVersion(ctx, info))

	for name, d := range map[string]rpfDeps{
		"nodes":  {nodes: rpfNodesRepo{err: boom}},
		"images": {images: &rpfImageRepo{deleteErr: boom}},
		"tx":     {tx: rpfTx{err: boom}},
	} {
		t.Run("package/"+name, func(t *testing.T) {
			failing, _ := rpfNewBase(d)
			require.ErrorIs(t, failing.DeletePackage(ctx, info), boom)
		})
	}
	failing, _ := rpfNewBase(rpfDeps{versions: &rpfArtifactRepo{deleteErr: boom}})
	require.ErrorIs(t, failing.DeletePackage(ctx, info), boom)
	require.ErrorIs(t, failing.DeleteVersion(ctx, info), boom)
	failing, _ = rpfNewBase(rpfDeps{nodes: rpfNodesRepo{err: boom}})
	require.ErrorIs(t, failing.DeleteVersion(ctx, info), boom)
}

func TestRpfAuditPush(t *testing.T) {
	lb, auditSvc := rpfNewBase(rpfDeps{})
	lb.AuditPush(rpfAuthedContext(), rpfArtifactInfo(), "1.0", "img", "art")
	require.Len(t, auditSvc.actions, 1)
	require.Equal(t, audit.ActionUploaded, auditSvc.actions[0])
}
