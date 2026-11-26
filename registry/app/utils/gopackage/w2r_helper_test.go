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

package gopackage

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	appcache "github.com/harness/gitness/app/store/cache"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	gopackagemetadata "github.com/harness/gitness/registry/app/metadata/gopackage"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	refcache2 "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"
	gtypes "github.com/harness/gitness/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type w2rMapCache[K comparable, V any] struct {
	get func(key K) (V, error)
}

func (c w2rMapCache[K, V]) Stats() (int64, int64)                   { return 0, 0 }
func (c w2rMapCache[K, V]) Get(_ context.Context, key K) (V, error) { return c.get(key) }
func (c w2rMapCache[K, V]) Evict(context.Context, K)                {}

func w2rSpaceFinder(err error) refcache.SpaceFinder {
	idCache := w2rMapCache[int64, *gtypes.SpaceCore]{get: func(id int64) (*gtypes.SpaceCore, error) {
		if err != nil {
			return nil, err
		}
		return &gtypes.SpaceCore{ID: id, Identifier: "root"}, nil
	}}
	pathCache := w2rMapCache[string, *gtypes.SpacePath]{get: func(string) (*gtypes.SpacePath, error) {
		return nil, gitnessstore.ErrResourceNotFound
	}}
	ciCache := w2rMapCache[string, int64]{get: func(string) (int64, error) { return 0, gitnessstore.ErrResourceNotFound }}
	return refcache.NewSpaceFinder(idCache, pathCache, ciCache, appcache.Evictor[*gtypes.SpaceCore]{})
}

type w2rDriver struct {
	storagedriver.StorageDriver
	content []byte
}

func (d *w2rDriver) Reader(context.Context, string, int64) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(d.content)), nil
}

type w2rRegFinder struct {
	refcache2.RegistryFinder
	err error
}

func (f *w2rRegFinder) FindByID(context.Context, int64) (*types.Registry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.Registry{ID: 3, Name: "reg"}, nil
}

type w2rArtifactRepo struct {
	store.ArtifactRepository
	batches   [][]types.ArtifactMetadata
	batchErr  error
	artifact  *types.Artifact
	getErr    error
	updateErr error
	updated   json.RawMessage
	calls     int
}

func (r *w2rArtifactRepo) GetArtifactsByRepoAndImageBatch(
	context.Context, int64, string, int, int64,
) (*[]types.ArtifactMetadata, error) {
	if r.batchErr != nil {
		return nil, r.batchErr
	}
	batch := r.batches[r.calls]
	r.calls++
	return &batch, nil
}

func (r *w2rArtifactRepo) GetByRegistryImageAndVersion(
	context.Context, int64, string, string,
) (*types.Artifact, error) {
	return r.artifact, r.getErr
}

func (r *w2rArtifactRepo) UpdateArtifactMetadata(_ context.Context, md json.RawMessage, _ int64) error {
	r.updated = md
	return r.updateErr
}

type w2rFileManager struct {
	filemanager.FileManager
	files        map[string][]byte
	uploadedPath string
	uploaded     string
	uploadErr    error
}

func (f *w2rFileManager) UploadFile(
	_ context.Context, filePath string, _ int64, _ int64, _ string, _ multipart.File,
	reader io.Reader, _ int64,
) (types.FileInfo, error) {
	if f.uploadErr != nil {
		return types.FileInfo{}, f.uploadErr
	}
	data, _ := io.ReadAll(reader)
	f.uploadedPath = filePath
	f.uploaded = string(data)
	return types.FileInfo{}, nil
}

func (f *w2rFileManager) DownloadFileByPath(
	ctx context.Context, filePath string, _ int64, _ string, _ string, _ bool,
) (*storage.FileReader, int64, string, error) {
	for suffix, content := range f.files {
		if strings.HasSuffix(filePath, suffix) {
			fr, err := storage.NewFileReader(ctx, &w2rDriver{content: content}, filePath, int64(len(content)))
			return fr, int64(len(content)), "", err
		}
	}
	return nil, 0, "", errors.New("missing " + filePath)
}

func w2rZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("mod@v1/README.md")
	require.NoError(t, err)
	_, err = w.Write([]byte("# readme"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func w2rFiles(t *testing.T) map[string][]byte {
	t.Helper()
	return map[string][]byte{
		".info": []byte(`{"Version":"v1.0.0","Time":"2024-01-01T00:00:00Z","Origin":{"VCS":"git"}}`),
		".mod":  []byte("module example.com/m\n\nrequire example.com/dep v1.2.3\n"),
		".zip":  w2rZip(t),
	}
}

func w2rArtifact(t *testing.T) *types.Artifact {
	t.Helper()
	raw, err := json.Marshal(gopackagemetadata.VersionMetadataDB{})
	require.NoError(t, err)
	return &types.Artifact{ID: 9, Metadata: raw}
}

func TestW2rUpdatePackageIndex(t *testing.T) {
	ctx := context.Background()

	t.Run("uploads versions", func(t *testing.T) {
		repo := &w2rArtifactRepo{batches: [][]types.ArtifactMetadata{
			{{ID: 1, Version: "v1"}, {ID: 2, Version: "v2"}},
		}}
		fm := &w2rFileManager{}
		h := NewRegistryHelper(fm, repo, w2rSpaceFinder(nil), &w2rRegFinder{})
		require.NoError(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "mod"))
		assert.Equal(t, "v1\nv2", fm.uploaded)
	})

	t.Run("full batch pages again", func(t *testing.T) {
		full := make([]types.ArtifactMetadata, 50)
		for i := range full {
			full[i] = types.ArtifactMetadata{ID: int64(i + 1), Version: "v"}
		}
		repo := &w2rArtifactRepo{batches: [][]types.ArtifactMetadata{full, {}}}
		h := NewRegistryHelper(&w2rFileManager{}, repo, w2rSpaceFinder(nil), &w2rRegFinder{})
		require.NoError(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "mod"))
		assert.Equal(t, 2, repo.calls)
	})

	t.Run("not found ends the loop", func(t *testing.T) {
		repo := &w2rArtifactRepo{batchErr: gitnessstore.ErrResourceNotFound}
		h := NewRegistryHelper(&w2rFileManager{}, repo, w2rSpaceFinder(nil), &w2rRegFinder{})
		require.NoError(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "mod"))
	})

	t.Run("errors", func(t *testing.T) {
		h := NewRegistryHelper(&w2rFileManager{}, &w2rArtifactRepo{}, w2rSpaceFinder(errors.New("x")), &w2rRegFinder{})
		require.ErrorContains(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "m"), "root space")

		repo := &w2rArtifactRepo{batchErr: errors.New("db")}
		h = NewRegistryHelper(&w2rFileManager{}, repo, w2rSpaceFinder(nil), &w2rRegFinder{})
		require.ErrorContains(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "m"), "regenerate")

		repo = &w2rArtifactRepo{batches: [][]types.ArtifactMetadata{{}}}
		fm := &w2rFileManager{uploadErr: errors.New("disk")}
		h = NewRegistryHelper(fm, repo, w2rSpaceFinder(nil), &w2rRegFinder{})
		require.ErrorContains(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "m"), "upload")
	})
}

func TestW2rUpdatePackageMetadata(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := &w2rArtifactRepo{artifact: w2rArtifact(t)}
		fm := &w2rFileManager{files: w2rFiles(t)}
		h := NewRegistryHelper(fm, repo, w2rSpaceFinder(nil), &w2rRegFinder{})
		require.NoError(t, h.UpdatePackageMetadata(ctx, 2, 3, "example.com/m", "v1.0.0"))
		var got gopackagemetadata.VersionMetadataDB
		require.NoError(t, json.Unmarshal(repo.updated, &got))
		assert.Equal(t, "example.com/m", got.Name)
		assert.Equal(t, "2024-01-01T00:00:00Z", got.Time)
		require.Len(t, got.Dependencies, 1)
		assert.Equal(t, "example.com/dep", got.Dependencies[0].Name)
		assert.Equal(t, "# readme", got.Readme)
	})

	t.Run("lookup errors", func(t *testing.T) {
		fm := &w2rFileManager{files: w2rFiles(t)}

		h := NewRegistryHelper(fm, &w2rArtifactRepo{}, w2rSpaceFinder(errors.New("x")), &w2rRegFinder{})
		require.ErrorContains(t, h.UpdatePackageMetadata(ctx, 2, 3, "m", "v1.0.0"), "root space")

		h = NewRegistryHelper(fm, &w2rArtifactRepo{}, w2rSpaceFinder(nil), &w2rRegFinder{err: errors.New("x")})
		require.ErrorContains(t, h.UpdatePackageMetadata(ctx, 2, 3, "m", "v1.0.0"), "registry by ID")

		repo := &w2rArtifactRepo{getErr: errors.New("x")}
		h = NewRegistryHelper(fm, repo, w2rSpaceFinder(nil), &w2rRegFinder{})
		require.ErrorContains(t, h.UpdatePackageMetadata(ctx, 2, 3, "m", "v1.0.0"), "get artifact")

		repo = &w2rArtifactRepo{artifact: &types.Artifact{Metadata: json.RawMessage("{")}}
		h = NewRegistryHelper(fm, repo, w2rSpaceFinder(nil), &w2rRegFinder{})
		require.ErrorContains(t, h.UpdatePackageMetadata(ctx, 2, 3, "m", "v1.0.0"), "convert")
	})

	t.Run("update error", func(t *testing.T) {
		repo := &w2rArtifactRepo{artifact: w2rArtifact(t), updateErr: errors.New("db")}
		fm := &w2rFileManager{files: w2rFiles(t)}
		h := NewRegistryHelper(fm, repo, w2rSpaceFinder(nil), &w2rRegFinder{})
		require.ErrorContains(t, h.UpdatePackageMetadata(ctx, 2, 3, "m", "v1.0.0"), "create or update")
	})

	t.Run("missing or invalid files", func(t *testing.T) {
		for _, drop := range []string{".info", ".mod", ".zip"} {
			files := w2rFiles(t)
			delete(files, drop)
			repo := &w2rArtifactRepo{artifact: w2rArtifact(t)}
			h := NewRegistryHelper(&w2rFileManager{files: files}, repo, w2rSpaceFinder(nil), &w2rRegFinder{})
			require.Error(t, h.UpdatePackageMetadata(ctx, 2, 3, "m", "v1.0.0"), drop)
		}

		for _, bad := range []string{".info", ".mod"} {
			files := w2rFiles(t)
			files[bad] = []byte("not valid")
			repo := &w2rArtifactRepo{artifact: w2rArtifact(t)}
			h := NewRegistryHelper(&w2rFileManager{files: files}, repo, w2rSpaceFinder(nil), &w2rRegFinder{})
			require.Error(t, h.UpdatePackageMetadata(ctx, 2, 3, "m", "v1.0.0"), bad)
		}
	})
}
