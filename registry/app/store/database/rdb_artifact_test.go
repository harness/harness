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

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	gstore "github.com/harness/gitness/store"

	"github.com/stretchr/testify/require"
)

func TestRdbArtifact_CreateAndGet(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newRegistry(t, rdbRootSpaceID, "maven", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	img := e.newImage(t, reg.ID, "lib")

	a := e.newArtifact(t, img.ID, "1.0.0")
	require.NotZero(t, a.ID)
	require.NotEmpty(t, a.UUID)

	got, err := e.artifacts.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, "1.0.0", got.Version)
	require.JSONEq(t, `{"k":"v"}`, string(got.Metadata))
	require.Equal(t, int64(7), got.CreatedBy)

	byUUID, err := e.artifacts.GetByUUID(ctx, a.UUID)
	require.NoError(t, err)
	require.Equal(t, a.ID, byUUID.ID)

	byName, err := e.artifacts.GetByName(ctx, img.ID, "1.0.0")
	require.NoError(t, err)
	require.Equal(t, a.ID, byName.ID)

	byRegistry, err := e.artifacts.GetByRegistryImageAndVersion(ctx, reg.ID, "lib", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, a.ID, byRegistry.ID)

	// Upserting the same version replaces the metadata and keeps the id.
	again := &types.Artifact{ImageID: img.ID, Version: "1.0.0", Metadata: []byte(`{"k":"w"}`)}
	id, err := e.artifacts.CreateOrUpdate(ctx, again)
	require.NoError(t, err)
	require.Equal(t, a.ID, id)
	got, err = e.artifacts.Get(ctx, a.ID)
	require.NoError(t, err)
	require.JSONEq(t, `{"k":"w"}`, string(got.Metadata))

	// A nil metadata payload is stored as JSON null.
	nilMeta := &types.Artifact{ImageID: img.ID, Version: "2.0.0"}
	_, err = e.artifacts.CreateOrUpdate(ctx, nilMeta)
	require.NoError(t, err)

	_, err = e.artifacts.CreateOrUpdate(ctx, &types.Artifact{ImageID: img.ID})
	require.Error(t, err)

	_, err = e.artifacts.Get(ctx, 9999)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.artifacts.GetByUUID(ctx, "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.artifacts.GetByName(ctx, img.ID, "9.9.9")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.artifacts.GetByRegistryImageAndVersion(ctx, reg.ID, "lib", "9.9.9")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	cnt, err := e.artifacts.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt)

	byRepoImage, err := e.artifacts.GetByRegistryIDAndImage(ctx, reg.ID, "lib")
	require.NoError(t, err)
	require.Len(t, *byRepoImage, 2)
}

func TestRdbArtifact_TypedAndLatest(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newRegistry(t, rdbRootSpaceID, "hf", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeHUGGINGFACE)
	model := artifact.ArtifactTypeModel

	typed := &types.Image{Name: "bert", RegistryID: reg.ID, ArtifactType: &model, Enabled: true}
	require.NoError(t, e.images.CreateOrUpdate(ctx, typed))
	a := e.newArtifact(t, typed.ID, "main")

	got, err := e.artifacts.GetByRegistryImageVersionAndArtifactType(ctx, reg.ID, "bert", "main", "model")
	require.NoError(t, err)
	require.Equal(t, a.ID, got.ID)
	_, err = e.artifacts.GetByRegistryImageVersionAndArtifactType(ctx, reg.ID, "bert", "main", "dataset")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	latest, err := e.artifacts.GetLatestByImageID(ctx, typed.ID)
	require.NoError(t, err)
	require.NotNil(t, latest)
	require.Equal(t, a.ID, latest.ID)

	none, err := e.artifacts.GetLatestByImageID(ctx, 9999)
	require.NoError(t, err)
	require.Nil(t, none)
}

func TestRdbArtifact_SoftDelete(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newRegistry(t, rdbRootSpaceID, "maven", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	img := e.newImage(t, reg.ID, "lib")
	live := e.newArtifact(t, img.ID, "1.0")
	gone := e.newArtifact(t, img.ID, "2.0")
	e.softDelete(t, "artifacts", "artifact", gone.ID)

	_, err := e.artifacts.GetByName(ctx, img.ID, "2.0")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.artifacts.GetByName(ctx, img.ID, "2.0", types.WithOnlyDeleted())
	require.NoError(t, err)
	got, err := e.artifacts.GetByName(ctx, img.ID, "2.0", types.WithAllDeleted())
	require.NoError(t, err)
	require.NotNil(t, got.DeletedAt)
	_, err = e.artifacts.GetByName(ctx, img.ID, "1.0", types.WithOnlyDeleted())
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	cnt, err := e.artifacts.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)

	latest, err := e.artifacts.GetLatestByImageID(ctx, img.ID)
	require.NoError(t, err)
	require.Equal(t, live.ID, latest.ID)

	// Re-creating a soft-deleted version is rejected.
	_, err = e.artifacts.CreateOrUpdate(ctx, &types.Artifact{ImageID: img.ID, Version: "2.0"})
	require.Error(t, err)

	md, err := e.artifacts.GetArtifactMetadata(ctx, reg.ParentID, reg.Name, "lib", "2.0", nil)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	require.Nil(t, md)
	md, err = e.artifacts.GetArtifactMetadata(ctx, reg.ParentID, reg.Name, "lib", "2.0", nil, types.WithOnlyDeleted())
	require.NoError(t, err)
	require.NotNil(t, md.DeletedAt)
	md, err = e.artifacts.GetArtifactMetadata(ctx, reg.ParentID, reg.Name, "lib", "2.0", nil, types.WithAllDeleted())
	require.NoError(t, err)
	require.Equal(t, gone.ID, md.ID)
}

func TestRdbArtifact_UpdateMetadataDuplicateAndDelete(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	src := e.newRegistry(t, rdbRootSpaceID, "src", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	dst := e.newRegistry(t, rdbRootSpaceID, "dst", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	srcImg := e.newImage(t, src.ID, "lib")
	dstImg := e.newImage(t, dst.ID, "lib")
	a := e.newArtifact(t, srcImg.ID, "1.0")
	e.newArtifact(t, srcImg.ID, "2.0")

	require.NoError(t, e.artifacts.UpdateArtifactMetadata(ctx, []byte(`{"size":"12"}`), a.ID))
	got, err := e.artifacts.Get(ctx, a.ID)
	require.NoError(t, err)
	require.JSONEq(t, `{"size":"12"}`, string(got.Metadata))
	require.ErrorIs(t, e.artifacts.UpdateArtifactMetadata(ctx, []byte(`{}`), 9999), gstore.ErrResourceNotFound)

	// Without an auth session the principal defaults to zero.
	require.NoError(t, e.artifacts.UpdateArtifactMetadata(rdbBackground(), []byte(`{"size":"13"}`), a.ID))

	dup, err := e.artifacts.DuplicateArtifact(ctx, got, dstImg.ID)
	require.NoError(t, err)
	require.Equal(t, dstImg.ID, dup.ImageID)
	require.NotEqual(t, a.ID, dup.ID)
	require.JSONEq(t, `{"size":"12"}`, string(dup.Metadata))

	require.NoError(t, e.artifacts.DeleteByVersionAndImageName(ctx, "lib", "1.0", src.ID))
	_, err = e.artifacts.Get(ctx, a.ID)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	require.NoError(t, e.artifacts.DeleteByImageNameAndRegistryID(ctx, src.ID, "lib"))
	remaining, err := e.artifacts.GetByRegistryIDAndImage(ctx, src.ID, "lib")
	require.NoError(t, err)
	require.Empty(t, *remaining)
	// The other registry is untouched.
	remaining, err = e.artifacts.GetByRegistryIDAndImage(ctx, dst.ID, "lib")
	require.NoError(t, err)
	require.Len(t, *remaining, 1)
}

// rdbArtifactListFixture creates two registries with several artifacts and returns them.
func rdbArtifactListFixture(t *testing.T, e *rdbEnv) *types.Registry {
	t.Helper()
	maven := e.newRegistry(t, 10, "maven-repo", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	npm := e.newRegistry(t, 10, "npm-repo", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeNPM)

	core := e.newImage(t, maven.ID, "core-lib")
	core.Labels = []string{"stable", "java"}
	require.NoError(t, e.images.Update(e.ctx, core))
	util := e.newImage(t, maven.ID, "util-lib")
	util.Labels = []string{"java"}
	require.NoError(t, e.images.Update(e.ctx, util))
	web := &types.Image{Name: "web-app", RegistryID: npm.ID, Enabled: true}
	require.NoError(t, e.images.CreateOrUpdate(e.ctx, web))

	e.newArtifact(t, core.ID, "1.0")
	newer := e.newArtifact(t, core.ID, "2.0")
	// Make the ordering by update time deterministic.
	e.exec(t, "UPDATE artifacts SET artifact_updated_at = artifact_updated_at + 1000 WHERE artifact_id = ?", newer.ID)
	e.newArtifact(t, util.ID, "0.1")
	e.newArtifact(t, web.ID, "5.0")
	return maven
}

func TestRdbArtifact_ListByParent(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	rdbArtifactListFixture(t, e)

	all := []string{}
	tests := []struct {
		name     string
		repos    []string
		pkgTypes []string
		search   string
		latest   bool
		wantLen  int
	}{
		{"all versions", all, nil, "", false, 4},
		{"repo filter", []string{"npm-repo"}, nil, "", false, 1},
		{"package type filter", all, []string{"MAVEN"}, "", false, 3},
		{"search", all, nil, "util", false, 1},
		{"no match", all, nil, "zzz", false, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repos := tc.repos
			res, err := e.artifacts.GetAllArtifactsByParentID(
				ctx, 10, &repos, "image_name", "ASC", 50, 0, tc.search, tc.latest, tc.pkgTypes)
			require.NoError(t, err)
			require.Len(t, *res, tc.wantLen)

			cnt, err := e.artifacts.CountAllArtifactsByParentID(ctx, 10, &repos, tc.search, tc.latest, tc.pkgTypes)
			require.NoError(t, err)
			require.Equal(t, int64(tc.wantLen), cnt)
		})
	}

	repos := []string{}
	page, err := e.artifacts.GetAllArtifactsByParentID(ctx, 10, &repos, "image_name", "ASC", 2, 1, "", false, nil)
	require.NoError(t, err)
	require.Len(t, *page, 2)

	sorted, err := e.artifacts.GetAllArtifactsByParentID(ctx, 10, &repos, "download_count", "DESC", 50, 0, "", false, nil)
	require.NoError(t, err)
	require.Len(t, *sorted, 4)

	// The latest-version count query works; the list variant is not exercised (see PR observations).
	latestCnt, err := e.artifacts.CountAllArtifactsByParentID(ctx, 10, &repos, "", true, nil)
	require.NoError(t, err)
	require.Equal(t, int64(3), latestCnt)
}

func TestRdbArtifact_ListByRepo(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	maven := rdbArtifactListFixture(t, e)

	res, err := e.artifacts.GetArtifactsByRepo(ctx, 10, maven.Name, "name", "ASC", 10, 0, "", nil, nil)
	require.NoError(t, err)
	require.Len(t, *res, 2)
	require.Equal(t, "core-lib", (*res)[0].Name)
	require.Equal(t, "2.0", (*res)[0].LatestVersion)

	res, err = e.artifacts.GetArtifactsByRepo(ctx, 10, maven.Name, "created_at", "DESC", 10, 0, "", nil, nil)
	require.NoError(t, err)
	require.Len(t, *res, 2)

	res, err = e.artifacts.GetArtifactsByRepo(ctx, 10, maven.Name, "download_count", "ASC", 10, 0, "", nil, nil)
	require.NoError(t, err)
	require.Len(t, *res, 2)

	res, err = e.artifacts.GetArtifactsByRepo(ctx, 10, maven.Name, "name", "ASC", 10, 0, "util", nil, nil)
	require.NoError(t, err)
	require.Len(t, *res, 1)

	res, err = e.artifacts.GetArtifactsByRepo(ctx, 10, maven.Name, "name", "ASC", 10, 0, "", []string{"stable"}, nil)
	require.NoError(t, err)
	require.Len(t, *res, 1)
	require.Equal(t, "core-lib", (*res)[0].Name)

	model := artifact.ArtifactTypeModel
	res, err = e.artifacts.GetArtifactsByRepo(ctx, 10, maven.Name, "name", "ASC", 10, 0, "", nil, &model)
	require.NoError(t, err)
	require.Empty(t, *res)

	tests := []struct {
		name   string
		search string
		labels []string
		typ    *artifact.ArtifactType
		want   int64
	}{
		{"all versions counted", "", nil, nil, 3},
		{"search", "util", nil, nil, 1},
		{"labels", "", []string{"java", "stable"}, nil, 2},
		{"type", "", nil, &model, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cnt, cerr := e.artifacts.CountArtifactsByRepo(ctx, 10, maven.Name, tc.search, tc.labels, tc.typ)
			require.NoError(t, cerr)
			require.Equal(t, tc.want, cnt)
		})
	}
}

func TestRdbArtifact_LatestAndVersions(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	maven := rdbArtifactListFixture(t, e)

	md, err := e.artifacts.GetLatestArtifactMetadata(ctx, 10, maven.Name, "core-lib")
	require.NoError(t, err)
	require.NotNil(t, md)
	require.Equal(t, "core-lib", md.Name)
	require.Equal(t, maven.Name, md.RepoName)

	md, err = e.artifacts.GetLatestArtifactMetadata(ctx, 10, maven.Name, "missing")
	require.NoError(t, err)
	require.Nil(t, md)

	version, err := e.artifacts.GetArtifactMetadata(ctx, 10, maven.Name, "core-lib", "1.0", nil)
	require.NoError(t, err)
	require.Equal(t, "1.0", version.Name)

	versions, err := e.artifacts.GetAllVersionsByRepoAndImage(ctx, maven.ID, "core-lib", "name", "ASC", 10, 0, "", nil)
	require.NoError(t, err)
	require.Len(t, *versions, 2)
	require.Equal(t, "1.0", (*versions)[0].Name)

	versions, err = e.artifacts.GetAllVersionsByRepoAndImage(
		ctx, maven.ID, "core-lib", "created_at", "DESC", 10, 0, "2", nil)
	require.NoError(t, err)
	require.Len(t, *versions, 1)

	versions, err = e.artifacts.GetAllVersionsByRepoAndImage(
		ctx, maven.ID, "core-lib", "download_count", "ASC", 10, 0, "", nil)
	require.NoError(t, err)
	require.Len(t, *versions, 2)

	model := artifact.ArtifactTypeModel
	versions, err = e.artifacts.GetAllVersionsByRepoAndImage(ctx, maven.ID, "core-lib", "name", "ASC", 10, 0, "", &model)
	require.NoError(t, err)
	require.Empty(t, *versions)

	cnt, err := e.artifacts.CountAllVersionsByRepoAndImage(ctx, 10, maven.Name, "core-lib", "", nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt)
	cnt, err = e.artifacts.CountAllVersionsByRepoAndImage(ctx, 10, maven.Name, "core-lib", "1", nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)
	cnt, err = e.artifacts.CountAllVersionsByRepoAndImage(ctx, 10, maven.Name, "core-lib", "", &model)
	require.NoError(t, err)
	require.Zero(t, cnt)
}

func TestRdbArtifact_BatchesAndSearch(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	maven := rdbArtifactListFixture(t, e)

	all, err := e.artifacts.GetAllArtifactsByRepo(ctx, maven.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, *all, 3)

	rest, err := e.artifacts.GetAllArtifactsByRepo(ctx, maven.ID, 10, (*all)[0].ID)
	require.NoError(t, err)
	require.Len(t, *rest, 2)

	latest, err := e.artifacts.GetLatestArtifactsByRepo(ctx, maven.ID, 10, 0)
	require.NoError(t, err)
	require.Len(t, *latest, 2)

	byImage, err := e.artifacts.GetArtifactsByRepoAndImageBatch(ctx, maven.ID, "core-lib", 500, 0)
	require.NoError(t, err)
	require.Len(t, *byImage, 2)

	search, err := e.artifacts.SearchByImageName(ctx, maven.ID, "lib", 10, 0)
	require.NoError(t, err)
	require.Len(t, *search, 3)
	search, err = e.artifacts.SearchByImageName(ctx, maven.ID, "", 1, 1)
	require.NoError(t, err)
	require.Len(t, *search, 1)

	cnt, err := e.artifacts.CountByImageName(ctx, maven.ID, "util")
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)
	cnt, err = e.artifacts.CountByImageName(ctx, maven.ID, "")
	require.NoError(t, err)
	require.Equal(t, int64(3), cnt)

	latestCnt, err := e.artifacts.CountLatestByName(ctx, maven.ID, "core")
	require.NoError(t, err)
	require.Equal(t, int64(2), latestCnt)

	dc, err := e.artifacts.GetDownloadCountByID(ctx, 1)
	require.NoError(t, err)
	require.Zero(t, dc)
}
