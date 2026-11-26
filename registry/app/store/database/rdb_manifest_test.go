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
	"database/sql"
	"testing"

	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	gstore "github.com/harness/gitness/store"

	"github.com/stretchr/testify/require"
)

func TestRdbManifest_CreateFindDelete(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newDockerRegistry(t, "docker")

	cfgBlob := e.newBlob(t, 5)
	m := &types.Manifest{
		RegistryID:    reg.ID,
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		ImageName:     "img",
		Digest:        rdbDigest(500),
		Payload:       types.Payload(`{"a":1}`),
		TotalSize:     77,
		ArtifactType:  sql.NullString{String: "application/x-art", Valid: true},
		Annotations:   types.JSONB{"k": "v"},
		Configuration: &types.Configuration{
			BlobID:    cfgBlob.ID,
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    cfgBlob.Digest,
			Payload:   types.Payload(`{"cfg":true}`),
		},
	}
	require.NoError(t, e.manifests.Create(ctx, m))
	require.NotZero(t, m.ID)

	dgst, err := types.NewDigest(m.Digest)
	require.NoError(t, err)

	byDigest, err := e.manifests.FindManifestByDigest(ctx, reg.ID, "img", dgst)
	require.NoError(t, err)
	require.Equal(t, m.ID, byDigest.ID)
	require.Equal(t, m.Digest, byDigest.Digest)
	require.Equal(t, int64(77), byDigest.TotalSize)
	require.Equal(t, "application/x-art", byDigest.ArtifactType.String)
	require.NotNil(t, byDigest.Configuration)
	require.Equal(t, cfgBlob.ID, byDigest.Configuration.BlobID)
	require.Equal(t, cfgBlob.Digest, byDigest.Configuration.Digest)

	byID, err := e.manifests.Get(ctx, m.ID)
	require.NoError(t, err)
	require.Equal(t, m.Digest, byID.Digest)
	require.Equal(t, "application/vnd.oci.image.manifest.v1+json", byID.MediaType)

	_, err = e.manifests.Get(ctx, m.ID+1000)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	payload, err := e.manifests.GetManifestPayload(ctx, reg.ParentID, reg.Name, "img", dgst)
	require.NoError(t, err)
	require.JSONEq(t, `{"a":1}`, string(*payload))

	_, err = e.manifests.GetManifestPayload(ctx, reg.ParentID, "nope", "img", dgst)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// Creating the identical manifest twice violates the unique constraint.
	dup := *m
	dup.ID = 0
	require.ErrorIs(t, e.manifests.Create(ctx, &dup), gstore.ErrDuplicate)

	count, err := e.manifests.CountByImageName(ctx, reg.ID, "img")
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	latest, err := e.manifests.GetLatestManifest(ctx, reg.ID, "img")
	require.NoError(t, err)
	require.Equal(t, m.ID, latest.ID)
	_, err = e.manifests.GetLatestManifest(ctx, reg.ID, "none")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	deleted, err := e.manifests.DeleteManifest(ctx, reg.ID, "img", m.Digest)
	require.NoError(t, err)
	require.True(t, deleted)
	deleted, err = e.manifests.DeleteManifest(ctx, reg.ID, "img", m.Digest)
	require.NoError(t, err)
	require.False(t, deleted)
	_, err = e.manifests.DeleteManifest(ctx, reg.ID, "img", "bogus")
	require.Error(t, err)
}

func TestRdbManifest_CreateOrFind(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newDockerRegistry(t, "docker")

	m := &types.Manifest{
		RegistryID: reg.ID, SchemaVersion: 2, ImageName: "img", Digest: rdbDigest(600),
		MediaType: "application/vnd.oci.image.manifest.v1+json", Payload: types.Payload(`{}`),
	}
	require.NoError(t, e.manifests.CreateOrFind(ctx, m))
	require.NotZero(t, m.ID)

	again := &types.Manifest{
		RegistryID: reg.ID, SchemaVersion: 2, ImageName: "img", Digest: m.Digest,
		MediaType: "application/vnd.oci.image.manifest.v1+json", Payload: types.Payload(`{}`),
	}
	require.NoError(t, e.manifests.CreateOrFind(ctx, again))
	require.Equal(t, m.ID, again.ID)

	require.Error(t, e.manifests.CreateOrFind(ctx, &types.Manifest{Digest: "bogus"}))
	require.ErrorIs(t, e.manifests.CreateOrFind(ctx, &types.Manifest{
		RegistryID: reg.ID, ImageName: "img", Digest: rdbDigest(601), MediaType: "application/x-unknown",
	}), gstore.ErrResourceNotFound)
	require.Error(t, e.manifests.Create(ctx, &types.Manifest{
		RegistryID: reg.ID, ImageName: "img", Digest: rdbDigest(602), MediaType: "application/x-unknown",
	}))
}

func TestRdbManifest_ReferencesAndSubjects(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newDockerRegistry(t, "docker")
	refDao := NewManifestReferenceDao(e.db)
	ociDao := NewOCIImageIndexMappingDao(e.db)

	parent := e.newManifest(t, reg.ID, "img")
	child1 := e.newManifest(t, reg.ID, "img")
	child2 := e.newManifest(t, reg.ID, "img")

	require.NoError(t, refDao.AssociateManifest(ctx, parent, child1))
	require.NoError(t, refDao.AssociateManifest(ctx, parent, child2))
	// Associating twice is idempotent.
	require.NoError(t, refDao.AssociateManifest(ctx, parent, child1))
	require.Error(t, refDao.AssociateManifest(ctx, parent, parent))

	missing := &types.Manifest{ID: child2.ID + 1000, RegistryID: reg.ID}
	require.ErrorIs(t, refDao.AssociateManifest(ctx, parent, missing), util.ErrRefManifestNotFound)

	refs, err := e.manifests.References(ctx, parent)
	require.NoError(t, err)
	require.Len(t, refs, 2)

	parents, err := e.manifests.ReferencedBy(ctx, child1)
	require.NoError(t, err)
	require.Len(t, parents, 1)
	require.Equal(t, parent.ID, parents[0].ID)

	parents, err = e.manifests.ReferencedBy(ctx, parent)
	require.NoError(t, err)
	require.Empty(t, parents)

	require.Panics(t, func() { _ = refDao.DissociateManifest(ctx, parent, child1) })

	// OCI image index mappings.
	require.NoError(t, ociDao.Create(ctx, &types.OCIImageIndexMapping{
		ParentManifestID: parent.ID, ChildManifestDigest: child1.Digest,
	}))
	childDgst, err := types.NewDigest(child1.Digest)
	require.NoError(t, err)
	mappings, err := ociDao.GetAllByChildDigest(ctx, reg.ID, "img", childDgst)
	require.NoError(t, err)
	require.Len(t, mappings, 1)
	require.Equal(t, parent.ID, mappings[0].ParentManifestID)
	require.Equal(t, child1.Digest, mappings[0].ChildManifestDigest)

	mappings, err = ociDao.GetAllByChildDigest(ctx, reg.ID, "other", childDgst)
	require.NoError(t, err)
	require.Empty(t, mappings)

	_, err = ociDao.GetAllByChildDigest(ctx, reg.ID, "img", "zz-not-hex")
	require.Error(t, err)

	// Subjects.
	subj := e.newManifest(t, reg.ID, "img")
	referrer := &types.Manifest{
		RegistryID: reg.ID, SchemaVersion: 2, ImageName: "img", Digest: rdbDigest(700),
		MediaType: "application/vnd.oci.image.manifest.v1+json", Payload: types.Payload(`{}`),
		SubjectID: sql.NullInt64{Int64: subj.ID, Valid: true}, SubjectDigest: subj.Digest,
	}
	require.NoError(t, e.manifests.Create(ctx, referrer))

	bySubject, err := e.manifests.ListManifestsBySubject(ctx, reg.ID, subj.ID)
	require.NoError(t, err)
	require.Len(t, bySubject, 1)
	require.Equal(t, referrer.ID, bySubject[0].ID)

	subjDgst, err := types.NewDigest(subj.Digest)
	require.NoError(t, err)
	bySubjectDigest, err := e.manifests.ListManifestsBySubjectDigest(ctx, reg.ID, subjDgst)
	require.NoError(t, err)
	require.Len(t, bySubjectDigest, 1)
	_, err = e.manifests.ListManifestsBySubjectDigest(ctx, reg.ID, "zz-not-hex")
	require.Error(t, err)
}

func TestRdbManifest_TagLookupsAndDeletes(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newDockerRegistry(t, "docker")

	m := e.newManifest(t, reg.ID, "img")
	e.newTag(t, reg.ID, "img", "v1", m.ID)

	byTag, err := e.manifests.FindManifestByTagName(ctx, reg.ID, "img", "v1")
	require.NoError(t, err)
	require.Equal(t, m.ID, byTag.ID)
	_, err = e.manifests.FindManifestByTagName(ctx, reg.ID, "img", "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	dgst, err := e.manifests.FindManifestDigestByTagName(ctx, reg.ID, "img", "v1")
	require.NoError(t, err)
	expected, err := types.NewDigest(m.Digest)
	require.NoError(t, err)
	require.Equal(t, expected, dgst)
	_, err = e.manifests.FindManifestDigestByTagName(ctx, reg.ID, "img", "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	payload, err := e.manifests.FindManifestPayloadByTagName(ctx, reg.ParentID, reg.Name, "img", "v1")
	require.NoError(t, err)
	require.JSONEq(t, `{"schemaVersion":2}`, string(*payload))
	_, err = e.manifests.FindManifestPayloadByTagName(ctx, reg.ParentID, reg.Name, "img", "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// Delete by id: missing is a no-op, existing is removed.
	require.NoError(t, e.manifests.Delete(ctx, reg.ID, m.ID+1000))
	require.NoError(t, e.manifests.Delete(ctx, reg.ID, m.ID))
	_, err = e.manifests.Get(ctx, m.ID)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	e.newManifest(t, reg.ID, "bulk")
	e.newManifest(t, reg.ID, "bulk")
	removed, err := e.manifests.DeleteManifestByImageName(ctx, reg.ID, "bulk")
	require.NoError(t, err)
	require.True(t, removed)
	removed, err = e.manifests.DeleteManifestByImageName(ctx, reg.ID, "bulk")
	require.NoError(t, err)
	require.False(t, removed)

	require.Panics(t, func() { _, _ = e.manifests.FindAll(ctx) })
	require.Panics(t, func() { _, _ = e.manifests.Count(ctx) })
	require.Panics(t, func() { _, _ = e.manifests.LayerBlobs(ctx, m) })
	require.Panics(t, func() { _ = e.manifests.AssociateLayerBlob(ctx, m, nil) })
	require.Panics(t, func() { _ = e.manifests.DissociateLayerBlob(ctx, m, nil) })
}

func TestRdbLayers(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newDockerRegistry(t, "docker")
	layers := NewLayersDao(e.db, e.mediaTypes)

	m := e.newManifest(t, reg.ID, "img")
	b1 := e.newBlob(t, 11)
	b2 := e.newBlob(t, 22)

	require.NoError(t, layers.AssociateLayerBlob(ctx, m, b1))
	require.NoError(t, layers.AssociateLayerBlob(ctx, m, b2))
	// Associating again is idempotent.
	require.NoError(t, layers.AssociateLayerBlob(ctx, m, b1))

	got, err := layers.GetAllLayersByManifestID(ctx, m.ID)
	require.NoError(t, err)
	require.Len(t, *got, 2)
	require.Equal(t, int64(11), (*got)[0].Size)
	require.Equal(t, b1.ID, (*got)[0].BlobID)
	require.Equal(t, m.ID, (*got)[0].ManifestID)

	empty, err := layers.GetAllLayersByManifestID(ctx, m.ID+1000)
	require.NoError(t, err)
	require.Empty(t, *empty)

	// A layer for a manifest that does not exist violates the foreign key.
	ghost := &types.Manifest{ID: m.ID + 1000, RegistryID: reg.ID}
	require.ErrorIs(t, layers.AssociateLayerBlob(ctx, ghost, b1), util.ErrRefManifestNotFound)

	// Unknown media types cannot be mapped.
	require.Error(t, layers.AssociateLayerBlob(ctx, m, &types.Blob{ID: b1.ID, MediaType: "application/x-unknown"}))
}
