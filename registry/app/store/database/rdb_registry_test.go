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

func TestRdbRegistry_CreateGetUpdateDelete(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx

	reg := &types.Registry{
		Name:            "docker-main",
		ParentID:        10,
		RootParentID:    rdbRootSpaceID,
		Description:     "main registry",
		Type:            artifact.RegistryTypeVIRTUAL,
		PackageType:     artifact.PackageTypeDOCKER,
		UpstreamProxies: []int64{3, 4},
		AllowedPattern:  []string{"a/*", "b/*"},
		BlockedPattern:  []string{"c/*"},
		Labels:          []string{"one", "two"},
		Config:          &types.RegistryConfig{RemoteUrlSuffix: "/suffix"},
	}
	id, err := e.registries.Create(ctx, reg)
	require.NoError(t, err)
	require.NotZero(t, id)
	require.NotEmpty(t, reg.UUID)

	got, err := e.registries.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "docker-main", got.Name)
	require.Equal(t, "main registry", got.Description)
	require.Equal(t, []int64{3, 4}, got.UpstreamProxies)
	require.Equal(t, []string{"a/*", "b/*"}, got.AllowedPattern)
	require.Equal(t, []string{"c/*"}, got.BlockedPattern)
	require.Equal(t, []string{"one", "two"}, got.Labels)
	require.NotNil(t, got.Config)
	require.Equal(t, "/suffix", got.Config.RemoteUrlSuffix)
	require.Equal(t, int64(7), got.CreatedBy)
	require.Nil(t, got.DeletedAt)

	byUUID, err := e.registries.GetByUUID(ctx, got.UUID)
	require.NoError(t, err)
	require.Equal(t, id, byUUID.ID)

	byParent, err := e.registries.GetByParentIDAndName(ctx, 10, "docker-main")
	require.NoError(t, err)
	require.Equal(t, id, byParent.ID)

	byRoot, err := e.registries.GetByRootParentIDAndName(ctx, rdbRootSpaceID, "docker-main")
	require.NoError(t, err)
	require.Equal(t, id, byRoot.ID)

	got.Description = "updated"
	got.Labels = []string{"three"}
	require.NoError(t, e.registries.Update(ctx, got))
	updated, err := e.registries.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "updated", updated.Description)
	require.Equal(t, []string{"three"}, updated.Labels)
	require.Equal(t, got.UUID, updated.UUID)

	cnt, err := e.registries.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)

	require.NoError(t, e.registries.Delete(ctx, 10, "docker-main"))
	_, err = e.registries.Get(ctx, id)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
}

func TestRdbRegistry_NotFoundAndConflicts(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx

	_, err := e.registries.Get(ctx, 999)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.registries.GetByUUID(ctx, "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.registries.GetByParentIDAndName(ctx, 1, "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = e.registries.GetByRootParentIDAndName(ctx, 1, "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	reg := e.newDockerRegistry(t, "dup")
	_, err = e.registries.Create(ctx, &types.Registry{
		Name: "dup", ParentID: rdbRootSpaceID, RootParentID: rdbRootSpaceID,
		Type: artifact.RegistryTypeVIRTUAL, PackageType: artifact.PackageTypeDOCKER,
	})
	require.ErrorIs(t, err, gstore.ErrDuplicate)

	// Update of a registry that does not exist reports a version conflict.
	missing := &types.Registry{ID: reg.ID + 100, Name: "x", ParentID: 1, RootParentID: 1}
	require.ErrorIs(t, e.registries.Update(ctx, missing), gstore.ErrVersionConflict)
}

func TestRdbRegistry_SoftDeleteFilters(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx

	live := e.newDockerRegistry(t, "live")
	gone := e.newDockerRegistry(t, "gone")
	e.softDelete(t, "registries", "registry", gone.ID)

	tests := []struct {
		name  string
		opts  []types.QueryOption
		found bool
	}{
		{"default excludes deleted", nil, false},
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
			_, err := e.registries.Get(ctx, gone.ID, tc.opts...)
			check(err)
			_, err = e.registries.GetByParentIDAndName(ctx, gone.ParentID, "gone", tc.opts...)
			check(err)
			_, err = e.registries.GetByRootParentIDAndName(ctx, rdbRootSpaceID, "gone", tc.opts...)
			check(err)
			list, err := e.registries.GetByIDIn(ctx, []int64{live.ID, gone.ID}, tc.opts...)
			require.NoError(t, err)
			if tc.found {
				require.GreaterOrEqual(t, len(*list), 1)
			} else {
				require.Len(t, *list, 1)
			}
		})
	}

	deletedGot, err := e.registries.Get(ctx, gone.ID, types.WithAllDeleted())
	require.NoError(t, err)
	require.NotNil(t, deletedGot.DeletedAt)

	cnt, err := e.registries.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)

	ids, err := e.registries.GetIDsByParentSpace(ctx, rdbRootSpaceID)
	require.NoError(t, err)
	require.Equal(t, []int64{live.ID}, ids)
	ids, err = e.registries.GetIDsByParentSpace(ctx, rdbRootSpaceID, types.WithOnlyDeleted())
	require.NoError(t, err)
	require.Equal(t, []int64{gone.ID}, ids)
	ids, err = e.registries.GetIDsByParentSpace(ctx, rdbRootSpaceID, types.WithAllDeleted())
	require.NoError(t, err)
	require.Len(t, ids, 2)
}

func TestRdbRegistry_FetchUpstreamProxyKeys(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx

	up1 := e.newRegistry(t, rdbRootSpaceID, "up-one", artifact.RegistryTypeUPSTREAM, artifact.PackageTypeDOCKER)
	up2 := e.newRegistry(t, rdbRootSpaceID, "up-two", artifact.RegistryTypeUPSTREAM, artifact.PackageTypeDOCKER)
	e.softDelete(t, "registries", "registry", up2.ID)

	keys, err := e.registries.FetchUpstreamProxyKeys(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, keys)

	keys, err = e.registries.FetchUpstreamProxyKeys(ctx, []int64{up1.ID, up2.ID, 9999})
	require.NoError(t, err)
	require.Equal(t, []string{"up-one", "", ""}, keys)

	keys, err = e.registries.FetchUpstreamProxyKeys(ctx, []int64{up2.ID, up1.ID}, types.WithAllDeleted())
	require.NoError(t, err)
	require.Equal(t, []string{"up-two", "up-one"}, keys)

	keys, err = e.registries.FetchUpstreamProxyKeys(ctx, []int64{up2.ID, up1.ID}, types.WithOnlyDeleted())
	require.NoError(t, err)
	require.Equal(t, []string{"up-two", ""}, keys)

	ids, err := e.registries.FetchUpstreamProxyIDs(ctx, []string{"up-one", "up-two", "missing"}, rdbRootSpaceID)
	require.NoError(t, err)
	require.Equal(t, []int64{up1.ID}, ids)
}

func TestRdbRegistry_FetchRegistriesIDByUpstreamProxyID(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx

	mk := func(name string, proxies []int64) *types.Registry {
		reg := &types.Registry{
			Name: name, ParentID: rdbRootSpaceID, RootParentID: rdbRootSpaceID,
			Type: artifact.RegistryTypeVIRTUAL, PackageType: artifact.PackageTypeDOCKER,
			UpstreamProxies: proxies,
		}
		id, err := e.registries.Create(ctx, reg)
		require.NoError(t, err)
		reg.ID = id
		return reg
	}
	only := mk("only", []int64{5})
	first := mk("first", []int64{5, 6})
	middle := mk("middle", []int64{4, 5, 6})
	last := mk("last", []int64{4, 5})
	mk("other", []int64{55})
	deleted := mk("deleted", []int64{5})
	e.softDelete(t, "registries", "registry", deleted.ID)

	ids, err := e.registries.FetchRegistriesIDByUpstreamProxyID(ctx, "5", rdbRootSpaceID)
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{only.ID, first.ID, middle.ID, last.ID}, ids)

	ids, err = e.registries.FetchRegistriesIDByUpstreamProxyID(ctx, "5", rdbRootSpaceID, types.WithAllDeleted())
	require.NoError(t, err)
	require.Len(t, ids, 5)

	ids, err = e.registries.FetchRegistriesIDByUpstreamProxyID(ctx, "5", rdbRootSpaceID, types.WithOnlyDeleted())
	require.NoError(t, err)
	require.Equal(t, []int64{deleted.ID}, ids)
}

func TestRdbRegistry_GetAllAndCountAll(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx

	up := e.newRegistry(t, 10, "alpha-up", artifact.RegistryTypeUPSTREAM, artifact.PackageTypeDOCKER)
	e.exec(t, `INSERT INTO upstream_proxy_configs (upstream_proxy_config_registry_id,
		upstream_proxy_config_source, upstream_proxy_config_url, upstream_proxy_config_auth_type)
		VALUES (?, 'Dockerhub', 'https://example.com', 'Anonymous')`, up.ID)
	docker := e.newRegistry(t, 10, "beta-docker", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeDOCKER)
	maven := e.newRegistry(t, 10, "gamma-maven", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	e.newRegistry(t, 11, "other-parent", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeDOCKER)
	gone := e.newRegistry(t, 10, "gone", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeDOCKER)
	e.softDelete(t, "registries", "registry", gone.ID)

	// Aggregates for the docker registry: images, OCI blobs and download counts.
	e.newImage(t, docker.ID, "img1")
	e.newImage(t, docker.ID, "img2")
	blob := e.newBlob(t, 500)
	require.NoError(t, NewRegistryBlobDao(e.db).LinkBlob(ctx, "img1", docker, blob.ID))
	e.dcf.byRegistry[docker.ID] = 42

	// Aggregate for the maven registry: generic blob via nodes.
	e.exec(t, `INSERT INTO generic_blobs (generic_blob_id, generic_blob_root_parent_id,
		generic_blob_sha_512, generic_blob_size, generic_blob_created_at, generic_blob_created_by)
		VALUES ('gb1', 100, 'sha512x', 900, 1, 1)`)
	e.exec(t, `INSERT INTO nodes (node_id, node_name, node_registry_id, node_is_file, node_path,
		node_generic_blob_id, node_created_at, node_created_by) VALUES ('n1', 'f', ?, true, '/f', 'gb1', 1, 1)`,
		maven.ID)

	res, err := e.registries.GetAll(ctx, []int64{10}, nil, "name", "ASC", 10, 0, "", "")
	require.NoError(t, err)
	require.Len(t, *res, 3)
	names := []string{(*res)[0].RegIdentifier, (*res)[1].RegIdentifier, (*res)[2].RegIdentifier}
	require.Equal(t, []string{"alpha-up", "beta-docker", "gamma-maven"}, names)
	require.Equal(t, "https://example.com", (*res)[0].URL)
	require.Equal(t, int64(2), (*res)[1].ArtifactCount)
	require.Equal(t, int64(500), (*res)[1].Size)
	require.Equal(t, int64(42), (*res)[1].DownloadCount)
	require.Equal(t, int64(900), (*res)[2].Size)

	res, err = e.registries.GetAll(ctx, []int64{10}, nil, "name", "DESC", 10, 0, "", "")
	require.NoError(t, err)
	require.Equal(t, "gamma-maven", (*res)[0].RegIdentifier)

	res, err = e.registries.GetAll(ctx, []int64{10}, []string{"MAVEN"}, "name", "ASC", 10, 0, "", "")
	require.NoError(t, err)
	require.Len(t, *res, 1)

	res, err = e.registries.GetAll(ctx, []int64{10}, nil, "name", "ASC", 10, 0, "", string(artifact.RegistryTypeUPSTREAM))
	require.NoError(t, err)
	require.Len(t, *res, 1)

	res, err = e.registries.GetAll(ctx, []int64{10}, nil, "name", "ASC", 1, 1, "a", "")
	require.NoError(t, err)
	require.Len(t, *res, 1)
	require.Equal(t, "beta-docker", (*res)[0].RegIdentifier)

	res, err = e.registries.GetAll(ctx, []int64{10}, nil, "name", "ASC", 10, 0, "nomatch", "")
	require.NoError(t, err)
	require.Empty(t, *res)

	_, err = e.registries.GetAll(ctx, []int64{10}, nil, "name", "ASC", -1, 0, "", "")
	require.Error(t, err)

	tests := []struct {
		name     string
		parents  []int64
		pkgTypes []string
		search   string
		repoType string
		want     int64
	}{
		{"all under parent", []int64{10}, nil, "", "", 3},
		{"multiple parents", []int64{10, 11}, nil, "", "", 4},
		{"package type", []int64{10}, []string{"DOCKER"}, "", "", 2},
		{"search", []int64{10}, nil, "gamma", "", 1},
		{"repo type", []int64{10}, nil, "", "UPSTREAM", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, cerr := e.registries.CountAll(ctx, tc.parents, tc.pkgTypes, tc.search, tc.repoType)
			require.NoError(t, cerr)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestRdbRegistry_UpdateParentSpace(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx

	e.newRegistry(t, 10, "r1", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeDOCKER)
	e.newRegistry(t, 10, "r2", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeDOCKER)
	e.newRegistry(t, 11, "r3", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeDOCKER)

	n, err := e.registries.UpdateParentSpace(ctx, 10, 20)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	ids, err := e.registries.GetIDsByParentSpace(ctx, 20)
	require.NoError(t, err)
	require.Len(t, ids, 2)

	n, err = e.registries.UpdateParentSpace(ctx, 10, 30)
	require.NoError(t, err)
	require.Zero(t, n)
}
