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

package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	gitnesstypes "github.com/harness/gitness/types"

	"github.com/lib/pq"
	"github.com/opencontainers/go-digest"
)

type MediaTypesRepository interface {
	MapMediaType(ctx context.Context, mediaType string) (int64, error)
	MediaTypeExists(ctx context.Context, mediaType string) (bool, error)
	GetMediaTypeByID(ctx context.Context, ids int64) (*types.MediaType, error)
}

type BlobRepository interface {
	FindByID(ctx context.Context, id int64) (*types.Blob, error)
	FindByDigestAndRootParentID(
		ctx context.Context, d digest.Digest,
		rootParentID int64,
	) (*types.Blob, error)
	FindByDigestAndRepoID(
		ctx context.Context, d digest.Digest, repoID int64,
		imageName string,
	) (*types.Blob, error)
	CreateOrFind(ctx context.Context, b *types.Blob) (*types.Blob, bool, error)
	DeleteByID(ctx context.Context, id int64) error
	ExistsBlob(
		ctx context.Context, repoID int64, d digest.Digest,
		image string,
	) (bool, error)
	TotalSizeByRootParentID(ctx context.Context, id int64) (int64, error)
}

type CleanupPolicyRepository interface {
	// GetIDsByRegistryID the CleanupPolicy Ids specified by Registry Key
	GetIDsByRegistryID(ctx context.Context, id int64) (ids []int64, err error)
	// GetByRegistryID the CleanupPolicy specified by Registry Key
	GetByRegistryID(
		ctx context.Context,
		id int64,
	) (cleanupPolicies *[]types.CleanupPolicy, err error)
	// Create a CleanupPolicy
	Create(
		ctx context.Context,
		cleanupPolicy *types.CleanupPolicy,
	) (id int64, err error)
	// Delete the CleanupPolicy specified by repokey and name
	Delete(ctx context.Context, id int64) (err error)
	// Update the CleanupPolicy.
	ModifyCleanupPolicies(
		ctx context.Context,
		cleanupPolicies *[]types.CleanupPolicy, ids []int64,
	) error
}

type ManifestRepository interface {
	// FindAll finds all manifests.
	FindAll(ctx context.Context) (types.Manifests, error)
	// Count counts all manifests.
	Count(ctx context.Context) (int, error)
	// LayerBlobs finds layer blobs associated with a manifest,
	// through the `layers` relationship entity.
	LayerBlobs(ctx context.Context, m *types.Manifest) (types.Blobs, error)
	// References finds all manifests directly
	// referenced by a manifest (if any).
	References(ctx context.Context, m *types.Manifest) (types.Manifests, error)
	// Create saves a new Manifest. ID value is updated in given request object
	Create(ctx context.Context, m *types.Manifest) error
	// CreateOrFind attempts to create a manifest. If the manifest already exists
	// (same digest in the scope of a given repository)
	// that record is loaded from the database into m.
	// This is similar to a repositoryStore.FindManifestByDigest followed by
	// a Create, but without being  prone to race conditions on write
	// operations between the corresponding read (FindManifestByDigest2)
	// and write (Create) operations.
	// Separate Find* and Create method calls should be preferred
	// to this when race conditions are not a concern.
	CreateOrFind(ctx context.Context, m *types.Manifest) error
	AssociateLayerBlob(ctx context.Context, m *types.Manifest, b *types.Blob) error
	DissociateLayerBlob(ctx context.Context, m *types.Manifest, b *types.Blob) error
	Delete(ctx context.Context, registryID, id int64) error
	FindManifestByDigest(
		ctx context.Context, repoID int64, imageName string,
		digest types.Digest,
	) (*types.Manifest, error)
	FindManifestByTagName(
		ctx context.Context, repoID int64, imageName string,
		tag string,
	) (*types.Manifest, error)
	FindManifestPayloadByTagName(
		ctx context.Context,
		parentID int64,
		repoKey string,
		imageName string,
		version string,
	) (*types.Payload, error)
	GetManifestPayload(
		ctx context.Context,
		parentID int64,
		repoKey string,
		imageName string,
		digest types.Digest,
	) (*types.Payload, error)

	FindManifestDigestByTagName(
		ctx context.Context, regID int64,
		imageName string, tag string,
	) (types.Digest, error)
	Get(ctx context.Context, manifestID int64) (*types.Manifest, error)
	DeleteManifest(
		ctx context.Context, repoID int64,
		imageName string, d digest.Digest,
	) (bool, error)
	DeleteManifestByImageName(
		ctx context.Context, repoID int64,
		imageName string,
	) (bool, error)
	ListManifestsBySubject(
		ctx context.Context, repoID int64,
		id int64,
	) (types.Manifests, error)
	ListManifestsBySubjectDigest(
		ctx context.Context, repoID int64,
		digest types.Digest,
	) (types.Manifests, error)
	GetLatestManifest(ctx context.Context, repoID int64, imageName string) (*types.Manifest, error)
	CountByImageName(ctx context.Context, repoID int64, imageName string) (int64, error)
}

type ManifestReferenceRepository interface {
	AssociateManifest(
		ctx context.Context, ml *types.Manifest,
		m *types.Manifest,
	) error
	DissociateManifest(
		ctx context.Context, ml *types.Manifest,
		m *types.Manifest,
	) error
}

type OCIImageIndexMappingRepository interface {
	Create(ctx context.Context, ociManifest *types.OCIImageIndexMapping) error
	GetAllByChildDigest(ctx context.Context, registryID int64, imageName string, childDigest types.Digest) (
		[]*types.OCIImageIndexMapping, error,
	)
}

type LayerRepository interface {
	AssociateLayerBlob(ctx context.Context, m *types.Manifest, b *types.Blob) error
	GetAllLayersByManifestID(ctx context.Context, id int64) (*[]types.Layer, error)
}

type TagRepository interface {
	// CreateOrUpdate upsert a tag. A tag with a given name
	// on a given repository may not exist (in which case it should be
	// inserted), already exist and point to the same manifest
	// (in which case nothing needs to be done) or already exist but
	// points to a different manifest (in which case it should be updated).
	CreateOrUpdate(ctx context.Context, t *types.Tag) error
	LockTagByNameForUpdate(
		ctx context.Context, repoID int64,
		name string,
	) (bool, error)
	DeleteTagByName(
		ctx context.Context, repoID int64,
		name string,
	) (bool, error)
	DeleteTagByManifestID(
		ctx context.Context, repoID int64,
		manifestID int64,
	) (bool, error)
	TagsPaginated(
		ctx context.Context, repoID int64, image string,
		filters types.FilterParams,
	) ([]*types.Tag, error)
	HasTagsAfterName(
		ctx context.Context, repoID int64,
		filters types.FilterParams,
	) (bool, error)

	GetAllArtifactsByParentID(
		ctx context.Context, parentID int64,
		registryIDs *[]string, sortByField string,
		sortByOrder string, limit int, offset int, search string,
		latestVersion bool, packageTypes []string,
	) (*[]types.ArtifactMetadata, error)

	GetAllArtifactsByParentIDUntagged(
		ctx context.Context, parentID int64,
		registryIDs *[]string, sortByField string,
		sortByOrder string, limit int, offset int, search string,
		packageTypes []string,
	) (*[]types.ArtifactMetadata, error)

	CountAllArtifactsByParentID(
		ctx context.Context, parentID int64,
		registryIDs *[]string, search string,
		latestVersion bool, packageTypes []string, untaggedImagesEnabled bool,
	) (int64, error)

	GetAllArtifactsByRepo(
		ctx context.Context, parentID int64, repoKey string,
		sortByField string, sortByOrder string,
		limit int, offset int, search string, labels []string,
	) (*[]types.ArtifactMetadata, error)

	GetLatestTagMetadata(
		ctx context.Context,
		parentID int64,
		repoKey string,
		imageName string,
	) (*types.ArtifactMetadata, error)

	GetLatestTagName(
		ctx context.Context, parentID int64, repoKey string,
		imageName string,
	) (string, error)

	GetTagMetadata(
		ctx context.Context,
		parentID int64,
		repoKey string,
		imageName string,
		name string,
	) (*types.OciVersionMetadata, error)

	GetOCIVersionMetadata(
		ctx context.Context,
		parentID int64,
		repoKey string,
		imageName string,
		dgst string,
	) (*types.OciVersionMetadata, error)

	CountAllArtifactsByRepo(
		ctx context.Context, parentID int64, repoKey string,
		search string, labels []string,
	) (int64, error)

	GetTagDetail(
		ctx context.Context, repoID int64, imageName string,
		name string,
	) (*types.TagDetail, error)

	GetLatestTag(ctx context.Context, repoID int64, imageName string) (*types.Tag, error)

	GetAllTagsByRepoAndImage(
		ctx context.Context,
		parentID int64,
		repoKey string,
		image string,
		sortByField string,
		sortByOrder string,
		limit int,
		offset int,
		search string,
	) (*[]types.OciVersionMetadata, error)

	GetAllOciVersionsByRepoAndImage(
		ctx context.Context,
		parentID int64,
		repoKey string,
		image string,
		sortByField string,
		sortByOrder string,
		limit int,
		offset int,
		search string,
	) (*[]types.OciVersionMetadata, error)

	GetOciTagsInfo(
		ctx context.Context, registryID int64,
		image string, limit int, offset int,
		search string,
	) (*[]types.TagInfo, error)

	DeleteTag(ctx context.Context, registryID int64, imageName string, name string) (err error)

	CountAllTagsByRepoAndImage(
		ctx context.Context, parentID int64, repoKey string,
		image string, search string,
	) (int64, error)
	CountOciVersionByRepoAndImage(
		ctx context.Context, parentID int64, repoKey string,
		image string, search string,
	) (int64, error)
	FindTag(
		ctx context.Context, repoID int64, imageName string,
		name string,
	) (*types.Tag, error)
	GetTagsByManifestID(
		ctx context.Context, manifestID int64,
	) (*[]string, error)
	DeleteTagsByImageName(
		ctx context.Context, registryID int64,
		imageName string,
	) (err error)
	GetQuarantineStatusForImages(
		ctx context.Context, imageNames []string, registryID int64,
	) ([]bool, error)
	GetQuarantineInfoForArtifacts(
		ctx context.Context, artifacts []types.ArtifactIdentifier, parentID int64,
	) (map[types.ArtifactIdentifier]*types.QuarantineInfo, error)
}

// UpstreamProxyConfig holds the record of a config of upstream proxy in DB.
type UpstreamProxyConfig struct {
	ID         int64
	RegistryID int64
	Source     string
	URL        string
	AuthType   string
	UserName   string
	Password   string
	Token      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type UpstreamProxyConfigRepository interface {
	// Get the upstreamproxy specified by ID
	Get(ctx context.Context, id int64) (upstreamProxy *types.UpstreamProxy, err error)

	// GetByRepoKey gets the upstreamproxy specified by registry key
	GetByRegistryIdentifier(
		ctx context.Context,
		parentID int64,
		repoKey string,
	) (upstreamProxy *types.UpstreamProxy, err error)

	// GetByParentUniqueId gets the upstreamproxy specified by parent id and parent unique id
	GetByParentID(ctx context.Context, parentID string) (
		upstreamProxies *[]types.UpstreamProxy,
		err error,
	)

	// Create a upstreamProxyConfig
	Create(ctx context.Context, upstreamproxyRecord *types.UpstreamProxyConfig) (
		id int64,
		err error,
	)

	// Update updates the upstreamproxy.
	Update(ctx context.Context, upstreamproxyRecord *types.UpstreamProxyConfig) (err error)

	// UpdateSecretSpaceID updates the secret space ID for all upstream proxy configs
	// referencing secrets from source space.
	UpdateSecretSpaceID(ctx context.Context, srcSpaceID int64, targetSpaceID int64) (int64, error)

	// UpdateUserNameSecretSpaceID updates the username secret space ID for all upstream proxy configs
	// that reference username secrets from the source space to the target space.
	UpdateUserNameSecretSpaceID(ctx context.Context, srcSpaceID int64, targetSpaceID int64) (int64, error)

	GetAll(
		ctx context.Context,
		parentID int64,
		packageTypes []string,
		sortByField string,
		sortByOrder string,
		limit int,
		offset int,
		search string,
	) (upstreamProxies *[]types.UpstreamProxy, err error)

	CountAll(
		ctx context.Context, parentID string, packageTypes []string,
		search string,
	) (count int64, err error)
}

type RegistryMetadata struct {
	RegUUID       string
	RegID         string
	ParentID      int64
	RegIdentifier string
	Description   string
	PackageType   artifact.PackageType
	Type          artifact.RegistryType
	LastModified  time.Time
	URL           string
	Labels        pq.StringArray
	Config        *types.RegistryConfig
	ArtifactCount int64
	DownloadCount int64
	Size          int64
}

type RegistryRepository interface {
	GetByUUID(ctx context.Context, uuid string) (*types.Registry, error)
	// Get the repository specified by ID
	Get(ctx context.Context, id int64) (repository *types.Registry, err error)
	// GetByName gets the repository specified by name
	GetByIDIn(
		ctx context.Context, ids []int64,
	) (registries *[]types.Registry, err error)
	// GetByName gets the repository specified by parent id and name
	GetByParentIDAndName(
		ctx context.Context, parentID int64,
		name string,
	) (registry *types.Registry, err error)
	GetByRootParentIDAndName(
		ctx context.Context, parentID int64,
		name string,
	) (registry *types.Registry, err error)
	// Create a repository
	Create(ctx context.Context, repository *types.Registry) (id int64, err error)
	// Delete the repository specified by ID
	Delete(ctx context.Context, parentID int64, name string) (err error)
	// Update updates the repository. Only the properties specified by "props" will be updated if it is set
	Update(ctx context.Context, repository *types.Registry) (err error)

	GetAll(
		ctx context.Context,
		parentIDs []int64,
		packageTypes []string,
		sortByField string,
		sortByOrder string,
		limit int,
		offset int,
		search string,
		repoType string,
	) (repos *[]RegistryMetadata, err error)

	CountAll(
		ctx context.Context, parentIDs []int64, packageTypes []string,
		search string, repoType string,
	) (count int64, err error)

	FetchUpstreamProxyIDs(
		ctx context.Context, repokeys []string,
		parentID int64,
	) (ids []int64, err error)

	FetchRegistriesIDByUpstreamProxyID(
		ctx context.Context, upstreamProxyID string,
		rootParentID int64,
	) (ids []int64, err error)

	FetchUpstreamProxyKeys(ctx context.Context, ids []int64) (repokeys []string, err error)
	Count(ctx context.Context) (int64, error)

	// GetIDsByParentSpace returns all registry IDs under a given parent space
	GetIDsByParentSpace(ctx context.Context, parentSpaceID int64) ([]int64, error)

	// UpdateParentSpace updates the parent space ID for all registries under a given source space to target space
	UpdateParentSpace(ctx context.Context, srcSpaceID int64, targetSpaceID int64) (int64, error)
}

type RegistryBlobRepository interface {
	LinkBlob(
		ctx context.Context, imageName string,
		registry *types.Registry, blobID int64,
	) error
	UnlinkBlob(
		ctx context.Context, imageName string,
		registry *types.Registry, blobID int64,
	) (bool, error)

	UnlinkBlobByImageName(
		ctx context.Context, registryID int64,
		imageName string,
	) (bool, error)
}

type ImageRepository interface {
	GetByUUID(ctx context.Context, uuid string) (*types.Image, error)
	// Get an Artifact specified by ID
	Get(ctx context.Context, id int64) (*types.Image, error)
	// Get an Artifact specified by Artifact Name
	GetByName(
		ctx context.Context, registryID int64,
		name string,
	) (*types.Image, error)

	GetByNameAndType(
		ctx context.Context, registryID int64,
		name string, artifactType *artifact.ArtifactType,
	) (*types.Image, error)

	// Get the Labels specified by Parent ID and Repo
	GetLabelsByParentIDAndRepo(
		ctx context.Context, parentID int64,
		repo string, limit int, offset int,
		search string,
	) (labels []string, err error)
	// Count the Labels specified by Parent ID and Repo
	CountLabelsByParentIDAndRepo(
		ctx context.Context, parentID int64,
		repo, search string,
	) (count int64, err error)
	// Get an Artifact specified by Artifact Name
	GetByRepoAndName(
		ctx context.Context, parentID int64,
		repo string, name string,
	) (*types.Image, error)
	// Create an Image
	CreateOrUpdate(ctx context.Context, image *types.Image) error
	// Update an Image
	Update(ctx context.Context, artifact *types.Image) (err error)

	UpdateStatus(ctx context.Context, artifact *types.Image) (err error)

	DeleteByImageNameAndRegID(ctx context.Context, regID int64, image string) (err error)
	DeleteByImageNameIfNoLinkedArtifacts(ctx context.Context, regID int64, image string) (err error)

	DuplicateImage(ctx context.Context, sourceImage *types.Image, targetRegistryID int64) (*types.Image, error)
}

type ArtifactRepository interface {
	GetByUUID(ctx context.Context, uuid string) (*types.Artifact, error)
	Get(ctx context.Context, id int64) (*types.Artifact, error)
	// Get an Artifact specified by ID
	GetByName(ctx context.Context, imageID int64, version string) (*types.Artifact, error)
	// Get an Artifact specified by RegistryID, image name and version
	GetByRegistryImageAndVersion(
		ctx context.Context, registryID int64, image string, version string,
	) (*types.Artifact, error)
	GetByRegistryImageVersionAndArtifactType(
		ctx context.Context, registryID int64, image string, version string, artifactType string,
	) (*types.Artifact, error)
	// Create an Artifact
	CreateOrUpdate(ctx context.Context, artifact *types.Artifact) (int64, error)
	Count(ctx context.Context) (int64, error)
	GetAllArtifactsByParentID(
		ctx context.Context, id int64,
		i *[]string, field string, order string,
		limit int, offset int, term string,
		version bool, packageTypes []string,
	) (*[]types.ArtifactMetadata, error)
	CountAllArtifactsByParentID(
		ctx context.Context, parentID int64,
		registryIDs *[]string, search string, latestVersion bool, packageTypes []string,
	) (int64, error)
	GetArtifactsByRepo(
		ctx context.Context, parentID int64, repoKey string, sortByField string, sortByOrder string,
		limit int, offset int, search string, labels []string,
		artifactType *artifact.ArtifactType,
	) (*[]types.ArtifactMetadata, error)
	CountArtifactsByRepo(
		ctx context.Context, parentID int64, repoKey, search string, labels []string,
		artifactType *artifact.ArtifactType,
	) (int64, error)
	GetLatestArtifactMetadata(
		ctx context.Context, id int64, identifier string,
		image string,
	) (*types.ArtifactMetadata, error)
	GetAllVersionsByRepoAndImage(
		ctx context.Context, id int64, image string, field string, order string, limit int,
		offset int, term string, artifactType *artifact.ArtifactType,
	) (*[]types.NonOCIArtifactMetadata, error)
	CountAllVersionsByRepoAndImage(
		ctx context.Context, parentID int64, repoKey string, image string,
		search string, artifactType *artifact.ArtifactType,
	) (int64, error)
	GetArtifactMetadata(
		ctx context.Context, id int64, identifier string, image string, version string,
		artifactType *artifact.ArtifactType,
	) (*types.ArtifactMetadata, error)
	UpdateArtifactMetadata(
		ctx context.Context, metadata json.RawMessage,
		artifactID int64,
	) (err error)

	GetByRegistryIDAndImage(ctx context.Context, registryID int64, image string) (
		*[]types.Artifact,
		error,
	)

	DeleteByImageNameAndRegistryID(ctx context.Context, regID int64, image string) (err error)

	DeleteByVersionAndImageName(ctx context.Context, image string, version string, regID int64) (err error)
	GetLatestByImageID(ctx context.Context, imageID int64) (*types.Artifact, error)

	// get latest artifacts from all images under repo
	GetLatestArtifactsByRepo(
		ctx context.Context, registryID int64, batchSize int, artifactID int64,
	) (*[]types.ArtifactMetadata, error)

	// get all artifacts from all images under repo
	GetAllArtifactsByRepo(
		ctx context.Context, registryID int64, batchSize int, artifactID int64,
	) (*[]types.ArtifactMetadata, error)

	GetArtifactsByRepoAndImageBatch(
		ctx context.Context, registryID int64, imageName string, batchSize int, artifactID int64,
	) (*[]types.ArtifactMetadata, error)

	SearchLatestByName(
		ctx context.Context, regID int64, name string, limit int, offset int,
	) (*[]types.Artifact, error)

	CountLatestByName(
		ctx context.Context, regID int64, name string,
	) (int64, error)

	SearchByImageName(
		ctx context.Context, regID int64, name string,
		limit int, offset int,
	) (*[]types.ArtifactMetadata, error)

	CountByImageName(
		ctx context.Context, regID int64, name string,
	) (int64, error)

	// DuplicateArtifact creates a copy of an artifact with a different image ID and created by user
	DuplicateArtifact(
		ctx context.Context, sourceArtifact *types.Artifact, targetImageID int64,
	) (*types.Artifact, error)
}

type DownloadStatRepository interface {
	Create(ctx context.Context, downloadStat *types.DownloadStat) error
	GetTotalDownloadsForImage(ctx context.Context, imageID int64) (int64, error)
	GetTotalDownloadsForManifests(
		ctx context.Context,
		artifactVersion []string,
		imageID int64,
	) (map[string]int64, error)
	CreateByRegistryIDImageAndArtifactName(ctx context.Context, regID int64, image string, artifactName string) error
	GetTotalDownloadsForArtifactID(ctx context.Context, artifactID int64) (int64, error)
}

type BandwidthStatRepository interface {
	Create(ctx context.Context, bandwidthStat *types.BandwidthStat) error
}

type GCBlobTaskRepository interface {
	FindAll(ctx context.Context) ([]*types.GCBlobTask, error)
	FindAndLockBefore(
		ctx context.Context, blobID int64,
		date time.Time,
	) (*types.GCBlobTask, error)
	Count(ctx context.Context) (int, error)
	Next(ctx context.Context) (*types.GCBlobTask, error)
	Reschedule(ctx context.Context, b *types.GCBlobTask, d time.Duration) error
	Postpone(ctx context.Context, b *types.GCBlobTask, d time.Duration) error
	IsDangling(ctx context.Context, b *types.GCBlobTask) (bool, error)
	Delete(ctx context.Context, b *types.GCBlobTask) error
}

type GCManifestTaskRepository interface {
	FindAndLock(
		ctx context.Context, registryID,
		manifestID int64,
	) (*types.GCManifestTask, error)
	FindAndLockBefore(
		ctx context.Context, registryID, manifestID int64,
		date time.Time,
	) (*types.GCManifestTask, error)
	FindAndLockNBefore(
		ctx context.Context, registryID int64,
		manifestIDs []int64, date time.Time,
	) ([]*types.GCManifestTask, error)
	Next(ctx context.Context) (*types.GCManifestTask, error)
	Postpone(ctx context.Context, b *types.GCManifestTask, d time.Duration) error
	IsDangling(ctx context.Context, b *types.GCManifestTask) (bool, error)
	Delete(ctx context.Context, b *types.GCManifestTask) error
	DeleteManifest(ctx context.Context, registryID, id int64) (*digest.Digest, error)
}

type NodesRepository interface {
	// Get a node specified by ID
	Get(ctx context.Context, id string) (*types.Node, error)
	// Get a node specified by node Name and registry id
	GetByNameAndRegistryID(
		ctx context.Context, registryID int64,
		name string,
	) (*types.Node, error)

	FindByPathAndRegistryID(
		ctx context.Context, registryID int64, pathPrefix string, filename string,
	) (*types.Node, error)

	FindByPathsAndRegistryID(ctx context.Context, paths []string, registryID int64) (*[]string, error)

	CountByPathAndRegistryID(
		ctx context.Context, registryID int64, path string,
	) (int64, error)
	// Create a node
	Create(ctx context.Context, node *types.Node) error
	// delete a node
	DeleteByID(ctx context.Context, id int64) (err error)

	GetByPathAndRegistryID(
		ctx context.Context, registryID int64,
		path string,
	) (*types.Node, error)

	// GetByBlobIDAndRegistryID retrieves a node by its blob ID and registry ID.
	GetByBlobIDAndRegistryID(ctx context.Context, blobID string, registryID int64) (*types.Node, error)

	GetFilesMetadataByPathAndRegistryID(
		ctx context.Context, registryID int64, path string,
		sortByField string,
		sortByOrder string,
		limit int,
		offset int,
		search string,
	) (*[]types.FileNodeMetadata, error)

	GetFileMetadataByPathAndRegistryID(
		ctx context.Context,
		registryID int64,
		path string,
	) (*types.FileNodeMetadata, error)

	DeleteByNodePathAndRegistryID(ctx context.Context, nodePath string, regID int64) (err error)
	DeleteByLeafNodePathAndRegistryID(ctx context.Context, nodePath string, regID int64) (err error)

	GetAllFileNodesByPathPrefixAndRegistryID(
		ctx context.Context, registryID int64, pathPrefix string,
	) (*[]types.Node, error)
}

type GenericBlobRepository interface {
	FindByID(ctx context.Context, id string) (*types.GenericBlob, error)
	FindBySha256AndRootParentID(
		ctx context.Context, sha256 string,
		rootParentID int64,
	) (*types.GenericBlob, error)
	Create(ctx context.Context, gb *types.GenericBlob) (string, bool, error)
	DeleteByID(ctx context.Context, id string) error
	TotalSizeByRootParentID(ctx context.Context, id int64) (int64, error)
}

type WebhooksRepository interface {
	Create(ctx context.Context, webhook *gitnesstypes.WebhookCore) error
	GetByRegistryAndIdentifier(
		ctx context.Context,
		registryID int64,
		webhookIdentifier string,
	) (*gitnesstypes.WebhookCore, error)
	Find(ctx context.Context, webhookID int64) (*gitnesstypes.WebhookCore, error)
	ListByRegistry(
		ctx context.Context,
		sortByField string,
		sortByOrder string,
		limit int,
		offset int,
		search string,
		registryID int64,
	) ([]*gitnesstypes.WebhookCore, error)
	ListAllByRegistry(
		ctx context.Context,
		parents []gitnesstypes.WebhookParentInfo,
	) ([]*gitnesstypes.WebhookCore, error)
	CountAllByRegistry(
		ctx context.Context,
		registryID int64,
		search string,
	) (int64, error)

	Update(ctx context.Context, webhook *gitnesstypes.WebhookCore) error
	DeleteByRegistryAndIdentifier(ctx context.Context, registryID int64, webhookIdentifier string) error
	UpdateOptLock(
		ctx context.Context, hook *gitnesstypes.WebhookCore,
		mutateFn func(hook *gitnesstypes.WebhookCore) error,
	) (*gitnesstypes.WebhookCore, error)

	// UpdateParentSpace updates the parent space ID for all registry webhooks under a given source space.
	UpdateParentSpace(ctx context.Context, srcSpaceID int64, targetSpaceID int64) (int64, error)

	// UpdateSecretSpaceID updates the secret space ID for all webhooks referencing secrets from source space.
	UpdateSecretSpaceID(ctx context.Context, srcSpaceID int64, targetSpaceID int64) (int64, error)
}

type WebhooksExecutionRepository interface {
	Find(ctx context.Context, id int64) (*gitnesstypes.WebhookExecutionCore, error)

	// Create creates a new webhook execution entry.
	Create(ctx context.Context, hook *gitnesstypes.WebhookExecutionCore) error

	// ListForWebhook lists the webhook executions for a given webhook id.
	ListForWebhook(
		ctx context.Context,
		webhookID int64,
		limit int,
		page int,
		size int,
	) ([]*gitnesstypes.WebhookExecutionCore, error)

	CountForWebhook(ctx context.Context, webhookID int64) (int64, error)

	// ListForTrigger lists the webhook executions for a given trigger id.
	ListForTrigger(ctx context.Context, triggerID string) ([]*gitnesstypes.WebhookExecutionCore, error)
}

type PackageTagRepository interface {
	FindByImageNameAndRegID(ctx context.Context, image string, regID int64) ([]*types.PackageTagMetadata, error)

	Create(ctx context.Context, tag *types.PackageTag) (string, error)

	DeleteByTagAndImageName(ctx context.Context, tag string, image string, regID int64) error
	DeleteByImageNameAndRegID(ctx context.Context, image string, regID int64) error
}

type TaskRepository interface {
	Find(ctx context.Context, key string) (*types.Task, error)

	UpsertTask(ctx context.Context, task *types.Task) error

	LockForUpdate(ctx context.Context, task *types.Task) (types.TaskStatus, error)

	SetRunAgain(ctx context.Context, taskKey string, runAgain bool) error

	UpdateStatus(ctx context.Context, taskKey string, status types.TaskStatus) error

	CompleteTask(ctx context.Context, key string, status types.TaskStatus) (bool, error)

	ListPendingTasks(ctx context.Context, limit int) ([]*types.Task, error)
}

type TaskSourceRepository interface {
	FindByTaskKeyAndSourceType(ctx context.Context, key string, sourceType string) (*types.TaskSource, error)

	InsertSource(ctx context.Context, key string, source types.SourceRef) error

	ClaimSources(ctx context.Context, key string, runID string) error

	UpdateSourceStatus(ctx context.Context, runID string, status types.TaskStatus, errMsg string) error
}
type TaskEventRepository interface {
	LogTaskEvent(ctx context.Context, key string, event string, payload []byte) error
}

type QuarantineArtifactRepository interface {
	Create(ctx context.Context, artifact *types.QuarantineArtifact) error
	GetByFilePath(
		ctx context.Context, filePath string,
		registryID int64,
		artifact string,
		version string,
		artifactType *artifact.ArtifactType,
	) ([]*types.QuarantineArtifact, error)
	DeleteByRegistryIDArtifactAndFilePath(
		ctx context.Context, registryID int64,
		artifactID *int64, imageID int64, nodeID *string,
	) error
}
