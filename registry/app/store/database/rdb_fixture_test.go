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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gtypes "github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

const rdbRootSpaceID int64 = 100

// rdbFakeDownloadCounts is a DownloadCountFinder returning fixed values.
type rdbFakeDownloadCounts struct {
	byRegistry map[int64]int64
}

func (f *rdbFakeDownloadCounts) FindByRegistryIDs(_ context.Context, ids []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	for _, id := range ids {
		if v, ok := f.byRegistry[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}

func (f *rdbFakeDownloadCounts) FindByImageID(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

func (f *rdbFakeDownloadCounts) FindByImageIDs(_ context.Context, _ []int64) (map[int64]int64, error) {
	return map[int64]int64{}, nil
}

func (f *rdbFakeDownloadCounts) FindByArtifactID(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

func (f *rdbFakeDownloadCounts) FindByArtifactIDs(_ context.Context, _ []int64) (map[int64]int64, error) {
	return map[int64]int64{}, nil
}

func (f *rdbFakeDownloadCounts) FindByManifests(_ context.Context, _ []string, _ int64) (map[string]int64, error) {
	return map[string]int64{}, nil
}

var _ store.DownloadCountFinder = (*rdbFakeDownloadCounts)(nil)

// rdbEnv bundles a migrated database and the DAOs under test.
type rdbEnv struct {
	db         *sqlx.DB
	ctx        context.Context
	dcf        *rdbFakeDownloadCounts
	mediaTypes store.MediaTypesRepository
	registries store.RegistryRepository
	blobs      store.BlobRepository
	images     store.ImageRepository
	artifacts  store.ArtifactRepository
	manifests  store.ManifestRepository
	tags       store.TagRepository
	seq        int
}

func rdbCtx() context.Context {
	return request.WithAuthSession(context.Background(), &auth.Session{
		Principal: gtypes.Principal{ID: 7},
	})
}

func rdbNewEnv(t *testing.T) *rdbEnv {
	t.Helper()
	db := rdbSetupDB(t)
	dcf := &rdbFakeDownloadCounts{byRegistry: map[int64]int64{}}
	mt := NewMediaTypesDao(db)
	return &rdbEnv{
		db:         db,
		ctx:        rdbCtx(),
		dcf:        dcf,
		mediaTypes: mt,
		registries: NewRegistryDao(db, mt, dcf),
		blobs:      NewBlobDao(db, mt),
		images:     NewImageDao(db),
		artifacts:  NewArtifactDao(db, dcf),
		manifests:  NewManifestDao(db, mt),
		tags:       NewTagDao(db, dcf),
	}
}

func (e *rdbEnv) next() int {
	e.seq++
	return e.seq
}

// rdbDigest returns a valid sha256 digest derived from the given seed.
func rdbDigest(seed int) digest.Digest {
	return digest.FromString(fmt.Sprintf("rdb-seed-%d", seed))
}

func (e *rdbEnv) newRegistry(
	t *testing.T, parentID int64, name string, regType artifact.RegistryType, pt artifact.PackageType,
) *types.Registry {
	t.Helper()
	reg := &types.Registry{
		Name:         name,
		ParentID:     parentID,
		RootParentID: rdbRootSpaceID,
		Type:         regType,
		PackageType:  pt,
	}
	id, err := e.registries.Create(e.ctx, reg)
	require.NoError(t, err)
	reg.ID = id
	return reg
}

func (e *rdbEnv) newDockerRegistry(t *testing.T, name string) *types.Registry {
	t.Helper()
	return e.newRegistry(t, rdbRootSpaceID, name, artifact.RegistryTypeVIRTUAL, artifact.PackageTypeDOCKER)
}

func (e *rdbEnv) newImage(t *testing.T, regID int64, name string) *types.Image {
	t.Helper()
	img := &types.Image{Name: name, RegistryID: regID, Enabled: true}
	require.NoError(t, e.images.CreateOrUpdate(e.ctx, img))
	require.NotZero(t, img.ID)
	return img
}

func (e *rdbEnv) newArtifact(t *testing.T, imageID int64, version string) *types.Artifact {
	t.Helper()
	a := &types.Artifact{ImageID: imageID, Version: version, Metadata: []byte(`{"k":"v"}`)}
	id, err := e.artifacts.CreateOrUpdate(e.ctx, a)
	require.NoError(t, err)
	a.ID = id
	return a
}

func (e *rdbEnv) newBlob(t *testing.T, size int64) *types.Blob {
	t.Helper()
	b := &types.Blob{
		RootParentID: rdbRootSpaceID,
		MediaType:    "application/octet-stream",
		Digest:       rdbDigest(e.next()),
		Size:         size,
	}
	created, isNew, err := e.blobs.CreateOrFind(e.ctx, b)
	require.NoError(t, err)
	require.True(t, isNew)
	return created
}

func (e *rdbEnv) newManifest(t *testing.T, regID int64, image string) *types.Manifest {
	t.Helper()
	m := &types.Manifest{
		RegistryID:    regID,
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		ImageName:     image,
		Digest:        rdbDigest(e.next()),
		Payload:       types.Payload(`{"schemaVersion":2}`),
		TotalSize:     10,
	}
	require.NoError(t, e.manifests.Create(e.ctx, m))
	require.NotZero(t, m.ID)
	return m
}

func (e *rdbEnv) newTag(t *testing.T, regID int64, image, name string, manifestID int64) {
	t.Helper()
	tag := &types.Tag{Name: name, ImageName: image, RegistryID: regID, ManifestID: manifestID}
	require.NoError(t, e.tags.CreateOrUpdate(e.ctx, tag))
}

// exec runs a statement directly against the test database.
func (e *rdbEnv) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := e.db.ExecContext(e.ctx, query, args...)
	require.NoError(t, err)
}

func (e *rdbEnv) softDelete(t *testing.T, table, col string, id int64) {
	t.Helper()
	e.exec(t, fmt.Sprintf("UPDATE %s SET %s_deleted_at = ? WHERE %s_id = ?", table, col, col),
		time.Now().UnixMilli(), id)
}

// rdbBackground returns a context without an auth session.
func rdbBackground() context.Context {
	return context.Background()
}
