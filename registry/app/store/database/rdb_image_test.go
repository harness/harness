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

func TestRdbImage_CreateAndGet(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newDockerRegistry(t, "docker")

	img := &types.Image{Name: "web", RegistryID: reg.ID, Enabled: true}
	require.NoError(t, e.images.CreateOrUpdate(ctx, img))
	require.NotZero(t, img.ID)
	require.NotEmpty(t, img.UUID)

	got, err := e.images.Get(ctx, img.ID)
	require.NoError(t, err)
	require.Equal(t, "web", got.Name)
	require.True(t, got.Enabled)
	require.Empty(t, got.Labels)
	require.Nil(t, got.ArtifactType)
	require.Equal(t, int64(7), got.CreatedBy)

	byName, err := e.images.GetByName(ctx, reg.ID, "web")
	require.NoError(t, err)
	require.Equal(t, img.ID, byName.ID)

	byUUID, err := e.images.GetByUUID(ctx, img.UUID)
	require.NoError(t, err)
	require.Equal(t, img.ID, byUUID.ID)

	byRepo, err := e.images.GetByRepoAndName(ctx, reg.ParentID, reg.Name, "web")
	require.NoError(t, err)
	require.Equal(t, img.ID, byRepo.ID)

	// Upsert flips the enabled flag and keeps the id.
	again := &types.Image{Name: "web", RegistryID: reg.ID, Enabled: false}
	require.NoError(t, e.images.CreateOrUpdate(ctx, again))
	require.Equal(t, img.ID, again.ID)
	got, err = e.images.Get(ctx, img.ID)
	require.NoError(t, err)
	require.False(t, got.Enabled)

	require.Error(t, e.images.CreateOrUpdate(ctx, &types.Image{RegistryID: reg.ID}))

	_, err = e.images.Get(ctx, 9999)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.images.GetByName(ctx, reg.ID, "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.images.GetByUUID(ctx, "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.images.GetByRepoAndName(ctx, reg.ParentID, reg.Name, "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
}

func TestRdbImage_TypedImages(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newRegistry(t, rdbRootSpaceID, "hf", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeHUGGINGFACE)

	model := artifact.ArtifactTypeModel
	dataset := artifact.ArtifactTypeDataset
	empty := artifact.ArtifactType("")

	modelImg := &types.Image{Name: "bert", RegistryID: reg.ID, ArtifactType: &model, Enabled: true}
	require.NoError(t, e.images.CreateOrUpdate(ctx, modelImg))
	datasetImg := &types.Image{Name: "bert", RegistryID: reg.ID, ArtifactType: &dataset, Enabled: true}
	require.NoError(t, e.images.CreateOrUpdate(ctx, datasetImg))
	require.NotEqual(t, modelImg.ID, datasetImg.ID)

	got, err := e.images.GetByNameAndType(ctx, reg.ID, "bert", &model)
	require.NoError(t, err)
	require.Equal(t, modelImg.ID, got.ID)
	require.NotNil(t, got.ArtifactType)
	require.Equal(t, model, *got.ArtifactType)

	got, err = e.images.GetByNameAndType(ctx, reg.ID, "bert", &dataset)
	require.NoError(t, err)
	require.Equal(t, datasetImg.ID, got.ID)

	// Without a type the lookup targets untyped images only.
	_, err = e.images.GetByNameAndType(ctx, reg.ID, "bert", nil)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.images.GetByNameAndType(ctx, reg.ID, "bert", &empty)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// Typed images are not returned by the untyped lookup.
	_, err = e.images.GetByName(ctx, reg.ID, "bert")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// Upsert on a typed image hits the partial unique index for typed images.
	upsert := &types.Image{Name: "bert", RegistryID: reg.ID, ArtifactType: &model, Enabled: false}
	require.NoError(t, e.images.CreateOrUpdate(ctx, upsert))
	require.Equal(t, modelImg.ID, upsert.ID)
}

func TestRdbImage_SoftDeleteFilters(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newDockerRegistry(t, "docker")
	model := artifact.ArtifactTypeModel

	img := e.newImage(t, reg.ID, "gone")
	typed := &types.Image{Name: "gone-typed", RegistryID: reg.ID, ArtifactType: &model, Enabled: true}
	require.NoError(t, e.images.CreateOrUpdate(ctx, typed))
	e.softDelete(t, "images", "image", img.ID)
	e.softDelete(t, "images", "image", typed.ID)

	tests := []struct {
		name  string
		opts  []types.QueryOption
		found bool
	}{
		{"default", nil, false},
		{"exclude", []types.QueryOption{types.WithExcludeDeleted()}, false},
		{"only deleted", []types.QueryOption{types.WithOnlyDeleted()}, true},
		{"include all", []types.QueryOption{types.WithAllDeleted()}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			check := func(err error) {
				if tc.found {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, gstore.ErrResourceNotFound)
				}
			}
			_, err := e.images.Get(ctx, img.ID, tc.opts...)
			check(err)
			_, err = e.images.GetByName(ctx, reg.ID, "gone", tc.opts...)
			check(err)
			_, err = e.images.GetByNameAndType(ctx, reg.ID, "gone-typed", &model, tc.opts...)
			check(err)
		})
	}

	// The default getters used by listing hide soft-deleted images.
	_, err := e.images.GetByUUID(ctx, img.UUID)
	require.NoError(t, err)
	_, err = e.images.GetByRepoAndName(ctx, reg.ParentID, reg.Name, "gone")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// Re-creating a soft-deleted image is rejected.
	require.Error(t, e.images.CreateOrUpdate(ctx, &types.Image{Name: "gone", RegistryID: reg.ID}))
	require.Error(t, e.images.CreateOrUpdate(ctx, &types.Image{
		Name: "gone-typed", RegistryID: reg.ID, ArtifactType: &model,
	}))
}

func TestRdbImage_UpdateAndStatus(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newDockerRegistry(t, "docker")

	img := e.newImage(t, reg.ID, "web")
	got, err := e.images.Get(ctx, img.ID)
	require.NoError(t, err)

	got.Labels = []string{"b", "a"}
	got.Enabled = false
	require.NoError(t, e.images.Update(ctx, got))
	updated, err := e.images.Get(ctx, img.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, updated.Labels)
	require.False(t, updated.Enabled)

	require.ErrorIs(t, e.images.Update(ctx, &types.Image{ID: 9999, Name: "x", RegistryID: reg.ID}),
		gstore.ErrVersionConflict)

	require.NoError(t, e.images.UpdateStatus(ctx, &types.Image{Name: "web", RegistryID: reg.ID, Enabled: true}))
	updated, err = e.images.Get(ctx, img.ID)
	require.NoError(t, err)
	require.True(t, updated.Enabled)

	require.ErrorIs(t, e.images.UpdateStatus(ctx, &types.Image{Name: "none", RegistryID: reg.ID}),
		gstore.ErrVersionConflict)
}

func TestRdbImage_LabelsAndDeletes(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newDockerRegistry(t, "docker")

	for name, labels := range map[string][]string{
		"a": {"red", "blue"},
		"b": {"blue", "green"},
		"c": nil,
	} {
		img := e.newImage(t, reg.ID, name)
		img.Labels = labels
		require.NoError(t, e.images.Update(ctx, img))
	}

	labels, err := e.images.GetLabelsByParentIDAndRepo(ctx, reg.ParentID, reg.Name, 10, 0, "")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"red", "blue", "green"}, labels)

	labels, err = e.images.GetLabelsByParentIDAndRepo(ctx, reg.ParentID, reg.Name, 10, 0, "gre")
	require.NoError(t, err)
	// The search matches whole label sets, so all labels of the matching image are returned.
	require.ElementsMatch(t, []string{"blue", "green"}, labels)

	count, err := e.images.CountLabelsByParentIDAndRepo(ctx, reg.ParentID, reg.Name, "")
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
	count, err = e.images.CountLabelsByParentIDAndRepo(ctx, reg.ParentID, reg.Name, "blue")
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	// Deletion honours linked artifacts.
	linked := e.newImage(t, reg.ID, "linked")
	e.newArtifact(t, linked.ID, "1.0")
	require.NoError(t, e.images.DeleteByImageNameIfNoLinkedArtifacts(ctx, reg.ID, "linked"))
	_, err = e.images.Get(ctx, linked.ID)
	require.NoError(t, err)

	require.NoError(t, e.images.DeleteByImageNameIfNoLinkedArtifacts(ctx, reg.ID, "a"))
	_, err = e.images.GetByName(ctx, reg.ID, "a")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	require.NoError(t, e.images.DeleteByImageNameAndRegID(ctx, reg.ID, "b"))
	_, err = e.images.GetByName(ctx, reg.ID, "b")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// Hard deleting an image cascades to its artifacts.
	require.NoError(t, e.images.DeleteByImageNameAndRegID(ctx, reg.ID, "linked"))
	cnt, err := e.artifacts.Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt)
}

func TestRdbImage_Duplicate(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	src := e.newDockerRegistry(t, "src")
	dst := e.newDockerRegistry(t, "dst")
	model := artifact.ArtifactTypeModel

	source := &types.Image{Name: "web", RegistryID: src.ID, ArtifactType: &model, Enabled: true}
	require.NoError(t, e.images.CreateOrUpdate(ctx, source))

	dup, err := e.images.DuplicateImage(ctx, source, dst.ID)
	require.NoError(t, err)
	require.NotEqual(t, source.ID, dup.ID)
	require.Equal(t, dst.ID, dup.RegistryID)

	got, err := e.images.GetByNameAndType(ctx, dst.ID, "web", &model)
	require.NoError(t, err)
	require.Equal(t, dup.ID, got.ID)

	// Duplicating an image without a name fails.
	_, err = e.images.DuplicateImage(ctx, &types.Image{RegistryID: src.ID}, dst.ID)
	require.Error(t, err)
}
