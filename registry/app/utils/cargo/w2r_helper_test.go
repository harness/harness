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

package cargo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	appcache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/registry/app/metadata"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
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

type w2rArtifactRepo struct {
	store.ArtifactRepository
	batches [][]types.ArtifactMetadata
	err     error
	calls   int
}

func (r *w2rArtifactRepo) GetArtifactsByRepoAndImageBatch(
	context.Context, int64, string, int, int64,
) (*[]types.ArtifactMetadata, error) {
	if r.err != nil {
		return nil, r.err
	}
	batch := r.batches[r.calls]
	r.calls++
	return &batch, nil
}

type w2rFileManager struct {
	filemanager.FileManager
	uploadedPath string
	uploaded     string
	err          error
}

func (f *w2rFileManager) UploadFile(
	_ context.Context, filePath string, _ int64, _ int64, _ string, _ multipart.File,
	reader io.Reader, _ int64,
) (types.FileInfo, error) {
	if f.err != nil {
		return types.FileInfo{}, f.err
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return types.FileInfo{}, err
	}
	f.uploadedPath = filePath
	f.uploaded = string(data)
	return types.FileInfo{}, nil
}

func w2rMeta(t *testing.T, md cargometadata.VersionMetadataDB) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(md)
	require.NoError(t, err)
	return raw
}

func TestW2rIndexFilePath(t *testing.T) {
	h := &registryHelper{}
	tests := map[string]string{
		"":      "",
		"a":     "/index/1/a",
		"ab":    "/index/2/ab",
		"abc":   "/index/3/a/abc",
		"serde": "/index/se/rd/serde",
	}
	for name, want := range tests {
		assert.Equal(t, want, h.GetIndexFilePathFromImageName(name), name)
	}
}

func TestW2rUpdatePackageIndex(t *testing.T) {
	ctx := context.Background()
	withFile := cargometadata.VersionMetadataDB{
		VersionMetadata: cargometadata.VersionMetadata{
			Dependencies: []cargometadata.VersionDependency{{
				Dependency:      cargometadata.Dependency{Name: "dep", Kind: "normal"},
				VersionRequired: "^1",
			}},
		},
	}
	withFile.SetFiles([]metadata.File{{Sha256: "abc", Filename: "serde-1.0.crate"}})
	noFiles := cargometadata.VersionMetadataDB{}

	t.Run("uploads", func(t *testing.T) {
		repo := &w2rArtifactRepo{batches: [][]types.ArtifactMetadata{{
			{ID: 1, Name: "serde", Version: "1.0.0", Metadata: w2rMeta(t, withFile)},
			{ID: 2, Name: "serde", Version: "1.0.1", Metadata: w2rMeta(t, noFiles)},
		}}}
		fm := &w2rFileManager{}
		h := NewRegistryHelper(fm, repo, w2rSpaceFinder(nil))
		require.NoError(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "serde"))
		assert.Equal(t, "/index/se/rd/serde", fm.uploadedPath)
		assert.Contains(t, fm.uploaded, `"cksum":"abc"`)
		assert.Contains(t, fm.uploaded, `"req":"^1"`)
	})

	t.Run("pages through full batches", func(t *testing.T) {
		full := make([]types.ArtifactMetadata, 50)
		for i := range full {
			full[i] = types.ArtifactMetadata{ID: int64(i + 1), Name: "x", Metadata: w2rMeta(t, noFiles)}
		}
		repo := &w2rArtifactRepo{batches: [][]types.ArtifactMetadata{full, {}}}
		h := NewRegistryHelper(&w2rFileManager{}, repo, w2rSpaceFinder(nil))
		require.NoError(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "x"))
		assert.Equal(t, 2, repo.calls)
	})

	t.Run("space lookup error", func(t *testing.T) {
		h := NewRegistryHelper(&w2rFileManager{}, &w2rArtifactRepo{}, w2rSpaceFinder(errors.New("boom")))
		require.ErrorContains(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "x"), "root space")
	})

	t.Run("artifact error", func(t *testing.T) {
		repo := &w2rArtifactRepo{err: errors.New("db")}
		h := NewRegistryHelper(&w2rFileManager{}, repo, w2rSpaceFinder(nil))
		require.ErrorContains(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "x"), "regenerate")
	})

	t.Run("bad metadata", func(t *testing.T) {
		repo := &w2rArtifactRepo{batches: [][]types.ArtifactMetadata{{{ID: 1, Metadata: json.RawMessage("{")}}}}
		h := NewRegistryHelper(&w2rFileManager{}, repo, w2rSpaceFinder(nil))
		require.ErrorContains(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "x"), "unmarshal")
	})

	t.Run("upload error", func(t *testing.T) {
		repo := &w2rArtifactRepo{batches: [][]types.ArtifactMetadata{{}}}
		h := NewRegistryHelper(&w2rFileManager{err: errors.New("disk")}, repo, w2rSpaceFinder(nil))
		require.ErrorContains(t, h.UpdatePackageIndex(ctx, 1, 2, 3, "x"), "upload")
	})
}

func TestW2rMapDependencies(t *testing.T) {
	h := &registryHelper{}
	assert.Empty(t, h.mapVersionDependenciesToIndexDependencies(nil))
	out := h.mapVersionDependenciesToIndexDependencies([]cargometadata.VersionDependency{{
		Dependency:      cargometadata.Dependency{Name: "d", Kind: "dev", IsOptional: true},
		VersionRequired: "1",
	}})
	require.Len(t, out, 1)
	assert.Equal(t, "d", out[0].Name)
	assert.Equal(t, "1", out[0].VersionRequired)
	assert.True(t, out[0].IsOptional)
}
