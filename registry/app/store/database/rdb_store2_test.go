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
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	appcache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	gstore "github.com/harness/gitness/store"
	gtypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestRdbNode_CreateAndLookups(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newRegistry(t, rdbRootSpaceID, "generic", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeGENERIC)
	other := e.newRegistry(t, rdbRootSpaceID, "generic-2", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeGENERIC)
	nodes := NewNodeDao(e.db)
	blobs := NewGenericBlobDao(e.db)

	blobID, _, err := blobs.Create(ctx, &types.GenericBlob{
		RootParentID: rdbRootSpaceID, Sha256: "sha-a", Sha512: "sha512-a", Sha1: "sha1-a", MD5: "md5-a", Size: 42,
	})
	require.NoError(t, err)
	blob2ID, _, err := blobs.Create(ctx, &types.GenericBlob{
		RootParentID: rdbRootSpaceID, Sha256: "sha-b", Sha512: "sha512-b", Size: 8,
	})
	require.NoError(t, err)

	dir := &types.Node{Name: "pkg", RegistryID: reg.ID, NodePath: "/pkg", IsFile: false}
	require.NoError(t, nodes.Create(ctx, dir))
	require.NotEmpty(t, dir.ID)

	file := &types.Node{
		Name: "a.txt", RegistryID: reg.ID, NodePath: "/pkg/1.0/a.txt", IsFile: true,
		ParentNodeID: dir.ID, BlobID: blobID,
	}
	require.NoError(t, nodes.Create(ctx, file))
	file2 := &types.Node{Name: "b.txt", RegistryID: reg.ID, NodePath: "/pkg/2.0/b.txt", IsFile: true, BlobID: blob2ID}
	require.NoError(t, nodes.Create(ctx, file2))
	require.NoError(t, nodes.Create(ctx, &types.Node{
		Name: "a.txt", RegistryID: other.ID, NodePath: "/pkg/1.0/a.txt", IsFile: true, BlobID: blobID,
	}))

	got, err := nodes.Get(ctx, file.ID)
	require.NoError(t, err)
	require.Equal(t, "a.txt", got.Name)
	require.Equal(t, dir.ID, got.ParentNodeID)
	require.Equal(t, blobID, got.BlobID)
	require.True(t, got.IsFile)

	got, err = nodes.Get(ctx, dir.ID)
	require.NoError(t, err)
	require.Empty(t, got.BlobID)
	require.Empty(t, got.ParentNodeID)

	_, err = nodes.Get(ctx, "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	byPath, err := nodes.GetByPathAndRegistryID(ctx, reg.ID, "/pkg/1.0/a.txt")
	require.NoError(t, err)
	require.Equal(t, file.ID, byPath.ID)
	_, err = nodes.GetByPathAndRegistryID(ctx, reg.ID, "/nope")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	byName, err := nodes.GetByNameAndRegistryID(ctx, reg.ID, "b.txt")
	require.NoError(t, err)
	require.Equal(t, file2.ID, byName.ID)
	_, err = nodes.GetByNameAndRegistryID(ctx, reg.ID, "nope")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	byBlob, err := nodes.GetByBlobIDAndRegistryID(ctx, blob2ID, reg.ID)
	require.NoError(t, err)
	require.Equal(t, file2.ID, byBlob.ID)
	_, err = nodes.GetByBlobIDAndRegistryID(ctx, "missing", reg.ID)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	found, err := nodes.FindByPathAndRegistryID(ctx, reg.ID, "/pkg", "b.txt")
	require.NoError(t, err)
	require.Equal(t, file2.ID, found.ID)
	_, err = nodes.FindByPathAndRegistryID(ctx, reg.ID, "/pkg", "nope.txt")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	ids, err := nodes.FindByPathsAndRegistryID(ctx, []string{"/pkg", "/pkg/1.0/a.txt", "/none"}, reg.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{dir.ID, file.ID}, *ids)

	cnt, err := nodes.CountByPathAndRegistryID(ctx, reg.ID, "/pkg/%")
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt)
	cnt, err = nodes.CountByPathAndRegistryID(ctx, reg.ID, "/other/%")
	require.NoError(t, err)
	require.Zero(t, cnt)

	// Creating a node at an existing path re-links the blob but keeps the id.
	again := &types.Node{Name: "a.txt", RegistryID: reg.ID, NodePath: "/pkg/1.0/a.txt", IsFile: true, BlobID: blob2ID}
	require.NoError(t, nodes.Create(ctx, again))
	require.Equal(t, file.ID, again.ID)
	got, err = nodes.Get(ctx, file.ID)
	require.NoError(t, err)
	require.Equal(t, blob2ID, got.BlobID)

	require.Panics(t, func() { _ = nodes.DeleteByID(ctx, 1) })
}

func TestRdbNode_MetadataAndDeletes(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newRegistry(t, rdbRootSpaceID, "generic", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeGENERIC)
	nodes := NewNodeDao(e.db)
	blobs := NewGenericBlobDao(e.db)

	var blobIDs []string
	for _, b := range []types.GenericBlob{
		{Sha256: "s1", Sha512: "x1", Sha1: "h1", MD5: "m1", Size: 10},
		{Sha256: "s2", Sha512: "x2", Sha1: "h2", MD5: "m2", Size: 20},
		{Sha256: "s3", Sha512: "x3", Sha1: "h3", MD5: "m3", Size: 30},
	} {
		blob := b
		blob.RootParentID = rdbRootSpaceID
		id, _, err := blobs.Create(ctx, &blob)
		require.NoError(t, err)
		blobIDs = append(blobIDs, id)
	}

	paths := []string{"/lib/1.0/one.jar", "/lib/1.0/two.jar", "/lib/2.0/three.jar"}
	for i, p := range paths {
		fileName := []string{"one.jar", "two.jar", "three.jar"}[i]
		require.NoError(t, nodes.Create(ctx, &types.Node{
			Name: fileName, RegistryID: reg.ID, NodePath: p, IsFile: true, BlobID: blobIDs[i],
		}))
	}

	list, err := nodes.GetFilesMetadataByPathAndRegistryID(ctx, reg.ID, "/lib/1.0/%", "n.node_name", "ASC", 10, 0, "")
	require.NoError(t, err)
	require.Len(t, *list, 2)
	require.Equal(t, "one.jar", (*list)[0].Name)
	require.Equal(t, int64(10), (*list)[0].Size)
	require.Equal(t, "s1", (*list)[0].Sha256)
	require.Equal(t, "m1", (*list)[0].MD5)

	list, err = nodes.GetFilesMetadataByPathAndRegistryID(ctx, reg.ID, "/lib/%", "n.node_name", "DESC", 1, 1, "")
	require.NoError(t, err)
	require.Len(t, *list, 1)

	list, err = nodes.GetFilesMetadataByPathAndRegistryID(ctx, reg.ID, "/lib/%", "n.node_name", "ASC", 10, 0, "three")
	require.NoError(t, err)
	require.Len(t, *list, 1)
	require.Equal(t, "three.jar", (*list)[0].Name)

	single, err := nodes.GetFileMetadataByPathAndRegistryID(ctx, reg.ID, "/lib/2.0/three.jar")
	require.NoError(t, err)
	require.Equal(t, "three.jar", single.Name)
	require.Equal(t, int64(30), single.Size)
	require.Equal(t, "/lib/2.0/three.jar", single.Path)
	_, err = nodes.GetFileMetadataByPathAndRegistryID(ctx, reg.ID, "/lib/9.9/none.jar")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	files, err := nodes.GetAllFileNodesByPathPrefixAndRegistryID(ctx, reg.ID, "/lib/1.0")
	require.NoError(t, err)
	require.Len(t, *files, 2)
	files, err = nodes.GetAllFileNodesByPathPrefixAndRegistryID(ctx, reg.ID, "/lib/2.0/three.jar")
	require.NoError(t, err)
	require.Len(t, *files, 1)

	require.NoError(t, nodes.DeleteByLeafNodePathAndRegistryID(ctx, "/lib/2.0/three.jar", reg.ID))
	files, err = nodes.GetAllFileNodesByPathPrefixAndRegistryID(ctx, reg.ID, "/lib")
	require.NoError(t, err)
	require.Len(t, *files, 2)
	// Deleting a path that does not exist is a no-op.
	require.NoError(t, nodes.DeleteByLeafNodePathAndRegistryID(ctx, "/lib/none", reg.ID))

	require.NoError(t, nodes.DeleteByNodePathAndRegistryID(ctx, "/lib/1.0", reg.ID))
	files, err = nodes.GetAllFileNodesByPathPrefixAndRegistryID(ctx, reg.ID, "/lib")
	require.NoError(t, err)
	require.Empty(t, *files)
}

func TestRdbQuarantineArtifact(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	reg := e.newRegistry(t, rdbRootSpaceID, "maven", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeMAVEN)
	img := e.newImage(t, reg.ID, "lib")
	art := e.newArtifact(t, img.ID, "1.0")
	dao := NewQuarantineArtifactDao(e.db)
	nodes := NewNodeDao(e.db)

	node := &types.Node{Name: "lib.jar", RegistryID: reg.ID, NodePath: "/lib/1.0/lib.jar", IsFile: true}
	require.NoError(t, nodes.Create(ctx, node))

	// Quarantine at the artifact level.
	require.NoError(t, dao.Create(ctx, &types.QuarantineArtifact{
		RegistryID: reg.ID, ImageID: img.ID, ArtifactID: art.ID, Reason: "vulnerable",
	}))

	res, err := dao.GetByFilePath(ctx, "/lib/1.0/lib.jar", reg.ID, "lib", "1.0", nil)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, "vulnerable", res[0].Reason)
	require.Equal(t, art.ID, res[0].ArtifactID)
	require.Equal(t, img.ID, res[0].ImageID)
	require.Nil(t, res[0].NodeID)
	require.Equal(t, int64(7), res[0].CreatedBy)

	res, err = dao.GetByFilePath(ctx, "/lib/1.0/lib.jar", reg.ID, "lib", "2.0", nil)
	require.NoError(t, err)
	require.Empty(t, res)

	model := artifact.ArtifactTypeModel
	res, err = dao.GetByFilePath(ctx, "/lib/1.0/lib.jar", reg.ID, "lib", "1.0", &model)
	require.NoError(t, err)
	require.Empty(t, res)

	// Quarantine a single file.
	require.NoError(t, dao.Create(ctx, &types.QuarantineArtifact{
		RegistryID: reg.ID, ImageID: img.ID, ArtifactID: art.ID, NodeID: &node.ID, Reason: "bad file",
	}))
	res, err = dao.GetByFilePath(ctx, "/lib/1.0/lib.jar", reg.ID, "lib", "1.0", nil)
	require.NoError(t, err)
	require.Len(t, res, 2)
	res, err = dao.GetByFilePath(ctx, "/lib/1.0/other.jar", reg.ID, "lib", "1.0", nil)
	require.NoError(t, err)
	require.Len(t, res, 1)

	require.NoError(t, dao.DeleteByRegistryIDArtifactAndFilePath(ctx, reg.ID, &art.ID, img.ID, &node.ID))
	res, err = dao.GetByFilePath(ctx, "/lib/1.0/lib.jar", reg.ID, "lib", "1.0", nil)
	require.NoError(t, err)
	require.Len(t, res, 1)

	require.NoError(t, dao.DeleteByRegistryIDArtifactAndFilePath(ctx, reg.ID, nil, img.ID, nil))
	res, err = dao.GetByFilePath(ctx, "/lib/1.0/lib.jar", reg.ID, "lib", "1.0", nil)
	require.NoError(t, err)
	require.Empty(t, res)

	// Without an auth session the creator defaults to zero.
	require.NoError(t, dao.Create(rdbBackground(), &types.QuarantineArtifact{
		RegistryID: reg.ID, ImageID: img.ID, Reason: "image level",
	}))
}

// rdbFakeSpaceIDCache resolves space ids from an in-memory map.
type rdbFakeSpaceIDCache struct {
	spaces map[int64]*gtypes.SpaceCore
}

func (c *rdbFakeSpaceIDCache) Stats() (int64, int64) { return 0, 0 }

func (c *rdbFakeSpaceIDCache) Get(_ context.Context, key int64) (*gtypes.SpaceCore, error) {
	if s, ok := c.spaces[key]; ok {
		return s, nil
	}
	return nil, gstore.ErrResourceNotFound
}

func (c *rdbFakeSpaceIDCache) Evict(_ context.Context, _ int64) {}

func rdbNewSpaceFinder() refcache.SpaceFinder {
	return refcache.NewSpaceFinder(&rdbFakeSpaceIDCache{spaces: map[int64]*gtypes.SpaceCore{
		31: {ID: 31, Path: "acme/secrets"},
		32: {ID: 32, Path: "acme/users"},
	}}, nil, nil, appcache.Evictor[*gtypes.SpaceCore]{})
}

func TestRdbUpstreamProxy_CreateGetUpdate(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	dao := NewUpstreamproxyDao(e.db, e.registries, rdbNewSpaceFinder())

	reg := &types.Registry{
		Name: "up-docker", ParentID: 10, RootParentID: rdbRootSpaceID, Description: "proxy",
		Type: artifact.RegistryTypeUPSTREAM, PackageType: artifact.PackageTypeDOCKER,
		Labels: []string{"x"}, AllowedPattern: []string{"a/*"}, BlockedPattern: []string{"b/*"},
		Config: &types.RegistryConfig{RemoteUrlSuffix: "/v2"},
	}
	regID, err := e.registries.Create(ctx, reg)
	require.NoError(t, err)

	cfg := &types.UpstreamProxyConfig{
		RegistryID: regID, Source: "Dockerhub", URL: "https://registry.example.com",
		FirewallMode: "BLOCK", AuthType: "UserPassword", UserName: "bob",
		SecretIdentifier: "pw", SecretSpaceID: 31,
		UserNameSecretIdentifier: "user", UserNameSecretSpaceID: 32,
	}
	_, err = dao.Create(ctx, cfg)
	require.NoError(t, err)
	require.NotZero(t, cfg.ID)

	got, err := dao.Get(ctx, regID)
	require.NoError(t, err)
	require.Equal(t, "up-docker", got.RepoKey)
	require.Equal(t, "https://registry.example.com", got.RepoURL)
	require.Equal(t, "Dockerhub", got.Source)
	require.Equal(t, "UserPassword", got.RepoAuthType)
	require.Equal(t, "bob", got.UserName)
	require.NotNil(t, got.FirewallMode)
	require.Equal(t, "BLOCK", *got.FirewallMode)
	require.Equal(t, "pw", got.SecretIdentifier)
	require.Equal(t, int64(31), got.SecretSpaceID)
	require.Equal(t, "acme/secrets", got.SecretSpacePath)
	require.Equal(t, "user", got.UserNameSecretIdentifier)
	require.Equal(t, int64(32), got.UserNameSecretSpaceID)
	require.Equal(t, "acme/users", got.UserNameSecretSpacePath)
	require.Equal(t, []string{"x"}, got.Labels)
	require.Equal(t, []string{"a/*"}, got.AllowedPattern)
	require.Equal(t, []string{"b/*"}, got.BlockedPattern)
	require.NotNil(t, got.Config)
	require.Equal(t, "/v2", got.Config.RemoteUrlSuffix)
	require.Equal(t, "proxy", got.Description)
	require.Equal(t, "10", got.ParentID)

	byIdentifier, err := dao.GetByRegistryIdentifier(ctx, 10, "up-docker")
	require.NoError(t, err)
	require.Equal(t, regID, byIdentifier.RegistryID)
	_, err = dao.GetByRegistryIdentifier(ctx, 10, "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	byParent, err := dao.GetByParentID(ctx, "10")
	require.NoError(t, err)
	require.Len(t, *byParent, 1)

	// Update the config (addressed by its own id) and re-read.
	cfg.ID = got.ID
	cfg.URL = "https://other.example.com"
	cfg.Token = "tok"
	require.NoError(t, dao.Update(ctx, cfg))
	got, err = dao.Get(ctx, regID)
	require.NoError(t, err)
	require.Equal(t, "https://other.example.com", got.RepoURL)
	require.Equal(t, "tok", got.Token)

	require.ErrorIs(t, dao.Update(ctx, &types.UpstreamProxyConfig{ID: 9999, RegistryID: regID, AuthType: "Anonymous"}),
		gstore.ErrVersionConflict)

	// Soft deleted registries are filtered by the query options.
	e.softDelete(t, "registries", "registry", regID)
	_, err = dao.Get(ctx, regID)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = dao.Get(ctx, regID, types.WithOnlyDeleted())
	require.NoError(t, err)
	_, err = dao.Get(ctx, regID, types.WithAllDeleted())
	require.NoError(t, err)
	_, err = dao.GetByRegistryIdentifier(ctx, 10, "up-docker")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = dao.GetByRegistryIdentifier(ctx, 10, "up-docker", types.WithOnlyDeleted())
	require.NoError(t, err)
	_, err = dao.GetByRegistryIdentifier(ctx, 10, "up-docker", types.WithAllDeleted())
	require.NoError(t, err)
	byParent, err = dao.GetByParentID(ctx, "10")
	require.NoError(t, err)
	require.Empty(t, *byParent)
}

func TestRdbUpstreamProxy_ListAndSecretSpaces(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	dao := NewUpstreamproxyDao(e.db, e.registries, rdbNewSpaceFinder())

	for _, name := range []string{"up-alpha", "up-beta", "up-gamma"} {
		regID := e.newRegistry(t, 10, name, artifact.RegistryTypeUPSTREAM, artifact.PackageTypeDOCKER).ID
		_, err := dao.Create(ctx, &types.UpstreamProxyConfig{
			RegistryID: regID, Source: "Dockerhub", URL: "https://" + name + ".example.com", AuthType: "Anonymous",
			SecretIdentifier: "pw", SecretSpaceID: 31, UserNameSecretIdentifier: "user", UserNameSecretSpaceID: 31,
		})
		require.NoError(t, err)
	}
	// A virtual registry in the same space is ignored.
	e.newRegistry(t, 10, "virtual", artifact.RegistryTypeVIRTUAL, artifact.PackageTypeDOCKER)

	all, err := dao.GetAll(ctx, 10, nil, "name", "ASC", 10, 0, "")
	require.NoError(t, err)
	require.Len(t, *all, 3)
	require.Equal(t, "up-alpha", (*all)[0].RepoKey)

	page, err := dao.GetAll(ctx, 10, nil, "name", "DESC", 1, 1, "")
	require.NoError(t, err)
	require.Len(t, *page, 1)
	require.Equal(t, "up-beta", (*page)[0].RepoKey)

	searched, err := dao.GetAll(ctx, 10, nil, "name", "ASC", 10, 0, "gamma")
	require.NoError(t, err)
	require.Len(t, *searched, 1)

	count, err := dao.CountAll(ctx, "10", nil, "")
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
	count, err = dao.CountAll(ctx, "10", nil, "alpha")
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	n, err := dao.UpdateSecretSpaceID(ctx, 31, 32)
	require.NoError(t, err)
	require.Equal(t, int64(3), n)
	n, err = dao.UpdateSecretSpaceID(ctx, 31, 32)
	require.NoError(t, err)
	require.Zero(t, n)

	n, err = dao.UpdateUserNameSecretSpaceID(ctx, 31, 32)
	require.NoError(t, err)
	require.Equal(t, int64(3), n)

	moved, err := dao.GetByRegistryIdentifier(ctx, 10, "up-alpha")
	require.NoError(t, err)
	require.Equal(t, int64(32), moved.SecretSpaceID)
	require.Equal(t, "acme/users", moved.SecretSpacePath)
	require.Equal(t, int64(32), moved.UserNameSecretSpaceID)

	// Resolving a secret space that is unknown surfaces an error.
	e.exec(t, "UPDATE upstream_proxy_configs SET upstream_proxy_config_secret_space_id = 999")
	_, err = dao.GetByRegistryIdentifier(ctx, 10, "up-alpha")
	require.Error(t, err)
}

func rdbWebhook(registryID int64, identifier string) *gtypes.WebhookCore {
	return &gtypes.WebhookCore{
		ParentID:    registryID,
		ParentType:  enum.WebhookParentRegistry,
		CreatedBy:   7,
		Identifier:  identifier,
		DisplayName: "hook " + identifier,
		Description: "desc",
		URL:         "https://hooks.example.com/" + identifier,
		Enabled:     true,
		Triggers:    []enum.WebhookTrigger{enum.WebhookTriggerArtifactCreated, enum.WebhookTriggerArtifactDeleted},
		ExtraHeaders: []gtypes.ExtraHeader{
			{Key: "X-One", Value: "1"}, {Key: "X-Two", Value: "2", Masked: true},
		},
		SecretIdentifier: "sec",
		SecretSpaceID:    31,
	}
}

func rdbInsertPrincipal(t *testing.T, e *rdbEnv) {
	t.Helper()
	e.exec(t, `INSERT INTO principals (principal_id, principal_uid, principal_uid_unique, principal_type)
		VALUES (7, 'u7', 'u7', 'user')`)
}

func TestRdbWebhook_CRUD(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	rdbInsertPrincipal(t, e)
	reg := e.newDockerRegistry(t, "docker")
	dao := NewWebhookDao(e.db)

	hook := rdbWebhook(reg.ID, "first")
	require.NoError(t, dao.Create(ctx, hook))
	require.NotZero(t, hook.ID)

	got, err := dao.Find(ctx, hook.ID)
	require.NoError(t, err)
	require.Equal(t, "first", got.Identifier)
	require.Equal(t, "hook first", got.DisplayName)
	require.Equal(t, enum.WebhookParentRegistry, got.ParentType)
	require.Equal(t, reg.ID, got.ParentID)
	require.Equal(t, enum.WebhookTypeExternal, got.Type)
	require.Equal(t, []enum.WebhookTrigger{enum.WebhookTriggerArtifactCreated, enum.WebhookTriggerArtifactDeleted},
		got.Triggers)
	require.Len(t, got.ExtraHeaders, 2)
	require.True(t, got.ExtraHeaders[1].Masked)
	require.Equal(t, "sec", got.SecretIdentifier)
	require.Equal(t, int64(31), got.SecretSpaceID)
	require.Nil(t, got.LatestExecutionResult)
	require.True(t, got.Enabled)

	byIdentifier, err := dao.GetByRegistryAndIdentifier(ctx, reg.ID, "first")
	require.NoError(t, err)
	require.Equal(t, hook.ID, byIdentifier.ID)
	_, err = dao.GetByRegistryAndIdentifier(ctx, reg.ID, "missing")
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	_, err = dao.Find(ctx, 9999)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// Duplicate identifiers within a registry are rejected.
	require.ErrorIs(t, dao.Create(ctx, rdbWebhook(reg.ID, "first")), gstore.ErrDuplicate)

	// An internal webhook round-trips its type.
	internal := rdbWebhook(reg.ID, "internal")
	internal.Type = enum.WebhookTypeInternal
	require.NoError(t, dao.Create(ctx, internal))
	got, err = dao.Find(ctx, internal.ID)
	require.NoError(t, err)
	require.Equal(t, enum.WebhookTypeInternal, got.Type)

	// Update changes the editable fields and bumps the version.
	got, err = dao.Find(ctx, hook.ID)
	require.NoError(t, err)
	got.DisplayName = "renamed"
	got.Enabled = false
	result := enum.WebhookExecutionResultSuccess
	got.LatestExecutionResult = &result
	require.NoError(t, dao.Update(ctx, got))
	updated, err := dao.Find(ctx, hook.ID)
	require.NoError(t, err)
	require.Equal(t, "renamed", updated.DisplayName)
	require.False(t, updated.Enabled)
	require.Equal(t, int64(1), updated.Version)
	require.NotNil(t, updated.LatestExecutionResult)
	require.Equal(t, result, *updated.LatestExecutionResult)

	missing := rdbWebhook(reg.ID, "missing")
	require.ErrorIs(t, dao.Update(ctx, missing), gstore.ErrVersionConflict)

	require.NoError(t, dao.DeleteByRegistryAndIdentifier(ctx, reg.ID, "internal"))
	_, err = dao.Find(ctx, internal.ID)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// Space level webhooks use the space id column.
	spaceHook := rdbWebhook(10, "space-hook")
	spaceHook.ParentType = enum.WebhookParentSpace
	require.NoError(t, dao.Create(ctx, spaceHook))
	got, err = dao.Find(ctx, spaceHook.ID)
	require.NoError(t, err)
	require.Equal(t, enum.WebhookParentSpace, got.ParentType)
	require.Equal(t, int64(10), got.ParentID)
}

func TestRdbWebhook_ListCountAndSpaces(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	rdbInsertPrincipal(t, e)
	reg := e.newDockerRegistry(t, "docker")
	dao := NewWebhookDao(e.db)

	for _, id := range []string{"alpha", "beta", "gamma"} {
		require.NoError(t, dao.Create(ctx, rdbWebhook(reg.ID, id)))
	}
	spaceHook := rdbWebhook(10, "space-hook")
	spaceHook.ParentType = enum.WebhookParentSpace
	require.NoError(t, dao.Create(ctx, spaceHook))

	list, err := dao.ListByRegistry(ctx, "name", "DESC", 10, 0, "", reg.ID)
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.Equal(t, "gamma", list[0].Identifier)

	list, err = dao.ListByRegistry(ctx, "unknown", "ASC", 2, 1, "", reg.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	list, err = dao.ListByRegistry(ctx, "name", "ASC", 10, 0, "bet", reg.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "beta", list[0].Identifier)

	cnt, err := dao.CountAllByRegistry(ctx, reg.ID, "")
	require.NoError(t, err)
	require.Equal(t, int64(3), cnt)
	cnt, err = dao.CountAllByRegistry(ctx, reg.ID, "alp")
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)

	all, err := dao.ListAllByRegistry(ctx, []gtypes.WebhookParentInfo{
		{Type: enum.WebhookParentRegistry, ID: reg.ID},
		{Type: enum.WebhookParentSpace, ID: 10},
	})
	require.NoError(t, err)
	require.Len(t, all, 4)

	onlySpace, err := dao.ListAllByRegistry(ctx, []gtypes.WebhookParentInfo{
		{Type: enum.WebhookParentSpace, ID: 10},
	})
	require.NoError(t, err)
	require.Len(t, onlySpace, 1)

	_, err = dao.ListAllByRegistry(ctx, []gtypes.WebhookParentInfo{{Type: enum.WebhookParent("bogus"), ID: 1}})
	require.Error(t, err)

	moved, err := dao.UpdateParentSpace(ctx, 10, 11)
	require.NoError(t, err)
	require.Equal(t, int64(1), moved)
	moved, err = dao.UpdateParentSpace(ctx, 10, 11)
	require.NoError(t, err)
	require.Zero(t, moved)

	moved, err = dao.UpdateSecretSpaceID(ctx, 31, 32)
	require.NoError(t, err)
	require.Equal(t, int64(4), moved)
}

func TestRdbWebhook_UpdateOptLock(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	rdbInsertPrincipal(t, e)
	reg := e.newDockerRegistry(t, "docker")
	dao := NewWebhookDao(e.db)

	hook := rdbWebhook(reg.ID, "locked")
	require.NoError(t, dao.Create(ctx, hook))

	updated, err := dao.UpdateOptLock(ctx, hook, func(h *gtypes.WebhookCore) error {
		h.Description = "changed"
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, "changed", updated.Description)

	stored, err := dao.Find(ctx, hook.ID)
	require.NoError(t, err)
	require.Equal(t, "changed", stored.Description)

	// Errors from the mutate function are wrapped and returned.
	_, err = dao.UpdateOptLock(ctx, hook, func(*gtypes.WebhookCore) error {
		return gstore.ErrResourceNotFound
	})
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)

	// A webhook that no longer exists surfaces the update failure.
	require.NoError(t, dao.DeleteByRegistryAndIdentifier(ctx, reg.ID, "locked"))
	_, err = dao.UpdateOptLock(ctx, hook, func(h *gtypes.WebhookCore) error {
		h.Description = "again"
		return nil
	})
	require.Error(t, err)
}

func TestRdbWebhookExecution(t *testing.T) {
	e := rdbNewEnv(t)
	ctx := e.ctx
	rdbInsertPrincipal(t, e)
	reg := e.newDockerRegistry(t, "docker")
	hooks := NewWebhookDao(e.db)
	dao := NewWebhookExecutionDao(e.db)

	hook := rdbWebhook(reg.ID, "exec")
	require.NoError(t, hooks.Create(ctx, hook))

	retrigger := int64(5)
	executions := make([]*gtypes.WebhookExecutionCore, 0, 3)
	for i, trig := range []string{"trig-a", "trig-b", "trig-a"} {
		exec := &gtypes.WebhookExecutionCore{
			WebhookID:     hook.ID,
			TriggerType:   enum.WebhookTriggerArtifactCreated,
			TriggerID:     trig,
			Result:        enum.WebhookExecutionResultSuccess,
			Created:       int64(100 + i),
			Duration:      12,
			Retriggerable: true,
			Error:         "",
			Request:       gtypes.WebhookExecutionRequest{URL: "https://hooks.example.com", Headers: "{}", Body: "{}"},
			Response:      gtypes.WebhookExecutionResponse{StatusCode: 200, Status: "200 OK", Headers: "{}", Body: "ok"},
		}
		if i == 2 {
			exec.RetriggerOf = &retrigger
			exec.Result = enum.WebhookExecutionResultFatalError
			exec.Error = "boom"
		}
		require.NoError(t, dao.Create(ctx, exec))
		executions = append(executions, exec)
	}

	// Executions of unrelated webhooks are not listed or counted.
	other := &gtypes.WebhookExecutionCore{
		WebhookID: hook.ID + 100, TriggerType: enum.WebhookTriggerArtifactCreated, TriggerID: "x",
		Result: enum.WebhookExecutionResultSuccess,
	}
	require.NoError(t, dao.Create(ctx, other))

	list, err := dao.ListForWebhook(ctx, hook.ID, 10, 1, 10)
	require.NoError(t, err)
	require.Len(t, list, 3)
	// Newest first.
	require.Equal(t, "trig-a", list[0].TriggerID)
	require.Equal(t, enum.WebhookExecutionResultFatalError, list[0].Result)
	require.Equal(t, "boom", list[0].Error)
	require.NotNil(t, list[0].RetriggerOf)
	require.Equal(t, int64(5), *list[0].RetriggerOf)
	require.Equal(t, "ok", list[1].Response.Body)
	require.Equal(t, 200, list[1].Response.StatusCode)
	require.Equal(t, "https://hooks.example.com", list[1].Request.URL)

	paged, err := dao.ListForWebhook(ctx, hook.ID, 2, 2, 2)
	require.NoError(t, err)
	require.Len(t, paged, 1)

	cnt, err := dao.CountForWebhook(ctx, hook.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), cnt)

	byTrigger, err := dao.ListForTrigger(ctx, "trig-a")
	require.NoError(t, err)
	require.Len(t, byTrigger, 2)
	byTrigger, err = dao.ListForTrigger(ctx, "none")
	require.NoError(t, err)
	require.Empty(t, byTrigger)

	first := list[len(list)-1]
	found, err := dao.Find(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, "trig-a", found.TriggerID)
	require.True(t, found.Retriggerable)
	require.Nil(t, found.RetriggerOf)
	_, err = dao.Find(ctx, 99999)
	require.ErrorIs(t, err, gstore.ErrResourceNotFound)
	require.Len(t, executions, 3)
}
