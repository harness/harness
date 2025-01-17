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
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/lib/pq"
	"github.com/opencontainers/go-digest"
)

type MediaTypesRepository interface {
	MapMediaType(ctx context.Context, mediaType string) (int64, error)
	MediaTypeExists(ctx context.Context, mediaType string) (bool, error)
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
	CreateOrFind(ctx context.Context, b *types.Blob) (*types.Blob, error)
	DeleteByID(ctx context.Context, id int64) error
	ExistsBlob(
		ctx context.Context, repoID int64, d digest.Digest,
		image string,
	) (bool, error)
}

type CleanupPolicyRepository interface {
	// GetIdsByRegistryId the CleanupPolicy Ids specified by Registry Key
	GetIDsByRegistryID(ctx context.Context, id int64) (ids []int64, err error)
	// GetByRegistryId the CleanupPolicy specified by Registry Key
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
	Get(ctx context.Context, manifestID int64) (*types.Manifest, error)
	DeleteManifest(
		ctx context.Context, repoID int64,
		imageName string, d digest.Digest,
	) (bool, error)
	ListManifestsBySubject(
		ctx context.Context, repoID int64,
		id int64,
	) (types.Manifests, error)
	ListManifestsBySubjectDigest(
		ctx context.Context, repoID int64,
		digest types.Digest,
	) (types.Manifests, error)
	DeleteManifestsByImageName(ctx context.Context, registryID int64, imageName string) (err error)
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

	CountAllArtifactsByParentID(
		ctx context.Context, parentID int64,
		registryIDs *[]string, search string,
		latestVersion bool, packageTypes []string,
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
	) (*types.TagMetadata, error)

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
	) (*[]types.TagMetadata, error)

	DeleteTag(ctx context.Context, registryID int64, imageName string, name string) (err error)

	CountAllTagsByRepoAndImage(
		ctx context.Context, parentID int64, repoKey string,
		image string, search string,
	) (int64, error)
	FindTag(
		ctx context.Context, repoID int64, imageName string,
		name string,
	) (*types.Tag, error)
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

	// Delete the upstreamProxyConfig specified by registry key
	Delete(ctx context.Context, parentID int64, repoKey string) (err error)

	// Update updates the upstreamproxy.
	Update(ctx context.Context, upstreamproxyRecord *types.UpstreamProxyConfig) (err error)

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
	RegID         string
	RegIdentifier string
	Description   string
	PackageType   artifact.PackageType
	Type          artifact.RegistryType
	LastModified  time.Time
	URL           string
	Labels        pq.StringArray
	ArtifactCount int64
	DownloadCount int64
	Size          int64
}

type RegistryRepository interface {
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
		parentID int64,
		packageTypes []string,
		sortByField string,
		sortByOrder string,
		limit int,
		offset int,
		search string,
		repoType string,
		recursive bool,
	) (repos *[]RegistryMetadata, err error)

	CountAll(
		ctx context.Context, parentID int64, packageTypes []string,
		search string, repoType string,
	) (count int64, err error)

	FetchUpstreamProxyIDs(
		ctx context.Context, repokeys []string,
		parentID int64,
	) (ids []int64, err error)

	FetchRegistriesIDByUpstreamProxyID(
		ctx context.Context, upstreamProxyID string,
		parentID int64,
	) (ids []int64, err error)

	FetchUpstreamProxyKeys(ctx context.Context, ids []int64) (repokeys []string, err error)
	Count(ctx context.Context) (int64, error)
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
}

type ImageRepository interface {
	// Get an Artifact specified by ID
	Get(ctx context.Context, id int64) (*types.Image, error)
	// Get an Artifact specified by Artifact Name
	GetByName(
		ctx context.Context, registryID int64,
		name string,
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

	DeleteByRegistryID(ctx context.Context, registryID int64) (err error)
	DeleteBandwidthStatByRegistryID(ctx context.Context, registryID int64) (err error)
	DeleteDownloadStatByRegistryID(ctx context.Context, registryID int64) (err error)
}

type ArtifactRepository interface {
	// Get an Artifact specified by ID
	GetByName(ctx context.Context, imageID int64, version string) (*types.Artifact, error)
	// Create an Artifact
	CreateOrUpdate(ctx context.Context, artifact *types.Artifact) error
	Count(ctx context.Context) (int64, error)
}

type DownloadStatRepository interface {
	Create(ctx context.Context, downloadStat *types.DownloadStat) error
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
	Get(ctx context.Context, id int64) (*types.Node, error)
	// Get a node specified by node Name and registry id
	GetByNameAndRegistryId(
		ctx context.Context, registryID int64,
		name string,
	) (*types.Node, error)
	// Create a node
	Create(ctx context.Context, node *types.Node) error
	// delete a node
	DeleteById(ctx context.Context, id int64) (err error)

	GetByPathAndRegistryId(
		ctx context.Context, registryID int64,
		path string,
	) (*types.Node, error)
}

type GenericBlobRepository interface {
	FindByID(ctx context.Context, id string) (*types.GenericBlob, error)
	FindBySha256AndRootParentID(
		ctx context.Context, sha256 string,
		rootParentID int64,
	) (*types.GenericBlob, error)
	Create(ctx context.Context, gb *types.GenericBlob) error
	DeleteByID(ctx context.Context, id string) error
}
