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
	"encoding/json"
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
	gtypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type rdbFailCase struct {
	name string
	fn   func(ctx context.Context) error
}

// rdbDiscard drops the value and keeps only the error of a store call.
func rdbDiscard[T any](_ T, err error) error {
	return err
}

// rdbDiscard2 drops the first two values and keeps only the error of a store call.
func rdbDiscard2[T, U any](_ T, _ U, err error) error {
	return err
}

func rdbRegistryFailCases(e *rdbEnv) []rdbFailCase {
	reg := &types.Registry{ID: 1, Name: "r", ParentID: 1, RootParentID: 1}
	repos := []string{}
	return []rdbFailCase{
		{"registry get", func(ctx context.Context) error { return rdbDiscard(e.registries.Get(ctx, 1)) }},
		{"registry get by uuid", func(ctx context.Context) error {
			return rdbDiscard(e.registries.GetByUUID(ctx, "u"))
		}},
		{"registry get by parent", func(ctx context.Context) error {
			return rdbDiscard(e.registries.GetByParentIDAndName(ctx, 1, "r"))
		}},
		{"registry get by root", func(ctx context.Context) error {
			return rdbDiscard(e.registries.GetByRootParentIDAndName(ctx, 1, "r"))
		}},
		{"registry get by ids", func(ctx context.Context) error {
			return rdbDiscard(e.registries.GetByIDIn(ctx, []int64{1}))
		}},
		{"registry count", func(ctx context.Context) error { return rdbDiscard(e.registries.Count(ctx)) }},
		{"registry count all", func(ctx context.Context) error {
			return rdbDiscard(e.registries.CountAll(ctx, []int64{1}, nil, "", ""))
		}},
		{"registry get all", func(ctx context.Context) error {
			return rdbDiscard(e.registries.GetAll(ctx, []int64{1}, nil, "name", "ASC", 1, 0, "", ""))
		}},
		{"registry create", func(ctx context.Context) error { return rdbDiscard(e.registries.Create(ctx, reg)) }},
		{"registry update", func(ctx context.Context) error { return e.registries.Update(ctx, reg) }},
		{"registry delete", func(ctx context.Context) error { return e.registries.Delete(ctx, 1, "r") }},
		{"registry upstream keys", func(ctx context.Context) error {
			return rdbDiscard(e.registries.FetchUpstreamProxyKeys(ctx, []int64{1}))
		}},
		{"registry upstream ids", func(ctx context.Context) error {
			return rdbDiscard(e.registries.FetchUpstreamProxyIDs(ctx, []string{"a"}, 1))
		}},
		{"registry by upstream", func(ctx context.Context) error {
			return rdbDiscard(e.registries.FetchRegistriesIDByUpstreamProxyID(ctx, "1", 1))
		}},
		{"registry ids by parent", func(ctx context.Context) error {
			return rdbDiscard(e.registries.GetIDsByParentSpace(ctx, 1))
		}},
		{"registry update parent", func(ctx context.Context) error {
			return rdbDiscard(e.registries.UpdateParentSpace(ctx, 1, 2))
		}},
		{"image get", func(ctx context.Context) error { return rdbDiscard(e.images.Get(ctx, 1)) }},
		{"image get by name", func(ctx context.Context) error { return rdbDiscard(e.images.GetByName(ctx, 1, "i")) }},
		{"image get by name and type", func(ctx context.Context) error {
			model := artifact.ArtifactTypeModel
			return rdbDiscard(e.images.GetByNameAndType(ctx, 1, "i", &model))
		}},
		{"image get by uuid", func(ctx context.Context) error { return rdbDiscard(e.images.GetByUUID(ctx, "u")) }},
		{"image get by repo", func(ctx context.Context) error {
			return rdbDiscard(e.images.GetByRepoAndName(ctx, 1, "r", "i"))
		}},
		{"image create", func(ctx context.Context) error {
			return e.images.CreateOrUpdate(ctx, &types.Image{Name: "i", RegistryID: 1})
		}},
		{"image update", func(ctx context.Context) error { return e.images.Update(ctx, &types.Image{ID: 1}) }},
		{"image status", func(ctx context.Context) error {
			return e.images.UpdateStatus(ctx, &types.Image{Name: "i"})
		}},
		{"image labels", func(ctx context.Context) error {
			return rdbDiscard(e.images.GetLabelsByParentIDAndRepo(ctx, 1, "r", 1, 0, ""))
		}},
		{"image label count", func(ctx context.Context) error {
			return rdbDiscard(e.images.CountLabelsByParentIDAndRepo(ctx, 1, "r", ""))
		}},
		{"image delete", func(ctx context.Context) error { return e.images.DeleteByImageNameAndRegID(ctx, 1, "i") }},
		{"image delete unlinked", func(ctx context.Context) error {
			return e.images.DeleteByImageNameIfNoLinkedArtifacts(ctx, 1, "i")
		}},
		{"artifact get", func(ctx context.Context) error { return rdbDiscard(e.artifacts.Get(ctx, 1)) }},
		{"artifact get by uuid", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetByUUID(ctx, "u"))
		}},
		{"artifact get by name", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetByName(ctx, 1, "1"))
		}},
		{"artifact by registry image", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetByRegistryImageAndVersion(ctx, 1, "i", "1"))
		}},
		{"artifact by type", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetByRegistryImageVersionAndArtifactType(ctx, 1, "i", "1", "model"))
		}},
		{"artifact by registry id", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetByRegistryIDAndImage(ctx, 1, "i"))
		}},
		{"artifact latest", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetLatestByImageID(ctx, 1))
		}},
		{"artifact create", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.CreateOrUpdate(ctx, &types.Artifact{ImageID: 1, Version: "1"}))
		}},
		{"artifact count", func(ctx context.Context) error { return rdbDiscard(e.artifacts.Count(ctx)) }},
		{"artifact list parent", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetAllArtifactsByParentID(ctx, 1, &repos, "image_name", "ASC", 1, 0, "", false, nil))
		}},
		{"artifact count parent", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.CountAllArtifactsByParentID(ctx, 1, &repos, "", false, nil))
		}},
		{"artifact list repo", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetArtifactsByRepo(ctx, 1, "r", "name", "ASC", 1, 0, "", nil, nil))
		}},
		{"artifact count repo", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.CountArtifactsByRepo(ctx, 1, "r", "", nil, nil))
		}},
		{"artifact latest metadata", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetLatestArtifactMetadata(ctx, 1, "r", "i"))
		}},
		{"artifact versions", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetAllVersionsByRepoAndImage(ctx, 1, "i", "name", "ASC", 1, 0, "", nil))
		}},
		{"artifact count versions", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.CountAllVersionsByRepoAndImage(ctx, 1, "r", "i", "", nil))
		}},
		{"artifact metadata", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetArtifactMetadata(ctx, 1, "r", "i", "1", nil))
		}},
		{"artifact update metadata", func(ctx context.Context) error {
			return e.artifacts.UpdateArtifactMetadata(ctx, json.RawMessage(`{}`), 1)
		}},
		{"artifact latest by repo", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetLatestArtifactsByRepo(ctx, 1, 1, 0))
		}},
		{"artifact all by repo", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetAllArtifactsByRepo(ctx, 1, 1, 0))
		}},
		{"artifact batch", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.GetArtifactsByRepoAndImageBatch(ctx, 1, "i", 1, 0))
		}},
		{"artifact search latest", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.SearchLatestByName(ctx, 1, "n", 1, 0))
		}},
		{"artifact count latest", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.CountLatestByName(ctx, 1, "n"))
		}},
		{"artifact search", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.SearchByImageName(ctx, 1, "n", 1, 0))
		}},
		{"artifact count by image", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.CountByImageName(ctx, 1, "n"))
		}},
		{"artifact delete by image", func(ctx context.Context) error {
			return e.artifacts.DeleteByImageNameAndRegistryID(ctx, 1, "i")
		}},
		{"artifact delete by version", func(ctx context.Context) error {
			return e.artifacts.DeleteByVersionAndImageName(ctx, "i", "1", 1)
		}},
		{"artifact duplicate", func(ctx context.Context) error {
			return rdbDiscard(e.artifacts.DuplicateArtifact(ctx, &types.Artifact{Version: "1"}, 2))
		}},
	}
}

func rdbOCIFailCases(e *rdbEnv) []rdbFailCase {
	repos := []string{}
	tx := dbtx.New(e.db)
	dgst := rdbDigest(1)
	tdgst, _ := types.NewDigest(dgst)
	manifest := &types.Manifest{
		RegistryID: 1, ImageName: "i", Digest: dgst, MediaType: "application/vnd.oci.image.manifest.v1+json",
	}
	blobs := NewRegistryBlobDao(e.db)
	layers := NewLayersDao(e.db, e.mediaTypes)
	refs := NewManifestReferenceDao(e.db)
	oci := NewOCIImageIndexMappingDao(e.db)
	return []rdbFailCase{
		{"media type map", func(ctx context.Context) error { return rdbDiscard(e.mediaTypes.MapMediaType(ctx, "x")) }},
		{"media type exists", func(ctx context.Context) error {
			return rdbDiscard(e.mediaTypes.MediaTypeExists(ctx, "x"))
		}},
		{"media type get", func(ctx context.Context) error {
			return rdbDiscard(e.mediaTypes.GetMediaTypeByID(ctx, 1))
		}},
		{"blob find", func(ctx context.Context) error { return rdbDiscard(e.blobs.FindByID(ctx, 1)) }},
		{"blob find by digest", func(ctx context.Context) error {
			return rdbDiscard(e.blobs.FindByDigestAndRootParentID(ctx, dgst, 1))
		}},
		{"blob find by repo", func(ctx context.Context) error {
			return rdbDiscard(e.blobs.FindByDigestAndRepoID(ctx, dgst, 1, "i"))
		}},
		{"blob create", func(ctx context.Context) error {
			return rdbDiscard2(e.blobs.CreateOrFind(ctx, &types.Blob{MediaType: "x", Digest: dgst}))
		}},
		{"blob delete", func(ctx context.Context) error { return e.blobs.DeleteByID(ctx, 1) }},
		{"blob exists", func(ctx context.Context) error { return rdbDiscard(e.blobs.ExistsBlob(ctx, 1, dgst, "i")) }},
		{"blob size", func(ctx context.Context) error { return rdbDiscard(e.blobs.TotalSizeByRootParentID(ctx, 1)) }},
		{"registry blob link", func(ctx context.Context) error {
			return blobs.LinkBlob(ctx, "i", &types.Registry{ID: 1}, 1)
		}},
		{"registry blob unlink", func(ctx context.Context) error {
			return rdbDiscard(blobs.UnlinkBlob(ctx, "i", &types.Registry{ID: 1}, 1))
		}},
		{"registry blob unlink by image", func(ctx context.Context) error {
			return rdbDiscard(blobs.UnlinkBlobByImageName(ctx, 1, "i"))
		}},
		{"layer associate", func(ctx context.Context) error {
			return layers.AssociateLayerBlob(ctx, &types.Manifest{ID: 1}, &types.Blob{ID: 1, MediaType: "x"})
		}},
		{"layer list", func(ctx context.Context) error { return rdbDiscard(layers.GetAllLayersByManifestID(ctx, 1)) }},
		{"manifest create", func(ctx context.Context) error { return e.manifests.Create(ctx, manifest) }},
		{"manifest create or find", func(ctx context.Context) error { return e.manifests.CreateOrFind(ctx, manifest) }},
		{"manifest get", func(ctx context.Context) error { return rdbDiscard(e.manifests.Get(ctx, 1)) }},
		{"manifest delete", func(ctx context.Context) error { return e.manifests.Delete(ctx, 1, 1) }},
		{"manifest delete by digest", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.DeleteManifest(ctx, 1, "i", dgst))
		}},
		{"manifest delete by image", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.DeleteManifestByImageName(ctx, 1, "i"))
		}},
		{"manifest find by digest", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.FindManifestByDigest(ctx, 1, "i", tdgst))
		}},
		{"manifest find by tag", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.FindManifestByTagName(ctx, 1, "i", "t"))
		}},
		{"manifest digest by tag", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.FindManifestDigestByTagName(ctx, 1, "i", "t"))
		}},
		{"manifest payload", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.GetManifestPayload(ctx, 1, "r", "i", tdgst))
		}},
		{"manifest payload by tag", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.FindManifestPayloadByTagName(ctx, 1, "r", "i", "t"))
		}},
		{"manifest by subject", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.ListManifestsBySubject(ctx, 1, 1))
		}},
		{"manifest by subject digest", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.ListManifestsBySubjectDigest(ctx, 1, tdgst))
		}},
		{"manifest latest", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.GetLatestManifest(ctx, 1, "i"))
		}},
		{"manifest count", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.CountByImageName(ctx, 1, "i"))
		}},
		{"manifest references", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.References(ctx, &types.Manifest{ID: 1, RegistryID: 1}))
		}},
		{"manifest referenced by", func(ctx context.Context) error {
			return rdbDiscard(e.manifests.ReferencedBy(ctx, &types.Manifest{ID: 1, RegistryID: 1}))
		}},
		{"manifest reference associate", func(ctx context.Context) error {
			return refs.AssociateManifest(ctx, &types.Manifest{ID: 1}, &types.Manifest{ID: 2})
		}},
		{"oci mapping create", func(ctx context.Context) error {
			return oci.Create(ctx, &types.OCIImageIndexMapping{ParentManifestID: 1, ChildManifestDigest: dgst})
		}},
		{"oci mapping list", func(ctx context.Context) error {
			return rdbDiscard(oci.GetAllByChildDigest(ctx, 1, "i", tdgst))
		}},
		{"tag create", func(ctx context.Context) error {
			return e.tags.CreateOrUpdate(ctx, &types.Tag{Name: "t", ImageName: "i", RegistryID: 1, ManifestID: 1})
		}},
		{"tag find", func(ctx context.Context) error { return rdbDiscard(e.tags.FindTag(ctx, 1, "i", "t")) }},
		{"tag detail", func(ctx context.Context) error { return rdbDiscard(e.tags.GetTagDetail(ctx, 1, "i", "t")) }},
		{"tag delete", func(ctx context.Context) error { return e.tags.DeleteTag(ctx, 1, "i", "t") }},
		{"tag delete by name", func(ctx context.Context) error {
			return rdbDiscard(e.tags.DeleteTagByName(ctx, 1, "t"))
		}},
		{"tag delete by manifest", func(ctx context.Context) error {
			return rdbDiscard(e.tags.DeleteTagByManifestID(ctx, 1, 1))
		}},
		{"tag delete by image", func(ctx context.Context) error { return e.tags.DeleteTagsByImageName(ctx, 1, "i") }},
		{"tag by manifest", func(ctx context.Context) error { return rdbDiscard(e.tags.GetTagsByManifestID(ctx, 1)) }},
		{"tag paginated", func(ctx context.Context) error {
			return rdbDiscard(e.tags.TagsPaginated(ctx, 1, "i", types.FilterParams{MaxEntries: 1}))
		}},
		{"tag has after", func(ctx context.Context) error {
			return rdbDiscard(e.tags.HasTagsAfterName(ctx, 1, types.FilterParams{MaxEntries: 1}))
		}},
		{"tag latest", func(ctx context.Context) error { return rdbDiscard(e.tags.GetLatestTag(ctx, 1, "i")) }},
		{"tag latest name", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetLatestTagName(ctx, 1, "r", "i"))
		}},
		{"tag latest metadata", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetLatestTagMetadata(ctx, 1, "r", "i"))
		}},
		{"tag tag metadata", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetTagMetadata(ctx, 1, "r", "i", "t"))
		}},
		{"tag oci version", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetOCIVersionMetadata(ctx, 1, "r", "i", dgst.String()))
		}},
		{"tag list parent", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetAllArtifactsByParentID(ctx, 1, &repos, "image_name", "ASC", 1, 0, "", false, nil))
		}},
		{"tag list untagged", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetAllArtifactsByParentIDUntagged(ctx, 1, &repos, "image_name", "ASC", 1, 0, "", nil))
		}},
		{"tag count parent", func(ctx context.Context) error {
			return rdbDiscard(e.tags.CountAllArtifactsByParentID(ctx, 1, &repos, "", false, nil, false))
		}},
		{"tag count untagged", func(ctx context.Context) error {
			return rdbDiscard(e.tags.CountAllArtifactsByParentID(ctx, 1, &repos, "", false, nil, true))
		}},
		{"tag list repo", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetAllArtifactsByRepo(ctx, 1, "r", "updated_at", "ASC", 1, 0, "", nil))
		}},
		{"tag count repo", func(ctx context.Context) error {
			return rdbDiscard(e.tags.CountAllArtifactsByRepo(ctx, 1, "r", "", nil))
		}},
		{"tag list tags", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetAllTagsByRepoAndImage(ctx, 1, "r", "i", "name", "ASC", 1, 0, ""))
		}},
		{"tag list versions", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetAllOciVersionsByRepoAndImage(ctx, 1, "r", "i", "size", "ASC", 1, 0, ""))
		}},
		{"tag tags info", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetOciTagsInfo(ctx, 1, "i", 1, 0, ""))
		}},
		{"tag count tags", func(ctx context.Context) error {
			return rdbDiscard(e.tags.CountAllTagsByRepoAndImage(ctx, 1, "r", "i", ""))
		}},
		{"tag count versions", func(ctx context.Context) error {
			return rdbDiscard(e.tags.CountOciVersionByRepoAndImage(ctx, 1, "r", "i", ""))
		}},
		{"tag quarantine status", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetQuarantineStatusForImages(ctx, []string{"i"}, 1))
		}},
		{"tag quarantine info", func(ctx context.Context) error {
			return rdbDiscard(e.tags.GetQuarantineInfoForArtifacts(ctx, []types.ArtifactIdentifier{{Name: "i"}}, 1))
		}},
		{"cleanup policy ids", func(ctx context.Context) error {
			return rdbDiscard(NewCleanupPolicyDao(e.db, tx).GetIDsByRegistryID(ctx, 1))
		}},
		{"cleanup policy create", func(ctx context.Context) error {
			return rdbDiscard(NewCleanupPolicyDao(e.db, tx).Create(ctx, &types.CleanupPolicy{RegistryID: 1}))
		}},
		{"cleanup policy delete", func(ctx context.Context) error {
			return NewCleanupPolicyDao(e.db, tx).Delete(ctx, 1)
		}},
		{"cleanup policy modify", func(ctx context.Context) error {
			policies := []types.CleanupPolicy{{RegistryID: 1}}
			return NewCleanupPolicyDao(e.db, tx).ModifyCleanupPolicies(ctx, &policies, []int64{1})
		}},
	}
}

func rdbMiscFailCases(e *rdbEnv) []rdbFailCase {
	tx := dbtx.New(e.db)
	nodes := NewNodeDao(e.db)
	generic := NewGenericBlobDao(e.db)
	quarantine := NewQuarantineArtifactDao(e.db)
	packageTags := NewPackageTagDao(e.db)
	downloads := NewDownloadStatDao(e.db)
	bandwidth := NewBandwidthStatDao(e.db)
	upstream := NewUpstreamproxyDao(e.db, e.registries, rdbNewSpaceFinder())
	webhooks := NewWebhookDao(e.db)
	executions := NewWebhookExecutionDao(e.db)
	tasks := NewTaskStore(e.db, tx)
	sources := NewTaskSourceStore(e.db, tx)
	events := NewTaskEventStore(e.db)
	hook := &gtypes.WebhookCore{
		ParentType: enum.WebhookParentRegistry, ParentID: 1, Identifier: "h", CreatedBy: 7,
	}
	return []rdbFailCase{
		{"node get", func(ctx context.Context) error { return rdbDiscard(nodes.Get(ctx, "n")) }},
		{"node by path", func(ctx context.Context) error {
			return rdbDiscard(nodes.GetByPathAndRegistryID(ctx, 1, "/p"))
		}},
		{"node by name", func(ctx context.Context) error {
			return rdbDiscard(nodes.GetByNameAndRegistryID(ctx, 1, "n"))
		}},
		{"node by blob", func(ctx context.Context) error {
			return rdbDiscard(nodes.GetByBlobIDAndRegistryID(ctx, "b", 1))
		}},
		{"node find", func(ctx context.Context) error {
			return rdbDiscard(nodes.FindByPathAndRegistryID(ctx, 1, "/p", "f"))
		}},
		{"node find paths", func(ctx context.Context) error {
			return rdbDiscard(nodes.FindByPathsAndRegistryID(ctx, []string{"/p"}, 1))
		}},
		{"node count", func(ctx context.Context) error {
			return rdbDiscard(nodes.CountByPathAndRegistryID(ctx, 1, "/p"))
		}},
		{"node create", func(ctx context.Context) error {
			return nodes.Create(ctx, &types.Node{Name: "n", RegistryID: 1})
		}},
		{"node delete", func(ctx context.Context) error { return nodes.DeleteByNodePathAndRegistryID(ctx, "/p", 1) }},
		{"node delete leaf", func(ctx context.Context) error {
			return nodes.DeleteByLeafNodePathAndRegistryID(ctx, "/p", 1)
		}},
		{"node prefix", func(ctx context.Context) error {
			return rdbDiscard(nodes.GetAllFileNodesByPathPrefixAndRegistryID(ctx, 1, "/p"))
		}},
		{"node files metadata", func(ctx context.Context) error {
			return rdbDiscard(nodes.GetFilesMetadataByPathAndRegistryID(ctx, 1, "/p", "n.node_name", "ASC", 1, 0, ""))
		}},
		{"node file metadata", func(ctx context.Context) error {
			return rdbDiscard(nodes.GetFileMetadataByPathAndRegistryID(ctx, 1, "/p"))
		}},
		{"generic blob find", func(ctx context.Context) error { return rdbDiscard(generic.FindByID(ctx, "b")) }},
		{"generic blob find sha", func(ctx context.Context) error {
			return rdbDiscard(generic.FindBySha256AndRootParentID(ctx, "s", 1))
		}},
		{"generic blob size", func(ctx context.Context) error {
			return rdbDiscard(generic.TotalSizeByRootParentID(ctx, 1))
		}},
		{"generic blob create", func(ctx context.Context) error {
			return rdbDiscard2(generic.Create(ctx, &types.GenericBlob{Sha256: "s", Sha512: "x"}))
		}},
		{"quarantine get", func(ctx context.Context) error {
			return rdbDiscard(quarantine.GetByFilePath(ctx, "/p", 1, "i", "1", nil))
		}},
		{"quarantine create", func(ctx context.Context) error {
			return quarantine.Create(ctx, &types.QuarantineArtifact{RegistryID: 1, Reason: "r"})
		}},
		{"quarantine delete", func(ctx context.Context) error {
			return quarantine.DeleteByRegistryIDArtifactAndFilePath(ctx, 1, nil, 1, nil)
		}},
		{"package tag find", func(ctx context.Context) error {
			return rdbDiscard(packageTags.FindByImageNameAndRegID(ctx, "i", 1, nil))
		}},
		{"package tag delete", func(ctx context.Context) error {
			return packageTags.DeleteByTagAndImageName(ctx, "t", "i", 1)
		}},
		{"package tag delete by image", func(ctx context.Context) error {
			return packageTags.DeleteByImageNameAndRegID(ctx, "i", 1)
		}},
		{"package tag create", func(ctx context.Context) error {
			return rdbDiscard(packageTags.Create(ctx, &types.PackageTag{ID: "p", Name: "t", ArtifactID: 1}))
		}},
		{"download create", func(ctx context.Context) error {
			return downloads.Create(ctx, &types.DownloadStat{ArtifactID: 1})
		}},
		{"download create by name", func(ctx context.Context) error {
			return downloads.CreateByRegistryIDImageAndArtifactName(ctx, 1, "i", "1", nil)
		}},
		{"download image", func(ctx context.Context) error {
			return rdbDiscard(downloads.GetTotalDownloadsForImage(ctx, 1))
		}},
		{"download artifact", func(ctx context.Context) error {
			return rdbDiscard(downloads.GetTotalDownloadsForArtifactID(ctx, 1))
		}},
		{"download manifests", func(ctx context.Context) error {
			return rdbDiscard(downloads.GetTotalDownloadsForManifests(ctx, []string{"1"}, 1))
		}},
		{"download registry", func(ctx context.Context) error {
			return rdbDiscard(downloads.GetTotalDownloadsForRegistryID(ctx, 1))
		}},
		{"download registries", func(ctx context.Context) error {
			return rdbDiscard(downloads.GetTotalDownloadsForRegistryIDs(ctx, []int64{1}))
		}},
		{"download images", func(ctx context.Context) error {
			return rdbDiscard(downloads.GetTotalDownloadsForImageIDs(ctx, []int64{1}))
		}},
		{"download artifacts", func(ctx context.Context) error {
			return rdbDiscard(downloads.GetTotalDownloadsForArtifactIDs(ctx, []int64{1}))
		}},
		{"bandwidth create", func(ctx context.Context) error {
			return bandwidth.Create(ctx, &types.BandwidthStat{ImageID: 1, Type: types.BandwidthType("DOWNLOAD")})
		}},
		{"upstream get", func(ctx context.Context) error { return rdbDiscard(upstream.Get(ctx, 1)) }},
		{"upstream by identifier", func(ctx context.Context) error {
			return rdbDiscard(upstream.GetByRegistryIdentifier(ctx, 1, "r"))
		}},
		{"upstream by parent", func(ctx context.Context) error { return rdbDiscard(upstream.GetByParentID(ctx, "1")) }},
		{"upstream create", func(ctx context.Context) error {
			return rdbDiscard(upstream.Create(ctx, &types.UpstreamProxyConfig{RegistryID: 1, AuthType: "Anonymous"}))
		}},
		{"upstream update", func(ctx context.Context) error {
			return upstream.Update(ctx, &types.UpstreamProxyConfig{ID: 1, RegistryID: 1, AuthType: "Anonymous"})
		}},
		{"upstream list", func(ctx context.Context) error {
			return rdbDiscard(upstream.GetAll(ctx, 1, nil, "name", "ASC", 1, 0, ""))
		}},
		{"upstream count", func(ctx context.Context) error { return rdbDiscard(upstream.CountAll(ctx, "1", nil, "")) }},
		{"upstream secret space", func(ctx context.Context) error {
			return rdbDiscard(upstream.UpdateSecretSpaceID(ctx, 1, 2))
		}},
		{"upstream user secret space", func(ctx context.Context) error {
			return rdbDiscard(upstream.UpdateUserNameSecretSpaceID(ctx, 1, 2))
		}},
		{"webhook create", func(ctx context.Context) error { return webhooks.Create(ctx, hook) }},
		{"webhook find", func(ctx context.Context) error { return rdbDiscard(webhooks.Find(ctx, 1)) }},
		{"webhook by identifier", func(ctx context.Context) error {
			return rdbDiscard(webhooks.GetByRegistryAndIdentifier(ctx, 1, "h"))
		}},
		{"webhook list", func(ctx context.Context) error {
			return rdbDiscard(webhooks.ListByRegistry(ctx, "name", "ASC", 1, 0, "", 1))
		}},
		{"webhook list all", func(ctx context.Context) error {
			return rdbDiscard(webhooks.ListAllByRegistry(ctx, []gtypes.WebhookParentInfo{
				{Type: enum.WebhookParentRegistry, ID: 1},
			}))
		}},
		{"webhook count", func(ctx context.Context) error {
			return rdbDiscard(webhooks.CountAllByRegistry(ctx, 1, ""))
		}},
		{"webhook update", func(ctx context.Context) error { return webhooks.Update(ctx, hook) }},
		{"webhook delete", func(ctx context.Context) error {
			return webhooks.DeleteByRegistryAndIdentifier(ctx, 1, "h")
		}},
		{"webhook parent space", func(ctx context.Context) error {
			return rdbDiscard(webhooks.UpdateParentSpace(ctx, 1, 2))
		}},
		{"webhook secret space", func(ctx context.Context) error {
			return rdbDiscard(webhooks.UpdateSecretSpaceID(ctx, 1, 2))
		}},
		{"execution find", func(ctx context.Context) error { return rdbDiscard(executions.Find(ctx, 1)) }},
		{"execution create", func(ctx context.Context) error {
			return executions.Create(ctx, &gtypes.WebhookExecutionCore{WebhookID: 1})
		}},
		{"execution list", func(ctx context.Context) error {
			return rdbDiscard(executions.ListForWebhook(ctx, 1, 10, 1, 10))
		}},
		{"execution count", func(ctx context.Context) error { return rdbDiscard(executions.CountForWebhook(ctx, 1)) }},
		{"execution by trigger", func(ctx context.Context) error {
			return rdbDiscard(executions.ListForTrigger(ctx, "t"))
		}},
		{"task find", func(ctx context.Context) error { return rdbDiscard(tasks.Find(ctx, "k")) }},
		{"task upsert", func(ctx context.Context) error {
			return tasks.UpsertTask(ctx, &types.Task{Key: "k", Kind: types.TaskKindBuildRegistryIndex})
		}},
		{"task lock", func(ctx context.Context) error {
			return rdbDiscard(tasks.LockForUpdate(ctx, &types.Task{Key: "k"}))
		}},
		{"task run again", func(ctx context.Context) error { return tasks.SetRunAgain(ctx, "k", true) }},
		{"task status", func(ctx context.Context) error {
			return tasks.UpdateStatus(ctx, "k", types.TaskStatusSuccess)
		}},
		{"task complete", func(ctx context.Context) error {
			return rdbDiscard(tasks.CompleteTask(ctx, "k", types.TaskStatusSuccess))
		}},
		{"task pending", func(ctx context.Context) error { return rdbDiscard(tasks.ListPendingTasks(ctx, 1)) }},
		{"task source find", func(ctx context.Context) error {
			return rdbDiscard(sources.FindByTaskKeyAndSourceType(ctx, "k", "Registry"))
		}},
		{"task source insert", func(ctx context.Context) error {
			return sources.InsertSource(ctx, "k", types.SourceRef{Type: types.SourceTypeRegistry, ID: 1})
		}},
		{"task source claim", func(ctx context.Context) error { return sources.ClaimSources(ctx, "k", "run") }},
		{"task source status", func(ctx context.Context) error {
			return sources.UpdateSourceStatus(ctx, "run", types.TaskStatusSuccess, "")
		}},
		{"task event", func(ctx context.Context) error { return events.LogTaskEvent(ctx, "k", "e", nil) }},
	}
}

// TestRdbStores_ClosedDatabase checks that every store reports database failures instead of swallowing them.
func TestRdbStores_ClosedDatabase(t *testing.T) {
	e := rdbNewEnv(t)
	require.NoError(t, e.db.Close())

	cases := rdbRegistryFailCases(e)
	cases = append(cases, rdbOCIFailCases(e)...)
	cases = append(cases, rdbMiscFailCases(e)...)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, tc.fn(e.ctx))
		})
	}
}
