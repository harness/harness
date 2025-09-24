// Source: https://gitlab.com/gitlab-org/container-registry

// Copyright 2019 Gitlab Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/event"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/manifest/ocischema"
	"github.com/harness/gitness/registry/app/manifest/schema2"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/gc"
	"github.com/harness/gitness/registry/services/webhook"
	"github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"
	db "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/registry/api/errcode"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog/log"
)

type manifestService struct {
	registryDao             store.RegistryRepository
	manifestDao             store.ManifestRepository
	layerDao                store.LayerRepository
	blobRepo                store.BlobRepository
	mtRepository            store.MediaTypesRepository
	tagDao                  store.TagRepository
	imageDao                store.ImageRepository
	artifactDao             store.ArtifactRepository
	manifestRefDao          store.ManifestReferenceRepository
	ociImageIndexMappingDao store.OCIImageIndexMappingRepository
	spaceFinder             refcache.SpaceFinder
	gcService               gc.Service
	tx                      dbtx.Transactor
	reporter                event.Reporter
	artifactEventReporter   registryevents.Reporter
	urlProvider             urlprovider.Provider
	untaggedImagesEnabled   func(ctx context.Context) bool
}

func NewManifestService(
	registryDao store.RegistryRepository, manifestDao store.ManifestRepository,
	blobRepo store.BlobRepository, mtRepository store.MediaTypesRepository, tagDao store.TagRepository,
	imageDao store.ImageRepository, artifactDao store.ArtifactRepository,
	layerDao store.LayerRepository, manifestRefDao store.ManifestReferenceRepository,
	tx dbtx.Transactor, gcService gc.Service, reporter event.Reporter, spaceFinder refcache.SpaceFinder,
	ociImageIndexMappingDao store.OCIImageIndexMappingRepository, artifactEventReporter registryevents.Reporter,
	urlProvider urlprovider.Provider, untaggedImagesEnabled func(ctx context.Context) bool,
) ManifestService {
	return &manifestService{
		registryDao:             registryDao,
		manifestDao:             manifestDao,
		layerDao:                layerDao,
		blobRepo:                blobRepo,
		mtRepository:            mtRepository,
		tagDao:                  tagDao,
		artifactDao:             artifactDao,
		imageDao:                imageDao,
		manifestRefDao:          manifestRefDao,
		gcService:               gcService,
		tx:                      tx,
		reporter:                reporter,
		spaceFinder:             spaceFinder,
		ociImageIndexMappingDao: ociImageIndexMappingDao,
		artifactEventReporter:   artifactEventReporter,
		urlProvider:             urlProvider,
		untaggedImagesEnabled:   untaggedImagesEnabled,
	}
}

type ManifestService interface {
	// GetTags gets the tags of a repository
	DBTag(
		ctx context.Context,
		mfst manifest.Manifest,
		d digest.Digest,
		tag string,
		headers *commons.ResponseHeaders,
		info pkg.RegistryInfo,
	) error
	DBPut(
		ctx context.Context,
		mfst manifest.Manifest,
		d digest.Digest,
		headers *commons.ResponseHeaders,
		info pkg.RegistryInfo,
	) error
	DeleteTag(ctx context.Context, repoKey string, tag string, info pkg.RegistryInfo) (bool, error)
	DeleteManifest(ctx context.Context, repoKey string, d digest.Digest, info pkg.RegistryInfo) error
	AddManifestAssociation(ctx context.Context, repoKey string, digest digest.Digest, info pkg.RegistryInfo) error
	DBFindRepositoryBlob(
		ctx context.Context, desc manifest.Descriptor, repoID int64, imageName string,
	) (*types.Blob, error)
	UpsertImage(ctx context.Context, info pkg.RegistryInfo) error
}

func (l *manifestService) DBTag(
	ctx context.Context,
	mfst manifest.Manifest,
	d digest.Digest,
	tag string,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	imageName := info.Image

	if err := l.dbTagManifest(ctx, d, tag, imageName, info); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create tag in database")
		err2 := l.handleTagError(ctx, mfst, d, tag, headers, info, err, imageName)
		if err2 != nil {
			return err2
		}
	}

	return nil
}

func (l *manifestService) handleTagError(
	ctx context.Context,
	mfst manifest.Manifest,
	d digest.Digest,
	tag string,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
	err error,
	imageName string,
) error {
	if errors.Is(err, util.ErrManifestNotFound) {
		// If online GC was already reviewing the manifest that we want to tag, and that manifest had no
		// tags before the review start, the API is unable to stop the GC from deleting the manifest (as
		// the GC already acquired the lock on the corresponding queue row). This means that once the API
		// is unblocked and tries to create the tag, a foreign key violation error will occur (because we're
		// trying to create a tag for a manifest that no longer exists) and lead to this specific error.
		// This should be extremely rare, if it ever occurs, but if it does, we should recreate the manifest
		// and tag it, instead of returning a "manifest not found response" to clients. It's expected that
		// this route handles the creation of a manifest if it doesn't exist already.
		if err = l.DBPut(ctx, mfst, "", headers, info); err != nil {
			return fmt.Errorf("failed to recreate manifest in database: %w", err)
		}
		if err = l.dbTagManifest(ctx, d, tag, imageName, info); err != nil {
			return fmt.Errorf("failed to create tag in database after manifest recreate: %w", err)
		}
	} else {
		return fmt.Errorf("failed to create tag in database: %w", err)
	}
	return nil
}

func (l *manifestService) dbTagManifest(
	ctx context.Context,
	dgst digest.Digest,
	tagName, imageName string,
	info pkg.RegistryInfo,
) error {
	dbRegistry := info.Registry
	newDigest, err := types.NewDigest(dgst)
	if err != nil {
		return formatFailedToTagErr(err)
	}
	dbManifest, err := l.manifestDao.FindManifestByDigest(ctx, dbRegistry.ID, info.Image, newDigest)
	if errors.Is(err, gitnessstore.ErrResourceNotFound) {
		return fmt.Errorf("manifest %s not found in database", dgst)
	}
	if err != nil {
		return formatFailedToTagErr(err)
	}
	err = l.tx.WithTx(ctx, func(ctx context.Context) error {
		// Prevent long running transactions by setting an upper limit of manifestTagGCLockTimeout. If the GC is holding
		// the lock of a related review record, the processing there should be fast enough to avoid this. Regardless, we
		// should not let transactions open (and clients waiting) for too long. If this sensible timeout is exceeded, abort
		// the tag creation and let the client retry. This will bubble up and lead to a 503 Service Unavailable response.
		// Set timeout for the transaction to prevent long-running operations
		ctx, cancel := context.WithTimeout(ctx, manifestTagGCLockTimeout)
		defer cancel()

		// Attempt to find and lock the manifest for GC review
		if err := l.lockManifestForGC(ctx, dbRegistry.ID, dbManifest.ID); err != nil {
			return formatFailedToTagErr(err)
		}

		// Create or update artifact and tag records
		if err := l.upsertTag(ctx, dbRegistry.ID, dbManifest.ID, imageName, tagName); err != nil {
			return formatFailedToTagErr(err)
		}

		return nil
	})

	if err != nil {
		return formatFailedToTagErr(err)
	}
	spacePath, packageType, err := l.getSpacePathAndPackageType(ctx, &dbRegistry)
	if err == nil {
		reg := info.Registry
		if !l.untaggedImagesEnabled(ctx) {
			l.reportEventAsync(
				ctx, reg.ID, info.RegIdentifier, imageName, tagName, packageType,
				spacePath, dbManifest.ID,
			)
		}
		session, _ := request.AuthSessionFrom(ctx)
		createPayload := webhook.GetArtifactCreatedPayload(ctx, info, session.Principal.ID,
			reg.ID, reg.Name, tagName, dgst.String(), l.urlProvider)
		l.artifactEventReporter.ArtifactCreated(ctx, &createPayload)
	} else {
		log.Ctx(ctx).Err(err).Msg("Failed to find spacePath, not publishing event")
	}

	return nil
}

func formatFailedToTagErr(err error) error {
	return fmt.Errorf("failed to tag manifest: %w", err)
}

// Locks the manifest for GC review.
func (l *manifestService) lockManifestForGC(ctx context.Context, repoID, manifestID int64) error {
	_, err := l.gcService.ManifestFindAndLockBefore(
		ctx, repoID, manifestID,
		time.Now().Add(manifestTagGCReviewWindow),
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// Use ProcessSQLErrorf for handling the SQL error abstraction
		return db.ProcessSQLErrorf(
			ctx,
			err,
			"failed to lock manifest for GC review [repoID: %d, manifestID: %d]", repoID, manifestID,
		)
	}
	return nil
}

// Creates or updates artifact and tag records.
func (l *manifestService) upsertTag(
	ctx context.Context,
	registryID,
	manifestID int64,
	imageName,
	tagName string,
) error {
	tag := &types.Tag{
		Name:       tagName,
		ImageName:  imageName,
		RegistryID: registryID,
		ManifestID: manifestID,
	}

	return l.tagDao.CreateOrUpdate(ctx, tag)
}

// Retrieves the spacePath and packageType.
func (l *manifestService) getSpacePathAndPackageType(
	ctx context.Context,
	dbRepo *types.Registry,
) (string, event.PackageType, error) {
	spacePath, err := l.spaceFinder.FindByID(ctx, dbRepo.ParentID)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Failed to find spacePath")
		return "", event.PackageType(0), err
	}

	packageType, err := event.GetPackageTypeFromString(string(dbRepo.PackageType))
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Failed to find packageType")
		return "", event.PackageType(0), err
	}

	return spacePath.Path, packageType, nil
}

// Reports event asynchronously.
func (l *manifestService) reportEventAsync(
	ctx context.Context,
	regID int64,
	regName,
	imageName,
	version string,
	packageType event.PackageType,
	spacePath string,
	manifestID int64,
) {
	artifactDetails := &event.ArtifactDetails{
		RegistryID:   regID,
		RegistryName: regName,
		PackageType:  packageType,
		ManifestID:   manifestID,
		ImagePath:    imageName + ":" + version,
	}
	//Todo: update this to include digest instead of tag after STO step fix
	// if l.untaggedImagesEnabled(ctx) {
	//	artifactDetails.ImagePath = imageName + "@" + version
	// } else {
	//	artifactDetails.ImagePath = imageName + ":" + version
	// }

	go l.reporter.ReportEvent(ctx, artifactDetails, spacePath)
}

func (l *manifestService) DBPut(
	ctx context.Context,
	mfst manifest.Manifest,
	d digest.Digest,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	_, payload, err := mfst.Payload()
	if err != nil {
		return err
	}

	err = l.dbPutManifest(ctx, mfst, payload, d, headers, info)
	if err == nil && l.untaggedImagesEnabled(ctx) {
		dgst, err := types.NewDigest(d)
		if err != nil {
			return err
		}
		dbManifest, err := l.manifestDao.FindManifestByDigest(ctx, info.Registry.ID, info.Image, dgst)
		if err != nil {
			return err
		}
		spacePath, packageType, err := l.getSpacePathAndPackageType(ctx, &info.Registry)
		if err != nil {
			return err
		}
		l.reportEventAsync(
			ctx, info.Registry.ID, info.RegIdentifier, info.Image, info.Tag, packageType,
			spacePath, dbManifest.ID,
		)
	}
	var mtErr util.UnknownMediaTypeError
	if errors.As(err, &mtErr) {
		return errcode.ErrorCodeManifestInvalid.WithDetail(mtErr.Error())
	}
	return err
}

func (l *manifestService) dbPutManifest(
	ctx context.Context,
	manifest manifest.Manifest,
	payload []byte,
	d digest.Digest,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	switch reqManifest := manifest.(type) {
	case *schema2.DeserializedManifest:
		log.Ctx(ctx).Debug().Msgf("Putting schema2 manifest %s to database", d.String())
		if err := l.dbPutManifestSchema2(ctx, reqManifest, payload, d, headers, info); err != nil {
			return err
		}
		return l.upsertImageAndArtifact(ctx, d, info)
	case *ocischema.DeserializedManifest:
		log.Ctx(ctx).Debug().Msgf("Putting ocischema manifest %s to database", d.String())
		if err := l.dbPutManifestOCI(ctx, reqManifest, payload, d, headers, info); err != nil {
			return err
		}
		return l.upsertImageAndArtifact(ctx, d, info)
	case *manifestlist.DeserializedManifestList:
		log.Ctx(ctx).Debug().Msgf("Putting manifestlist manifest %s to database", d.String())
		return l.dbPutManifestList(ctx, reqManifest, payload, d, headers, info)
	case *ocischema.DeserializedImageIndex:
		log.Ctx(ctx).Debug().Msgf("Putting ocischema image index %s to database", d.String())
		return l.dbPutImageIndex(ctx, reqManifest, payload, d, headers, info)
	default:
		log.Ctx(ctx).Info().Msgf("Invalid manifest type: %T", reqManifest)
		return errcode.ErrorCodeManifestInvalid.WithDetail("manifest type unsupported")
	}
}

func (l *manifestService) upsertImageAndArtifact(ctx context.Context, d digest.Digest, info pkg.RegistryInfo) error {
	dbRepo := info.Registry
	dbImage := &types.Image{
		Name:       info.Image,
		RegistryID: dbRepo.ID,
		Enabled:    true,
	}

	if err := l.imageDao.CreateOrUpdate(ctx, dbImage); err != nil {
		return err
	}

	dgst, err := types.NewDigest(d)
	if err != nil {
		return err
	}
	dbArtifact := &types.Artifact{
		ImageID: dbImage.ID,
		Version: dgst.String(),
	}

	if _, err := l.artifactDao.CreateOrUpdate(ctx, dbArtifact); err != nil {
		return err
	}
	return nil
}

func (l *manifestService) UpsertImage(
	ctx context.Context,
	info pkg.RegistryInfo,
) error {
	dbRepo := info.Registry
	image, err := l.imageDao.GetByName(ctx, dbRepo.ID, info.Image)
	if err != nil && !errors.Is(err, gitnessstore.ErrResourceNotFound) {
		return err
	} else if image != nil {
		return nil
	}

	dbImage := &types.Image{
		Name:       info.Image,
		RegistryID: dbRepo.ID,
		Enabled:    false,
	}

	if err := l.imageDao.CreateOrUpdate(ctx, dbImage); err != nil {
		return err
	}
	return nil
}

func (l *manifestService) dbPutManifestSchema2(
	ctx context.Context,
	manifest *schema2.DeserializedManifest,
	payload []byte,
	d digest.Digest,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	return l.dbPutManifestV2(ctx, manifest, payload, false, d, headers, info)
}

func (l *manifestService) dbPutManifestV2(
	ctx context.Context,
	mfst manifest.ManifestV2,
	payload []byte,
	nonConformant bool,
	digest digest.Digest,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	// find target repository
	dbRepo := info.Registry

	// Find the config now to ensure that the config's blob is associated with the repository.
	dbCfgBlob, err := l.DBFindRepositoryBlob(ctx, mfst.Config(), dbRepo.ID, info.Image)
	if err != nil {
		return err
	}

	dgst, err := types.NewDigest(digest)
	if err != nil {
		return err
	}

	dbManifest, err := l.manifestDao.FindManifestByDigest(ctx, dbRepo.ID, info.Image, dgst)
	if err != nil && !errors.Is(err, gitnessstore.ErrResourceNotFound) {
		return err
	}

	if dbManifest != nil {
		return nil
	}

	log.Ctx(ctx).Debug().Msgf("manifest %s not found in database", dgst.String())

	cfg := &types.Configuration{
		MediaType: mfst.Config().MediaType,
		Digest:    dbCfgBlob.Digest,
		BlobID:    dbCfgBlob.ID,
	}

	//TODO: check if we need to store the config payload in the database

	// skip retrieval and caching of config payload if its size is over the limit
	/*if dbCfgBlob.Size <= datastore.ConfigSizeLimit {
		// Since filesystem writes may be optional, We cannot be sure that the
		// repository scoped filesystem blob service will have a link to the
		// configuration blob; however, since we check for repository scoped access
		// via the database above, we may retrieve the blob directly common storage.
		cfgPayload, err := imh.blobProvider.Get(imh, dbCfgBlob.Digest)
		if err != nil {
			return err
		}
		cfg.Payload = cfgPayload
	}*/

	m := &types.Manifest{
		RegistryID:    dbRepo.ID,
		TotalSize:     mfst.TotalSize(),
		SchemaVersion: mfst.Version().SchemaVersion,
		MediaType:     mfst.Version().MediaType,
		Digest:        digest,
		Payload:       payload,
		Configuration: cfg,
		NonConformant: nonConformant,
		ImageName:     info.Image,
	}

	var artifactMediaType sql.NullString
	ocim, ok := mfst.(manifest.ManifestOCI)
	if ok {
		subjectHandlingError := l.handleSubject(
			ctx, ocim.Subject(), ocim.ArtifactType(),
			ocim.Annotations(), &dbRepo, m, headers, info,
		)
		if subjectHandlingError != nil {
			return subjectHandlingError
		}
		if ocim.ArtifactType() != "" {
			artifactMediaType.Valid = true
			artifactMediaType.String = ocim.ArtifactType()
			m.ArtifactType = artifactMediaType
		}
	} else if mfst.Config().MediaType != "" {
		artifactMediaType.Valid = true
		artifactMediaType.String = mfst.Config().MediaType
		m.ArtifactType = artifactMediaType
	}

	// check if the manifest references non-distributable layers and mark it as such on the DB
	ll := mfst.DistributableLayers()
	m.NonDistributableLayers = len(ll) < len(mfst.Layers())

	// Use CreateOrFind to prevent race conditions while pushing the same manifest with digest for different tags
	if err := l.manifestDao.CreateOrFind(ctx, m); err != nil {
		return err
	}

	dbManifest = m

	// find and associate distributable manifest layer blobs
	for _, reqLayer := range mfst.DistributableLayers() {
		log.Ctx(ctx).Debug().Msgf("associating layer %s with manifest %s", reqLayer.Digest.String(), digest.String())
		dbBlob, err := l.DBFindRepositoryBlob(ctx, reqLayer, dbRepo.ID, info.Image)
		if err != nil {
			return err
		}

		// Overwrite the media type from common blob storage with the one
		// specified in the manifest json for the layer entity. The layer entity
		// has a 1-1 relationship with with the manifest, so we want to reflect
		// the manifest's description of the layer. Multiple manifest can reference
		// the same blob, so the common blob storage should remain generic.
		if ok2 := l.layerMediaTypeExists(ctx, reqLayer.MediaType); ok2 {
			dbBlob.MediaType = reqLayer.MediaType
		}

		if err2 := l.layerDao.AssociateLayerBlob(ctx, dbManifest, dbBlob); err2 != nil {
			return err2
		}
	}

	return nil
}

func (l *manifestService) DBFindRepositoryBlob(
	ctx context.Context, desc manifest.Descriptor,
	repoID int64, imageName string,
) (*types.Blob, error) {
	b, err := l.blobRepo.FindByDigestAndRepoID(ctx, desc.Digest, repoID, imageName)
	if err != nil {
		if errors.Is(err, gitnessstore.ErrResourceNotFound) {
			return nil, fmt.Errorf("blob not found in database")
		}
		return nil, err
	}
	return b, nil
}

// AddManifestAssociation This updates the manifestRefs for all new childDigests to their already existing parent
// manifests. This is used when a manifest from a manifest list is pulled from the remote and manifest list already
// exists in the database.
func (l *manifestService) AddManifestAssociation(
	ctx context.Context, repoKey string, childDigest digest.Digest, info pkg.RegistryInfo,
) error {
	newDigest, err2 := types.NewDigest(childDigest)
	if err2 != nil {
		return fmt.Errorf("failed to create digest: %s %w", childDigest, err2)
	}
	r, err := l.registryDao.GetByParentIDAndName(ctx, info.ParentID, repoKey)
	if err != nil {
		return fmt.Errorf("failed to get registry: %s %w", repoKey, err)
	}
	childManifest, err2 := l.manifestDao.FindManifestByDigest(ctx, r.ID, info.Image, newDigest)
	if err2 != nil {
		return fmt.Errorf("failed to find manifest by digest. Repo: %d Image: %s %w", r.ID, info.Image, err2)
	}
	mappings, err := l.ociImageIndexMappingDao.GetAllByChildDigest(ctx, r.ID, childManifest.ImageName, newDigest)
	if err != nil {
		return fmt.Errorf("failed to get oci image index mappings. Repo: %d Image: %s %w",
			r.ID,
			childManifest.ImageName,
			err)
	}
	for _, mapping := range mappings {
		parentManifest, err := l.manifestDao.Get(ctx, mapping.ParentManifestID)
		if err != nil {
			return fmt.Errorf("failed to get manifest with ID: %d %w", mapping.ParentManifestID, err)
		}
		if err := l.manifestRefDao.AssociateManifest(ctx, parentManifest, childManifest); err != nil {
			if errors.Is(err, util.ErrRefManifestNotFound) {
				// This can only happen if the online GC deleted one
				// of the referenced manifests (because they were
				// untagged/unreferenced) between the call to
				// `FindAndLockNBefore` and `AssociateManifest`. For now
				// we need to return this error to mimic the behaviour
				// of the corresponding filesystem validation.
				log.Error().
					Msgf("Failed to associate manifest Ref Manifest not found. parentDigest:%s childDigest:%s %v",
						parentManifest.Digest.String(),
						childManifest.Digest.String(),
						err)
				return err
			}
		}
	}
	return nil
}

func (l *manifestService) handleSubject(
	ctx context.Context, subject manifest.Descriptor,
	artifactType string, annotations map[string]string, dbRepo *types.Registry,
	m *types.Manifest, headers *commons.ResponseHeaders, info pkg.RegistryInfo,
) error {
	if subject.Digest.String() != "" {
		// Fetch subject_id from digest
		subjectDigest, err := types.NewDigest(subject.Digest)
		if err != nil {
			return err
		}
		dbSubject, err := l.manifestDao.FindManifestByDigest(ctx, dbRepo.ID, info.Image, subjectDigest)
		if err != nil && !errors.Is(err, gitnessstore.ErrResourceNotFound) {
			return err
		}

		if errors.Is(err, gitnessstore.ErrResourceNotFound) {
			// in case something happened to the referenced manifest after validation
			// return distribution.ManifestBlobUnknownError{Digest: subject.Digest}
			log.Ctx(ctx).Warn().Msgf("subject manifest not found in database")
		} else {
			m.SubjectID.Int64 = dbSubject.ID
			m.SubjectID.Valid = true
		}
		m.SubjectDigest = subject.Digest
		headers.Headers["OCI-Subject"] = subject.Digest.String()
	}

	if artifactType != "" {
		m.ArtifactType.String = artifactType
		m.ArtifactType.Valid = true
	}
	m.Annotations = annotations
	return nil
}

func (l *manifestService) dbPutManifestOCI(
	ctx context.Context,
	manifest *ocischema.DeserializedManifest,
	payload []byte,
	d digest.Digest,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	return l.dbPutManifestV2(ctx, manifest, payload, false, d, headers, info)
}

func (l *manifestService) dbPutManifestList(
	ctx context.Context,
	manifestList *manifestlist.DeserializedManifestList,
	payload []byte,
	digest digest.Digest,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	if LikelyBuildxCache(manifestList) {
		return l.dbPutBuildkitIndex(ctx, manifestList, payload, digest, headers, info)
	}

	r := info.Registry
	dgst, err := types.NewDigest(digest)
	if err != nil {
		return err
	}

	ml, err := l.manifestDao.FindManifestByDigest(ctx, r.ID, info.Image, dgst)
	if err != nil && !errors.Is(err, gitnessstore.ErrResourceNotFound) {
		return err
	}

	// Media type can be either Docker (`application/vnd.docker.distribution.manifest.list.v2+json`)
	// or OCI (empty).
	// We need to make it explicit if empty, otherwise we're not able to distinguish between media types.
	mediaType := manifestList.MediaType
	if mediaType == "" {
		mediaType = v1.MediaTypeImageIndex
	}

	ml = &types.Manifest{
		RegistryID:    r.ID,
		SchemaVersion: manifestList.SchemaVersion,
		MediaType:     mediaType,
		Digest:        digest,
		Payload:       payload,
		ImageName:     info.Image,
	}

	mm, ids, err2 := l.validateManifestList(ctx, manifestList, info)
	if err2 != nil {
		return err2
	}

	err = l.tx.WithTx(
		ctx, func(ctx context.Context) error {
			// Prevent long running transactions by setting an upper limit of
			// manifestListCreateGCLockTimeout. If the GC is
			// holding the lock of a related review record, the processing
			// there should be fast enough to avoid this.
			// Regardless, we should not let transactions open (and clients waiting)
			// for too long. If this sensible timeout
			// is exceeded, abort the request and let the client retry.
			// This will bubble up and lead to a 503 Service
			// Unavailable response.
			ctx, cancel := context.WithTimeout(ctx, manifestListCreateGCLockTimeout)
			defer cancel()

			if _, err := l.gcService.ManifestFindAndLockNBefore(
				ctx, r.ID, ids,
				time.Now().Add(manifestListCreateGCReviewWindow),
			); err != nil {
				return err
			}

			// use CreateOrFind to prevent race conditions when the same digest is used by different tags
			// and pushed at the same time
			if err := l.manifestDao.CreateOrFind(ctx, ml); err != nil {
				return err
			}

			// Associate manifests to the manifest list.
			for _, m := range mm {
				if err := l.manifestRefDao.AssociateManifest(ctx, ml, m); err != nil {
					if errors.Is(err, util.ErrRefManifestNotFound) {
						// This can only happen if the online GC deleted one
						// of the referenced manifests (because they were
						// untagged/unreferenced) between the call to
						// `FindAndLockNBefore` and `AssociateManifest`. For now
						// we need to return this error to mimic the behaviour
						// of the corresponding filesystem validation.
						return distribution.ErrManifestVerification{
							distribution.ErrManifestBlobUnknown{Digest: m.Digest},
						}
					}
					return err
				}
			}

			err = l.mapManifestList(ctx, ml.ID, manifestList, &info.Registry)
			if err != nil {
				return fmt.Errorf("failed to map manifest list: %w", err)
			}

			return nil
		},
	)

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to create manifest list in database: %v", err)
		return fmt.Errorf("failed to create manifest list in database: %w", err)
	}
	return nil
}

func (l *manifestService) validateManifestIndex(
	ctx context.Context, manifestList *ocischema.DeserializedImageIndex, r *types.Registry, info pkg.RegistryInfo,
) ([]*types.Manifest, []int64, error) {
	mm := make([]*types.Manifest, 0, len(manifestList.Manifests))
	ids := make([]int64, 0, len(manifestList.Manifests))
	for _, desc := range manifestList.Manifests {
		m, err := l.dbFindManifestListManifest(ctx, r, info.Image, desc.Digest)
		if errors.Is(err, gitnessstore.ErrResourceNotFound) && r.Type == artifact.RegistryTypeUPSTREAM {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		mm = append(mm, m)
		ids = append(ids, m.ID)
	}
	log.Ctx(ctx).Debug().Msgf("validated %d / %d manifests in index", len(mm), len(manifestList.Manifests))
	return mm, ids, nil
}

func (l *manifestService) mapManifestIndex(
	ctx context.Context, mi int64, manifestList *ocischema.DeserializedImageIndex, r *types.Registry,
) error {
	if r.Type != artifact.RegistryTypeUPSTREAM {
		return nil
	}
	for _, desc := range manifestList.Manifests {
		err := l.ociImageIndexMappingDao.Create(ctx, &types.OCIImageIndexMapping{
			ParentManifestID:    mi,
			ChildManifestDigest: desc.Digest,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).
				Msgf("failed to create oci image index manifest for digest %s", desc.Digest)
			return fmt.Errorf("failed to create oci image index manifest: %w", err)
		}
	}
	log.Ctx(ctx).Debug().Msgf("successfully mapped manifest index %d with its manifests", mi)
	return nil
}

func (l *manifestService) validateManifestList(
	ctx context.Context,
	manifestList *manifestlist.DeserializedManifestList,
	info pkg.RegistryInfo,
) ([]*types.Manifest, []int64, error) {
	mm := make([]*types.Manifest, 0, len(manifestList.Manifests))
	ids := make([]int64, 0, len(manifestList.Manifests))
	for _, desc := range manifestList.Manifests {
		m, err := l.dbFindManifestListManifest(ctx, &info.Registry, info.Image, desc.Digest)
		if errors.Is(err, gitnessstore.ErrResourceNotFound) && info.Registry.Type == artifact.RegistryTypeUPSTREAM {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		mm = append(mm, m)
		ids = append(ids, m.ID)
	}
	log.Ctx(ctx).Debug().Msgf("validated %d / %d manifests in list", len(mm), len(manifestList.Manifests))
	return mm, ids, nil
}

func (l *manifestService) mapManifestList(
	ctx context.Context, mi int64, manifestList *manifestlist.DeserializedManifestList, r *types.Registry,
) error {
	if r.Type != artifact.RegistryTypeUPSTREAM {
		return nil
	}
	for _, desc := range manifestList.Manifests {
		err := l.ociImageIndexMappingDao.Create(ctx, &types.OCIImageIndexMapping{
			ParentManifestID:    mi,
			ChildManifestDigest: desc.Digest,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("failed to create oci image index manifest for digest %s", desc.Digest)
			return fmt.Errorf("failed to create oci image index manifest: %w", err)
		}
	}
	log.Ctx(ctx).Debug().Msgf("successfully mapped manifest list %d with its manifests", mi)
	return nil
}

func (l *manifestService) dbPutImageIndex(
	ctx context.Context,
	imageIndex *ocischema.DeserializedImageIndex,
	payload []byte,
	digest digest.Digest,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	r, err := l.registryDao.GetByParentIDAndName(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return err
	}
	if r == nil {
		return errors.New("repository not found in database")
	}

	dgst, err := types.NewDigest(digest)
	if err != nil {
		return err
	}

	mi, err := l.manifestDao.FindManifestByDigest(ctx, r.ID, info.Image, dgst)
	if err != nil && !errors.Is(err, gitnessstore.ErrResourceNotFound) {
		return err
	}

	// Media type can be either Docker (`application/vnd.docker.distribution.manifest.list.v2+json`)
	// or OCI (empty).
	// We need to make it explicit if empty, otherwise we're not able to distinguish
	//  between media types.
	mediaType := imageIndex.MediaType
	if mediaType == "" {
		mediaType = v1.MediaTypeImageIndex
	}

	mi = &types.Manifest{
		RegistryID:    r.ID,
		SchemaVersion: imageIndex.SchemaVersion,
		MediaType:     mediaType,
		Digest:        digest,
		Payload:       payload,
		ImageName:     info.Image,
	}

	subjectHandlingError := l.handleSubject(
		ctx, imageIndex.Subject(), imageIndex.ArtifactType(),
		imageIndex.Annotations(), r, mi, headers, info,
	)
	if subjectHandlingError != nil {
		return subjectHandlingError
	}

	mm, ids, err := l.validateManifestIndex(ctx, imageIndex, r, info)
	if err != nil {
		return fmt.Errorf("failed to map manifest index: %w", err)
	}

	err = l.tx.WithTx(
		ctx, func(ctx context.Context) error {
			// Prevent long running transactions by setting an upper limit of
			// manifestListCreateGCLockTimeout. If the GC is
			// holding the lock of a related review record, the processing
			//  there should be fast enough to avoid this.
			// Regardless, we should not let transactions open (and clients waiting)
			//  for too long. If this sensible timeout
			// is exceeded, abort the request and let the client retry.
			// This will bubble up and lead to a 503 Service
			// Unavailable response.
			ctx, cancel := context.WithTimeout(ctx, manifestListCreateGCLockTimeout)
			defer cancel()

			if _, err := l.gcService.ManifestFindAndLockNBefore(
				ctx, r.ID, ids,
				time.Now().Add(manifestListCreateGCReviewWindow),
			); err != nil {
				return err
			}
			// use CreateOrFind to prevent race conditions when the same digest is used by different tags
			// and pushed at the same time
			if err := l.manifestDao.CreateOrFind(ctx, mi); err != nil {
				return err
			}

			// Associate manifests to the manifest list.
			for _, m := range mm {
				if err := l.manifestRefDao.AssociateManifest(ctx, mi, m); err != nil {
					if errors.Is(err, util.ErrRefManifestNotFound) {
						// This can only happen if the online GC deleted one of the
						// referenced manifests (because they were
						// untagged/unreferenced) between the call to
						// `FindAndLockNBefore` and `AssociateManifest`. For now
						// we need to return this error to mimic the behaviour
						//  of the corresponding filesystem validation.
						return distribution.ErrManifestVerification{
							distribution.ErrManifestBlobUnknown{Digest: m.Digest},
						}
					}
					return err
				}
			}

			err = l.mapManifestIndex(ctx, mi.ID, imageIndex, r)
			if err != nil {
				return fmt.Errorf("failed to map manifest index: %w", err)
			}
			return nil
		},
	)

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to create image index in database")
	}
	return err
}

func (l *manifestService) dbPutBuildkitIndex(
	ctx context.Context,
	ml *manifestlist.DeserializedManifestList,
	payload []byte,
	digest digest.Digest,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	// convert to OCI manifest and process as if it was one
	m, err := OCIManifestFromBuildkitIndex(ml)
	if err != nil {
		return fmt.Errorf("converting buildkit index to manifest: %w", err)
	}

	// Note that `payload` is not the deserialized manifest (`m`) payload but
	// rather the index payload, untouched.
	// Within dbPutManifestOCIOrSchema2 we use this value for the
	// `manifests.payload` column and source the value for
	// the `manifests.digest` column from `imh.Digest`, and not from `m`.
	// Therefore, we keep behavioral consistency for
	// the outside world by preserving the index payload and digest while
	//  storing things internally as an OCI manifest.
	return l.dbPutManifestV2(ctx, m, payload, true, digest, headers, info)
}

func (l *manifestService) dbFindManifestListManifest(
	ctx context.Context,
	repository *types.Registry, imageName string, digest digest.Digest,
) (*types.Manifest, error) {
	dgst, err := types.NewDigest(digest)
	if err != nil {
		return nil, err
	}
	dbManifest, err := l.manifestDao.FindManifestByDigest(
		ctx, repository.ID,
		imageName, dgst,
	)
	if err != nil {
		if errors.Is(err, gitnessstore.ErrResourceNotFound) {
			return nil, fmt.Errorf(
				"manifest %s not found for %s/%s: %w", digest.String(),
				repository.Name, imageName, err,
			)
		}
		return nil, err
	}

	return dbManifest, nil
}

func (l *manifestService) layerMediaTypeExists(ctx context.Context, mt string) bool {
	exists, err := l.mtRepository.MediaTypeExists(ctx, mt)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msgf("error checking for existence of media type: %v", err)
		return false
	}

	if exists {
		return true
	}

	log.Ctx(ctx).Warn().Msgf("unknown layer media type")

	return false
}

func (l *manifestService) DeleteTag(
	ctx context.Context,
	repoKey string,
	tag string,
	info pkg.RegistryInfo,
) (bool, error) {
	// Fetch the registry by parent ID and name
	registry, err := l.registryDao.GetByParentIDAndName(ctx, info.ParentID, repoKey)
	if err != nil {
		return false, err
	}

	existingDigest := l.getTagDigest(ctx, registry.ID, info.Image, tag)
	deleted, err := l.tagDao.DeleteTagByName(ctx, registry.ID, tag)
	if err != nil {
		return false, fmt.Errorf("failed to delete tag in database: %w", err)
	}
	if !deleted {
		return false, distribution.ErrTagUnknown{Tag: tag}
	}

	if existingDigest != "" {
		session, _ := request.AuthSessionFrom(ctx)
		payload := webhook.GetArtifactDeletedPayload(ctx, session.Principal.ID, registry.ID,
			registry.Name, tag, existingDigest.String(), info.RootIdentifier, info.PackageType, info.Image,
			l.urlProvider)
		l.artifactEventReporter.ArtifactDeleted(ctx, &payload)
	}

	return true, nil
}

func (l *manifestService) getTagDigest(
	ctx context.Context,
	registryID int64,
	imageName string,
	tag string,
) digest.Digest {
	existingTag, findTagErr := l.tagDao.FindTag(ctx, registryID, imageName, tag)
	if findTagErr == nil && existingTag != nil {
		existingTaggedManifest, getManifestErr := l.manifestDao.Get(ctx, existingTag.ManifestID)
		if getManifestErr == nil {
			return existingTaggedManifest.Digest
		}
	}
	return ""
}

func (l *manifestService) DeleteTagsByManifestID(
	ctx context.Context,
	repoKey string,
	manifestID int64,
	info pkg.RegistryInfo,
) (bool, error) {
	registry, err := l.registryDao.GetByParentIDAndName(ctx, info.ParentID, repoKey)

	if err != nil {
		return false, err
	}

	return l.tagDao.DeleteTagByManifestID(ctx, registry.ID, manifestID)
}

func (l *manifestService) DeleteManifest(
	ctx context.Context,
	repoKey string,
	d digest.Digest,
	info pkg.RegistryInfo,
) error {
	log.Ctx(ctx).Debug().Msg("deleting manifest from repository in database")

	registry, err := l.registryDao.GetByParentIDAndName(ctx, info.ParentID, repoKey)
	imageName := info.Image

	if registry == nil || err != nil {
		return fmt.Errorf("repository not found in database: %w", err)
	}

	// We need to find the manifest first and then lookup for any manifest
	// it references (if it's a manifest list). This
	// is needed to ensure we lock any related online GC tasks to prevent
	// race conditions around the delete.
	newDigest, err := types.NewDigest(d)
	if err != nil {
		return err
	}
	m, err := l.manifestDao.FindManifestByDigest(ctx, registry.ID, imageName, newDigest)
	if err != nil {
		if errors.Is(err, gitnessstore.ErrResourceNotFound) {
			return util.ErrManifestNotFound
		}
		return err
	}

	return l.tx.WithTx(
		ctx, func(ctx context.Context) error {
			switch m.MediaType {
			case manifestlist.MediaTypeManifestList, v1.MediaTypeImageIndex:
				mm, err := l.manifestDao.References(ctx, m)
				if err != nil {
					return err
				}

				// This should never happen, as it's not possible to delete a
				//  child manifest if it's referenced by a list, which
				// means that we'll always have at least one child manifest here.
				//  Nevertheless, log error if this ever happens.
				if len(mm) == 0 {
					log.Ctx(ctx).Error().Stack().Err(err).Msgf("stored manifest list has no references")
					break
				}
				ids := make([]int64, 0, len(mm))
				for _, m := range mm {
					ids = append(ids, m.ID)
				}

				// Prevent long running transactions by setting an upper limit of
				//  manifestDeleteGCLockTimeout. If the GC is
				// holding the lock of a related review record, the processing
				// there should be fast enough to avoid this.
				// Regardless, we should not let transactions open (and clients waiting)
				//  for too long. If this sensible timeout
				// is exceeded, abort the manifest delete and let the client retry.
				//  This will bubble up and lead to a 503
				// Service Unavailable response.
				ctx, cancel := context.WithTimeout(ctx, manifestDeleteGCLockTimeout)
				defer cancel()

				if _, err := l.gcService.ManifestFindAndLockNBefore(
					ctx, registry.ID,
					ids, time.Now().Add(manifestDeleteGCReviewWindow),
				); err != nil {
					return err
				}
			}

			found, err := l.manifestDao.DeleteManifest(ctx, registry.ID, imageName, d)
			if err != nil {
				return err
			}
			if !found {
				return util.ErrManifestNotFound
			}

			return nil
		},
	)
}
