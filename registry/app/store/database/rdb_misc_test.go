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
	"encoding/json"
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	gstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/stretchr/testify/require"
)

func TestRdbGenericBlob(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	dao := NewGenericBlobDao(e.db)

	size, err := dao.TotalSizeByRootParentID(ctx, rdbRootSpaceID)
	require.NoError(t, err)
	require.Zero(t, size)

	gb := &types.GenericBlob{
		RootParentID: rdbRootSpaceID,
		Sha1:         "aa11",
		Sha256:       "bb22",
		Sha512:       "cc33",
		MD5:          "dd44",
		Size:         321,
	}
	id, isNew, err := dao.Create(ctx, gb)
	require.NoError(t, err)
	require.True(t, isNew)
	require.NotEmpty(t, id)

	// Creating a blob with the same sha256 reports the existing id.
	dup := &types.GenericBlob{RootParentID: rdbRootSpaceID, Sha256: "bb22", Sha512: "other", Size: 1}
	dupID, _, err := dao.Create(ctx, dup)
	require.NoError(t, err)
	require.Equal(t, id, dupID)

	byID, err := dao.FindByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "bb22", byID.Sha256)
	require.Equal(t, int64(321), byID.Size)
	require.Equal(t, "dd44", byID.MD5)

	bySha, err := dao.FindBySha256AndRootParentID(ctx, "bb22", rdbRootSpaceID)
	require.NoError(t, err)
	require.Equal(t, id, bySha.ID)

	_, err = dao.FindByID(ctx, "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = dao.FindBySha256AndRootParentID(ctx, "bb22", rdbRootSpaceID+1)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	size, err = dao.TotalSizeByRootParentID(ctx, rdbRootSpaceID)
	require.NoError(t, err)
	require.Equal(t, int64(321), size)

	require.Panics(t, func() { _ = dao.DeleteByID(ctx, id) })
}

func TestRdbBandwidthStat(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newDockerRegistry(t, "docker")
	img := e.newImage(t, reg.ID, "app")
	dao := NewBandwidthStatDao(e.db)

	stat := &types.BandwidthStat{ImageID: img.ID, Type: types.BandwidthType("DOWNLOAD"), Bytes: 2048}
	require.NoError(t, dao.Create(ctx, stat))
	require.NotZero(t, stat.ID)

	var total int64
	require.NoError(t, e.db.Get(&total, "SELECT SUM(bandwidth_stat_bytes) FROM bandwidth_stats"))
	require.Equal(t, int64(2048), total)

	// A stat for an image that does not exist violates the foreign key.
	err := dao.Create(ctx, &types.BandwidthStat{ImageID: 9999, Type: types.BandwidthType("UPLOAD")})
	require.ErrorIs(t, err, gstore.ErrForeignKeyViolation)
}

func TestRdbPackageTag(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newRegistry(t, rdbRootSpaceID, "npm", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeNPM)
	img := e.newImage(t, reg.ID, "left-pad")
	a1 := e.newArtifact(t, img.ID, "1.0.0")
	a2 := e.newArtifact(t, img.ID, "2.0.0")
	dao := NewPackageTagDao(e.db)

	id, err := dao.Create(ctx, &types.PackageTag{ID: "pt-1", Name: "latest", ArtifactID: a1.ID})
	require.NoError(t, err)
	require.Equal(t, "pt-1", id)
	_, err = dao.Create(ctx, &types.PackageTag{ID: "pt-2", Name: "next", ArtifactID: a2.ID})
	require.NoError(t, err)
	// Same name and artifact upserts without changing the original id.
	_, err = dao.Create(ctx, &types.PackageTag{ID: "pt-3", Name: "latest", ArtifactID: a1.ID})
	require.NoError(t, err)

	tags, err := dao.FindByImageNameAndRegID(ctx, "left-pad", reg.ID, nil)
	require.NoError(t, err)
	require.Len(t, tags, 2)
	versions := map[string]string{}
	for _, tg := range tags {
		versions[tg.Name] = tg.Version
	}
	require.Equal(t, map[string]string{"latest": "1.0.0", "next": "2.0.0"}, versions)

	typed := "model"
	tags, err = dao.FindByImageNameAndRegID(ctx, "left-pad", reg.ID, &typed)
	require.NoError(t, err)
	require.Empty(t, tags)

	require.NoError(t, dao.DeleteByTagAndImageName(ctx, "latest", "left-pad", reg.ID))
	tags, err = dao.FindByImageNameAndRegID(ctx, "left-pad", reg.ID, nil)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	require.Equal(t, "next", tags[0].Name)

	require.NoError(t, dao.DeleteByImageNameAndRegID(ctx, "left-pad", reg.ID))
	tags, err = dao.FindByImageNameAndRegID(ctx, "left-pad", reg.ID, nil)
	require.NoError(t, err)
	require.Empty(t, tags)
}

func TestRdbDownloadStats(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg1 := e.newRegistry(t, rdbRootSpaceID, "maven-one", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	reg2 := e.newRegistry(t, rdbRootSpaceID, "maven-two", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	img1 := e.newImage(t, reg1.ID, "lib-a")
	img2 := e.newImage(t, reg2.ID, "lib-b")
	a1 := e.newArtifact(t, img1.ID, "1.0")
	a2 := e.newArtifact(t, img1.ID, "2.0")
	b1 := e.newArtifact(t, img2.ID, "1.0")
	dao := NewDownloadStatDao(e.db)

	for _, id := range []int64{a1.ID, a1.ID, a2.ID, b1.ID} {
		require.NoError(t, dao.Create(ctx, &types.DownloadStat{ArtifactID: id}))
	}

	total, err := dao.GetTotalDownloadsForImage(ctx, img1.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)

	total, err = dao.GetTotalDownloadsForArtifactID(ctx, a1.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)

	total, err = dao.GetTotalDownloadsForRegistryID(ctx, reg1.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)

	byVersion, err := dao.GetTotalDownloadsForManifests(ctx, []string{"1.0", "2.0", "3.0"}, img1.ID)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"1.0": 2, "2.0": 1}, byVersion)

	regCounts, err := dao.GetTotalDownloadsForRegistryIDs(ctx, []int64{reg1.ID, reg2.ID})
	require.NoError(t, err)
	got := map[int64]int64{}
	for _, c := range regCounts {
		got[c.EntityID] = c.Count
	}
	require.Equal(t, map[int64]int64{reg1.ID: 3, reg2.ID: 1}, got)

	imgCounts, err := dao.GetTotalDownloadsForImageIDs(ctx, []int64{img1.ID, img2.ID})
	require.NoError(t, err)
	got = map[int64]int64{}
	for _, c := range imgCounts {
		got[c.EntityID] = c.Count
	}
	require.Equal(t, map[int64]int64{img1.ID: 3, img2.ID: 1}, got)

	artCounts, err := dao.GetTotalDownloadsForArtifactIDs(ctx, []int64{a1.ID, a2.ID})
	require.NoError(t, err)
	got = map[int64]int64{}
	for _, c := range artCounts {
		got[c.EntityID] = c.Count
	}
	require.Equal(t, map[int64]int64{a1.ID: 2, a2.ID: 1}, got)

	// Empty id lists short-circuit without querying.
	empty, err := dao.GetTotalDownloadsForRegistryIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, empty)
	empty, err = dao.GetTotalDownloadsForImageIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, empty)
	empty, err = dao.GetTotalDownloadsForArtifactIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, empty)

	// A stat for an artifact that does not exist violates the foreign key.
	require.ErrorIs(t, dao.Create(ctx, &types.DownloadStat{ArtifactID: 9999}), gstore.ErrForeignKeyViolation)
}

func TestRdbDownloadStats_CreateByRegistryImageAndVersion(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newRegistry(t, rdbRootSpaceID, "maven", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	img := e.newImage(t, reg.ID, "lib")
	a := e.newArtifact(t, img.ID, "1.0")
	dao := NewDownloadStatDao(e.db)

	require.NoError(t, dao.CreateByRegistryIDImageAndArtifactName(ctx, reg.ID, "lib", "1.0", nil))
	total, err := dao.GetTotalDownloadsForArtifactID(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)

	// Unknown versions insert nothing.
	require.NoError(t, dao.CreateByRegistryIDImageAndArtifactName(ctx, reg.ID, "lib", "9.9", nil))
	total, err = dao.GetTotalDownloadsForArtifactID(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)

	model := artifact.ArtifactTypeModel
	require.NoError(t, dao.CreateByRegistryIDImageAndArtifactName(ctx, reg.ID, "lib", "1.0", &model))
	total, err = dao.GetTotalDownloadsForArtifactID(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
}

func TestRdbCleanupPolicy(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newDockerRegistry(t, "docker")
	dao := NewCleanupPolicyDao(e.db, dbtx.New(e.db))

	ids, err := dao.GetIDsByRegistryID(ctx, reg.ID)
	require.NoError(t, err)
	require.Empty(t, ids)

	policies := []types.CleanupPolicy{
		{
			RegistryID: reg.ID, Name: "old", ExpiryTime: 1000,
			PackagePrefix: []string{"pkg-a", "pkg-b"}, VersionPrefix: []string{"1."},
		},
		{RegistryID: reg.ID, Name: "ancient", ExpiryTime: 5000, VersionPrefix: []string{"0."}},
	}
	require.NoError(t, dao.ModifyCleanupPolicies(ctx, &policies, nil))

	ids, err = dao.GetIDsByRegistryID(ctx, reg.ID)
	require.NoError(t, err)
	require.Len(t, ids, 2)

	got, err := dao.GetByRegistryID(ctx, reg.ID)
	require.NoError(t, err)
	require.Len(t, *got, 2)
	byName := map[string]types.CleanupPolicy{}
	for _, cp := range *got {
		byName[cp.Name] = cp
	}
	require.Equal(t, int64(1000), byName["old"].ExpiryTime)
	require.ElementsMatch(t, []string{"pkg-a", "pkg-b"}, byName["old"].PackagePrefix)
	require.Equal(t, []string{"1."}, byName["old"].VersionPrefix)
	require.Equal(t, []string{"0."}, byName["ancient"].VersionPrefix)
	require.Empty(t, byName["ancient"].PackagePrefix)

	// Replace all existing policies with a single new one.
	replacement := []types.CleanupPolicy{{RegistryID: reg.ID, Name: "fresh", ExpiryTime: 1, PackagePrefix: []string{"x"}}}
	require.NoError(t, dao.ModifyCleanupPolicies(ctx, &replacement, ids))
	got, err = dao.GetByRegistryID(ctx, reg.ID)
	require.NoError(t, err)
	require.Len(t, *got, 1)
	require.Equal(t, "fresh", (*got)[0].Name)

	// Deleting the remaining policy cascades its prefix mappings.
	remaining, err := dao.GetIDsByRegistryID(ctx, reg.ID)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	require.NoError(t, dao.Delete(ctx, remaining[0]))
	got, err = dao.GetByRegistryID(ctx, reg.ID)
	require.NoError(t, err)
	require.Empty(t, *got)

	// A policy for an unknown registry fails inside the transaction and is rolled back.
	bad := []types.CleanupPolicy{{RegistryID: 9999, Name: "bad"}}
	require.Error(t, dao.ModifyCleanupPolicies(ctx, &bad, nil))
	// Clearing with no policies and no ids is a no-op.
	require.NoError(t, dao.ModifyCleanupPolicies(ctx, nil, nil))
}

func TestRdbTasks(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	tasks := NewTaskStore(e.db, dbtx.New(e.db))
	sources := NewTaskSourceStore(e.db, dbtx.New(e.db))
	events := NewTaskEventStore(e.db)

	payload := json.RawMessage(`{"key":"k1","registry_id":1}`)
	task := &types.Task{Key: "k1", Kind: types.TaskKindBuildRegistryIndex, Payload: payload}
	require.NoError(t, tasks.UpsertTask(ctx, task))
	require.NoError(t, tasks.UpsertTask(ctx, task))
	require.NoError(t, tasks.UpsertTask(ctx, &types.Task{
		Key: "k2", Kind: types.TaskKindBuildPackageIndex, Payload: json.RawMessage(`{}`),
	}))

	found, err := tasks.Find(ctx, "k1")
	require.NoError(t, err)
	require.Equal(t, types.TaskKindBuildRegistryIndex, found.Kind)
	require.Equal(t, types.TaskStatusPending, found.Status)
	require.False(t, found.RunAgain)
	require.JSONEq(t, string(payload), string(found.Payload))
	_, err = tasks.Find(ctx, "missing")
	require.Error(t, err)

	pending, err := tasks.ListPendingTasks(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 2)
	pending, err = tasks.ListPendingTasks(ctx, 1)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	status, err := tasks.LockForUpdate(ctx, task)
	require.NoError(t, err)
	require.Equal(t, types.TaskStatusPending, status)
	_, err = tasks.LockForUpdate(ctx, &types.Task{Key: "missing"})
	require.Error(t, err)

	require.NoError(t, tasks.UpdateStatus(ctx, "k1", types.TaskStatusProcessing))
	found, err = tasks.Find(ctx, "k1")
	require.NoError(t, err)
	require.Equal(t, types.TaskStatusProcessing, found.Status)
	pending, err = tasks.ListPendingTasks(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	require.NoError(t, tasks.SetRunAgain(ctx, "k1", true))
	runAgain, err := tasks.CompleteTask(ctx, "k1", types.TaskStatusSuccess)
	require.NoError(t, err)
	require.True(t, runAgain)
	found, err = tasks.Find(ctx, "k1")
	require.NoError(t, err)
	require.Equal(t, types.TaskStatusSuccess, found.Status)
	_, err = tasks.CompleteTask(ctx, "missing", types.TaskStatusSuccess)
	require.Error(t, err)

	// Task sources. The status column is read back with raw SQL because the finder does not select it.
	sourceStatus := func(key string) types.TaskStatus {
		var st string
		require.NoError(t, e.db.Get(&st,
			"SELECT registry_task_source_status FROM registry_task_sources WHERE registry_task_source_key = ?", key))
		return types.TaskStatus(st)
	}
	ref := types.SourceRef{Type: types.SourceTypeRegistry, ID: 5}
	require.NoError(t, sources.InsertSource(ctx, "k2", ref))
	require.NoError(t, sources.InsertSource(ctx, "k2", ref))
	src, err := sources.FindByTaskKeyAndSourceType(ctx, "k2", string(types.SourceTypeRegistry))
	require.NoError(t, err)
	require.Equal(t, int64(5), src.SrcID)
	require.Equal(t, types.SourceTypeRegistry, src.SrcType)
	require.Equal(t, types.TaskStatusPending, sourceStatus("k2"))
	require.Nil(t, src.RunID)
	require.Equal(t, ref, (&TaskSourceDB{SrcType: string(src.SrcType), SrcID: src.SrcID}).ToSourceRef())
	_, err = sources.FindByTaskKeyAndSourceType(ctx, "k2", string(types.SourceTypeArtifact))
	require.Error(t, err)

	require.NoError(t, sources.ClaimSources(ctx, "k2", "run-1"))
	src, err = sources.FindByTaskKeyAndSourceType(ctx, "k2", string(types.SourceTypeRegistry))
	require.NoError(t, err)
	require.Equal(t, types.TaskStatusProcessing, sourceStatus("k2"))
	require.NotNil(t, src.RunID)
	require.Equal(t, "run-1", *src.RunID)

	require.NoError(t, sources.UpdateSourceStatus(ctx, "run-1", types.TaskStatusFailure, "boom"))
	src, err = sources.FindByTaskKeyAndSourceType(ctx, "k2", string(types.SourceTypeRegistry))
	require.NoError(t, err)
	require.Equal(t, types.TaskStatusFailure, sourceStatus("k2"))
	require.NotNil(t, src.Error)
	require.Equal(t, "boom", *src.Error)

	// Re-inserting a failed source resets it to pending.
	require.NoError(t, sources.InsertSource(ctx, "k2", ref))
	require.Equal(t, types.TaskStatusPending, sourceStatus("k2"))

	require.NoError(t, sources.UpdateSourceStatus(ctx, "run-1", types.TaskStatusSuccess, ""))
	src, err = sources.FindByTaskKeyAndSourceType(ctx, "k2", string(types.SourceTypeRegistry))
	require.NoError(t, err)
	require.Nil(t, src.Error)
	require.Equal(t, types.TaskStatusSuccess, sourceStatus("k2"))

	// Task events.
	require.NoError(t, events.LogTaskEvent(ctx, "k1", "started", []byte(`{"a":1}`)))
	var n int
	require.NoError(t, e.db.Get(&n, "SELECT COUNT(*) FROM registry_task_events WHERE registry_task_event_key = 'k1'"))
	require.Equal(t, 1, n)
}
