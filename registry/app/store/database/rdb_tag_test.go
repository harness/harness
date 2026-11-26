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

// rdbTagFixture is a docker registry with one image, two manifests and three tags.
type rdbTagFixture struct {
	reg    *types.Registry
	image  *types.Image
	m1, m2 *types.Manifest
}

func rdbNewTagFixture(t *testing.T, e *rdbEnv) *rdbTagFixture {
	t.Helper()
	reg := e.newRegistry(t, 10, "docker-repo", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeDOCKER)
	img := e.newImage(t, reg.ID, "app")
	m1 := e.newManifest(t, reg.ID, "app")
	m2 := e.newManifest(t, reg.ID, "app")
	e.newTag(t, reg.ID, "app", "v1", m1.ID)
	e.newTag(t, reg.ID, "app", "v2", m2.ID)
	e.newTag(t, reg.ID, "app", "latest", m2.ID)
	return &rdbTagFixture{reg: reg, image: img, m1: m1, m2: m2}
}

func TestRdbTag_CreateFindDelete(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	f := rdbNewTagFixture(t, e)

	got, err := e.tags.FindTag(ctx, f.reg.ID, "app", "v1")
	require.NoError(t, err)
	require.Equal(t, f.m1.ID, got.ManifestID)
	require.Equal(t, "v1", got.Name)

	// Re-pointing a tag at another manifest updates it in place.
	require.NoError(t, e.tags.CreateOrUpdate(ctx, &types.Tag{
		Name: "v1", ImageName: "app", RegistryID: f.reg.ID, ManifestID: f.m2.ID,
	}))
	got, err = e.tags.FindTag(ctx, f.reg.ID, "app", "v1")
	require.NoError(t, err)
	require.Equal(t, f.m2.ID, got.ManifestID)

	// Re-creating a tag that already points at the same manifest is a no-op.
	require.NoError(t, e.tags.CreateOrUpdate(ctx, &types.Tag{
		Name: "v1", ImageName: "app", RegistryID: f.reg.ID, ManifestID: f.m2.ID,
	}))

	// A tag pointing to a missing manifest violates the foreign key.
	err = e.tags.CreateOrUpdate(ctx, &types.Tag{
		Name: "bad", ImageName: "app", RegistryID: f.reg.ID, ManifestID: 99999,
	})
	require.ErrorIs(t, err, gstore.ErrForeignKeyViolation)

	_, err = e.tags.FindTag(ctx, f.reg.ID, "app", "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	detail, err := e.tags.GetTagDetail(ctx, f.reg.ID, "app", "v2")
	require.NoError(t, err)
	require.Equal(t, "v2", detail.Name)
	require.Equal(t, "app", detail.ImageName)
	_, err = e.tags.GetTagDetail(ctx, f.reg.ID, "app", "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	names, err := e.tags.GetTagsByManifestID(ctx, f.m2.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"v1", "v2", "latest"}, *names)
	names, err = e.tags.GetTagsByManifestID(ctx, f.m1.ID)
	require.NoError(t, err)
	require.Empty(t, *names)

	require.NoError(t, e.tags.DeleteTag(ctx, f.reg.ID, "app", "v1"))
	_, err = e.tags.FindTag(ctx, f.reg.ID, "app", "v1")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	deleted, err := e.tags.DeleteTagByName(ctx, f.reg.ID, "v2")
	require.NoError(t, err)
	require.True(t, deleted)
	deleted, err = e.tags.DeleteTagByName(ctx, f.reg.ID, "v2")
	require.NoError(t, err)
	require.False(t, deleted)

	deleted, err = e.tags.DeleteTagByManifestID(ctx, f.reg.ID, f.m2.ID)
	require.NoError(t, err)
	require.True(t, deleted)
	deleted, err = e.tags.DeleteTagByManifestID(ctx, f.reg.ID, f.m2.ID)
	require.NoError(t, err)
	require.False(t, deleted)

	e.newTag(t, f.reg.ID, "app", "again", f.m1.ID)
	require.NoError(t, e.tags.DeleteTagsByImageName(ctx, f.reg.ID, "app"))
	names, err = e.tags.GetTagsByManifestID(ctx, f.m1.ID)
	require.NoError(t, err)
	require.Empty(t, *names)
}

func TestRdbTag_PaginationAndLatest(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	f := rdbNewTagFixture(t, e)

	page, err := e.tags.TagsPaginated(ctx, f.reg.ID, "app", types.FilterParams{MaxEntries: 2})
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "latest", page[0].Name)
	require.Equal(t, "v1", page[1].Name)

	page, err = e.tags.TagsPaginated(ctx, f.reg.ID, "app", types.FilterParams{MaxEntries: 5, LastEntry: "v1"})
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "v2", page[0].Name)

	hasTests := []struct {
		name   string
		params types.FilterParams
		want   bool
	}{
		{"nothing after last", types.FilterParams{MaxEntries: 5, LastEntry: "v2"}, false},
		{"something after v1", types.FilterParams{MaxEntries: 5, LastEntry: "v1"}, true},
		{"descending finds latest", types.FilterParams{MaxEntries: 5, LastEntry: "v1", SortOrder: OrderDesc}, true},
		{"descending nothing before", types.FilterParams{MaxEntries: 5, LastEntry: "latest", SortOrder: OrderDesc}, false},
		{"name filter", types.FilterParams{MaxEntries: 5, LastEntry: "v1", Name: "v"}, true},
	}
	for _, tc := range hasTests {
		t.Run(tc.name, func(t *testing.T) {
			has, herr := e.tags.HasTagsAfterName(ctx, f.reg.ID, tc.params)
			require.NoError(t, herr)
			require.Equal(t, tc.want, has)
		})
	}

	latest, err := e.tags.GetLatestTag(ctx, f.reg.ID, "app")
	require.NoError(t, err)
	require.NotEmpty(t, latest.Name)
	_, err = e.tags.GetLatestTag(ctx, f.reg.ID, "none")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	latestName, err := e.tags.GetLatestTagName(ctx, f.reg.ParentID, f.reg.Name, "app")
	require.NoError(t, err)
	require.NotEmpty(t, latestName)
	_, err = e.tags.GetLatestTagName(ctx, f.reg.ParentID, f.reg.Name, "none")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	md, err := e.tags.GetLatestTagMetadata(ctx, f.reg.ParentID, f.reg.Name, "app")
	require.NoError(t, err)
	require.Equal(t, "app", md.Name)
	_, err = e.tags.GetLatestTagMetadata(ctx, f.reg.ParentID, f.reg.Name, "none")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// Soft deleted images are hidden unless the query options say otherwise.
	e.softDelete(t, "images", "image", f.image.ID)
	_, err = e.tags.GetLatestTag(ctx, f.reg.ID, "app")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.tags.GetLatestTag(ctx, f.reg.ID, "app", types.WithOnlyDeleted())
	require.NoError(t, err)
	_, err = e.tags.GetLatestTag(ctx, f.reg.ID, "app", types.WithAllDeleted())
	require.NoError(t, err)

	_, err = e.tags.GetLatestTagName(ctx, f.reg.ParentID, f.reg.Name, "app")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.tags.GetLatestTagName(ctx, f.reg.ParentID, f.reg.Name, "app", types.WithOnlyDeleted())
	require.NoError(t, err)
	_, err = e.tags.GetLatestTagName(ctx, f.reg.ParentID, f.reg.Name, "app", types.WithAllDeleted())
	require.NoError(t, err)

	_, err = e.tags.GetLatestTagMetadata(ctx, f.reg.ParentID, f.reg.Name, "app")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.tags.GetLatestTagMetadata(ctx, f.reg.ParentID, f.reg.Name, "app", types.WithOnlyDeleted())
	require.NoError(t, err)
	_, err = e.tags.GetLatestTagMetadata(ctx, f.reg.ParentID, f.reg.Name, "app", types.WithAllDeleted())
	require.NoError(t, err)

	_, err = e.tags.FindTag(ctx, f.reg.ID, "app", "v1")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
}

func TestRdbTag_RepoAndImageListings(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	f := rdbNewTagFixture(t, e)
	f.image.Labels = []string{"prod", "web"}
	require.NoError(t, e.images.Update(ctx, f.image))
	other := e.newImage(t, f.reg.ID, "worker")
	wm := e.newManifest(t, f.reg.ID, "worker")
	e.newTag(t, f.reg.ID, "worker", "w1", wm.ID)
	require.NotZero(t, other.ID)

	res, err := e.tags.GetAllArtifactsByRepo(ctx, 10, f.reg.Name, "updated_at", "DESC", 10, 0, "", nil)
	require.NoError(t, err)
	require.Len(t, *res, 2)

	res, err = e.tags.GetAllArtifactsByRepo(ctx, 10, f.reg.Name, "download_count", "ASC", 10, 0, "work", nil)
	require.NoError(t, err)
	require.Len(t, *res, 1)
	require.Equal(t, "worker", (*res)[0].Name)

	res, err = e.tags.GetAllArtifactsByRepo(ctx, 10, f.reg.Name, "updated_at", "ASC", 10, 0, "", []string{"web", "prod"})
	require.NoError(t, err)
	require.Len(t, *res, 1)
	require.Equal(t, "app", (*res)[0].Name)

	cnt, err := e.tags.CountAllArtifactsByRepo(ctx, 10, f.reg.Name, "", nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt)
	cnt, err = e.tags.CountAllArtifactsByRepo(ctx, 10, f.reg.Name, "work", nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)
	cnt, err = e.tags.CountAllArtifactsByRepo(ctx, 10, f.reg.Name, "", []string{"prod"})
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)

	tags, err := e.tags.GetAllTagsByRepoAndImage(ctx, 10, f.reg.Name, "app", "name", "ASC", 10, 0, "")
	require.NoError(t, err)
	require.Len(t, *tags, 3)
	require.Equal(t, "latest", (*tags)[0].Name)
	tags, err = e.tags.GetAllTagsByRepoAndImage(ctx, 10, f.reg.Name, "app", "name", "DESC", 1, 1, "v")
	require.NoError(t, err)
	require.Len(t, *tags, 1)

	tagCount, err := e.tags.CountAllTagsByRepoAndImage(ctx, 10, f.reg.Name, "app", "")
	require.NoError(t, err)
	require.Equal(t, int64(3), tagCount)
	tagCount, err = e.tags.CountAllTagsByRepoAndImage(ctx, 10, f.reg.Name, "app", "v")
	require.NoError(t, err)
	require.Equal(t, int64(2), tagCount)

	info, err := e.tags.GetOciTagsInfo(ctx, f.reg.ID, "app", 10, 0, "")
	require.NoError(t, err)
	require.Len(t, *info, 3)
	info, err = e.tags.GetOciTagsInfo(ctx, f.reg.ID, "app", 10, 0, "lat")
	require.NoError(t, err)
	require.Len(t, *info, 1)
	require.Equal(t, "latest", (*info)[0].Name)
}

func TestRdbTag_OciVersions(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	f := rdbNewTagFixture(t, e)

	versions, err := e.tags.GetAllOciVersionsByRepoAndImage(ctx, 10, f.reg.Name, "app", "size", "ASC", 10, 0, "")
	require.NoError(t, err)
	require.Len(t, *versions, 2)

	versions, err = e.tags.GetAllOciVersionsByRepoAndImage(ctx, 10, f.reg.Name, "app", "updated_at", "DESC", 10, 0, "")
	require.NoError(t, err)
	require.Len(t, *versions, 2)

	// Searching by a valid digest matches the manifest, other terms match tag names.
	byDigest, err := e.tags.GetAllOciVersionsByRepoAndImage(
		ctx, 10, f.reg.Name, "app", "size", "ASC", 10, 0, f.m1.Digest.String())
	require.NoError(t, err)
	require.Len(t, *byDigest, 1)

	byTag, err := e.tags.GetAllOciVersionsByRepoAndImage(ctx, 10, f.reg.Name, "app", "size", "ASC", 10, 0, "lat")
	require.NoError(t, err)
	require.Len(t, *byTag, 1)

	cnt, err := e.tags.CountOciVersionByRepoAndImage(ctx, 10, f.reg.Name, "app", "")
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt)
	cnt, err = e.tags.CountOciVersionByRepoAndImage(ctx, 10, f.reg.Name, "app", f.m1.Digest.String())
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)
	cnt, err = e.tags.CountOciVersionByRepoAndImage(ctx, 10, f.reg.Name, "app", "lat")
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)
}

func TestRdbTag_ParentListings(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	f := rdbNewTagFixture(t, e)

	maven := e.newRegistry(t, 10, "maven-repo", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	lib := e.newImage(t, maven.ID, "lib")
	e.newArtifact(t, lib.ID, "1.0")
	e.newArtifact(t, lib.ID, "2.0")

	none := []string{}
	tests := []struct {
		name     string
		repos    []string
		pkgTypes []string
		search   string
		latest   bool
		want     int
	}{
		{"everything", none, nil, "", false, 5},
		{"latest only", none, nil, "", true, 2},
		{"docker only", none, []string{"DOCKER"}, "", false, 3},
		{"repo filter", []string{"maven-repo"}, nil, "", false, 2},
		{"search", none, nil, "lib", false, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repos := tc.repos
			res, err := e.tags.GetAllArtifactsByParentID(
				ctx, 10, &repos, "image_name", "ASC", 50, 0, tc.search, tc.latest, tc.pkgTypes)
			require.NoError(t, err)
			require.Len(t, *res, tc.want)
		})
	}

	repos := none
	res, err := e.tags.GetAllArtifactsByParentID(ctx, 10, &repos, "modified_at", "DESC", 1, 1, "", false, nil)
	require.NoError(t, err)
	require.Len(t, *res, 1)

	cnt, err := e.tags.CountAllArtifactsByParentID(ctx, 10, &repos, "", false, nil, false)
	require.NoError(t, err)
	require.Equal(t, int64(5), cnt)
	cnt, err = e.tags.CountAllArtifactsByParentID(ctx, 10, &repos, "", true, nil, false)
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt)
	cnt, err = e.tags.CountAllArtifactsByParentID(ctx, 10, &repos, "lib", false, []string{"MAVEN"}, false)
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt)

	untaggedCnt, err := e.tags.CountAllArtifactsByParentID(ctx, 10, &repos, "", false, nil, true)
	require.NoError(t, err)
	require.Equal(t, int64(2), untaggedCnt)
	untaggedCnt, err = e.tags.CountAllArtifactsByParentID(ctx, 10, &[]string{"maven-repo"}, "lib", false,
		[]string{"MAVEN"}, true)
	require.NoError(t, err)
	require.Equal(t, int64(2), untaggedCnt)

	untagged, err := e.tags.GetAllArtifactsByParentIDUntagged(ctx, 10, &repos, "image_name", "ASC", 10, 0, "", nil)
	require.NoError(t, err)
	require.Len(t, *untagged, 2)

	untagged, err = e.tags.GetAllArtifactsByParentIDUntagged(
		ctx, 10, &[]string{"maven-repo"}, "modified_at", "DESC", 10, 0, "lib", []string{"MAVEN"})
	require.NoError(t, err)
	require.Len(t, *untagged, 2)

	untagged, err = e.tags.GetAllArtifactsByParentIDUntagged(ctx, 10, &repos, "modified_at", "ASC", 10, 0, "zzz", nil)
	require.NoError(t, err)
	require.Empty(t, *untagged)

	require.NotNil(t, f)
}

func TestRdbTag_QuarantineLookups(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newRegistry(t, 10, "maven-repo", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	lib := e.newImage(t, reg.ID, "lib")
	other := e.newImage(t, reg.ID, "other")
	a := e.newArtifact(t, lib.ID, "1.0")

	empty, err := e.tags.GetQuarantineStatusForImages(ctx, nil, reg.ID)
	require.NoError(t, err)
	require.Empty(t, empty)

	status, err := e.tags.GetQuarantineStatusForImages(ctx, []string{"lib", "other"}, reg.ID)
	require.NoError(t, err)
	require.Equal(t, []bool{false, false}, status)

	e.exec(t, `INSERT INTO quarantined_paths (quarantined_path_id, quarantined_path_registry_id,
		quarantined_path_image_id, quarantined_path_artifact_id, quarantined_path_reason,
		quarantined_path_created_by, quarantined_path_created_at) VALUES ('q1', ?, ?, ?, 'bad', 1, 5)`,
		reg.ID, lib.ID, a.ID)

	status, err = e.tags.GetQuarantineStatusForImages(ctx, []string{"lib", "other"}, reg.ID)
	require.NoError(t, err)
	require.Equal(t, []bool{true, false}, status)
	require.NotZero(t, other.ID)

	noInfo, err := e.tags.GetQuarantineInfoForArtifacts(ctx, nil, 10)
	require.NoError(t, err)
	require.Empty(t, noInfo)

	info, err := e.tags.GetQuarantineInfoForArtifacts(ctx, []types.ArtifactIdentifier{
		{Name: "lib", Version: "1.0"}, {Name: "other", Version: "9"},
	}, 10)
	require.NoError(t, err)
	require.Len(t, info, 1)
	got := info[types.ArtifactIdentifier{Name: "lib", Version: "1.0", RegistryName: reg.Name}]
	require.NotNil(t, got)
	require.Equal(t, "bad", got.Reason)
	require.Equal(t, int64(5), got.CreatedAt)

	dc, err := e.tags.GetDownloadCountByImageID(ctx, lib.ID)
	require.NoError(t, err)
	require.Zero(t, dc)
	manifestCounts, err := e.tags.GetDownloadCountByManifests(ctx, nil, lib.ID)
	require.NoError(t, err)
	require.Empty(t, manifestCounts)
}
