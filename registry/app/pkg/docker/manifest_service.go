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

	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/manifest/ocischema"
	"github.com/harness/gitness/registry/app/manifest/schema2"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/gc"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/registry/api/errcode"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog/log"
)

type manifestService struct {
	registryDao    store.RegistryRepository
	manifestDao    store.ManifestRepository
	layerDao       store.LayerRepository
	blobRepo       store.BlobRepository
	mtRepository   store.MediaTypesRepository
	tagDao         store.TagRepository
	imageDao       store.ImageRepository
	artifactDao    store.ArtifactRepository
	manifestRefDao store.ManifestReferenceRepository
	gcService      gc.Service
	tx             dbtx.Transactor
}

func NewManifestService(
	registryDao store.RegistryRepository, manifestDao store.ManifestRepository,
	blobRepo store.BlobRepository, mtRepository store.MediaTypesRepository, tagDao store.TagRepository,
	imageDao store.ImageRepository, artifactDao store.ArtifactRepository,
	layerDao store.LayerRepository, manifestRefDao store.ManifestReferenceRepository,
	tx dbtx.Transactor, gcService gc.Service,
) ManifestService {
	return &manifestService{
		registryDao:    registryDao,
		manifestDao:    manifestDao,
		layerDao:       layerDao,
		blobRepo:       blobRepo,
		mtRepository:   mtRepository,
		tagDao:         tagDao,
		artifactDao:    artifactDao,
		imageDao:       imageDao,
		manifestRefDao: manifestRefDao,
		gcService:      gcService,
		tx:             tx,
	}
}

type ManifestService interface {
	// GetTags gets the tags of a repository
	DBTag(
		ctx context.Context,
		mfst manifest.Manifest,
		d digest.Digest,
		tag string,
		repoKey string,
		headers *commons.ResponseHeaders,
		info pkg.RegistryInfo,
	) error
	DBPut(
		ctx context.Context,
		mfst manifest.Manifest,
		d digest.Digest,
		repoKey string,
		headers *commons.ResponseHeaders,
		info pkg.RegistryInfo,
	) error
	DeleteTag(ctx context.Context, repoKey string, tag string, info pkg.RegistryInfo) (bool, error)
	DeleteManifest(ctx context.Context, repoKey string, d digest.Digest, info pkg.RegistryInfo) error
	DBFindRepositoryBlob(
		ctx context.Context, desc manifest.Descriptor, repoID int64,
		info pkg.RegistryInfo,
	) (*types.Blob, error)
}

func (l *manifestService) DBTag(
	ctx context.Context,
	mfst manifest.Manifest,
	d digest.Digest,
	tag string,
	repoKey string,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	imageName := info.Image

	if err := l.dbTagManifest(ctx, d, tag, imageName, info); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create tag in database")
		err2 := l.handleTagError(ctx, mfst, d, tag, repoKey, headers, info, err, imageName)
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
	repoKey string,
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
		if err = l.DBPut(ctx, mfst, "", repoKey, headers, info); err != nil {
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
	dbRepo, err := l.registryDao.GetByParentIDAndName(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return err
	}
	newDigest, err := types.NewDigest(dgst)
	if err != nil {
		return err
	}
	dbManifest, err := l.manifestDao.FindManifestByDigest(ctx, dbRepo.ID, info.Image, newDigest)
	if err != nil {
		if errors.Is(err, store2.ErrResourceNotFound) {
			return fmt.Errorf("manifest %s not found in database", dgst)
		}
		return err
	}

	// We need to find and lock a GC manifest task that is related with the manifest that we're about to tag. This
	// is needed to ensure we lock any related online GC tasks to prevent race conditions around the tag creation. See:

	return l.tx.WithTx(
		ctx, func(ctx context.Context) error {
			// Prevent long running transactions by setting an upper limit of manifestTagGCLockTimeout. If the GC is holding
			// the lock of a related review record, the processing there should be fast enough to avoid this. Regardless, we
			// should not let transactions open (and clients waiting) for too long. If this sensible timeout is exceeded, abort
			// the tag creation and let the client retry. This will bubble up and lead to a 503 Service Unavailable response.
			ctx, cancel := context.WithTimeout(ctx, manifestTagGCLockTimeout)
			defer cancel()

			if _, err := l.gcService.ManifestFindAndLockBefore(
				ctx, dbRepo.ID, dbManifest.ID,
				time.Now().Add(manifestTagGCReviewWindow),
			); err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			image := &types.Image{
				Name:       imageName,
				RegistryID: dbRepo.ID,
				Enabled:    true,
			}

			if err := l.imageDao.CreateOrUpdate(ctx, image); err != nil {
				return err
			}

			digest, err := types.NewDigest(dgst)
			if err != nil {
				return err
			}
			artifact := &types.Artifact{
				ImageID: image.ID,
				Version: digest.String(),
			}

			if err := l.artifactDao.CreateOrUpdate(ctx, artifact); err != nil {
				return err
			}

			tag := &types.Tag{
				Name:       tagName,
				ImageName:  imageName,
				RegistryID: dbRepo.ID,
				ManifestID: dbManifest.ID,
			}

			if err := l.tagDao.CreateOrUpdate(ctx, tag); err != nil {
				return err
			}

			return nil
		},
	)
}

func (l *manifestService) DBPut(
	ctx context.Context,
	mfst manifest.Manifest,
	d digest.Digest,
	repoKey string,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	_, payload, err := mfst.Payload()
	if err != nil {
		return err
	}

	err = l.dbPutManifest(ctx, mfst, payload, d, repoKey, headers, info)
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
	repoKey string,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	switch reqManifest := manifest.(type) {
	case *schema2.DeserializedManifest:
		return l.dbPutManifestSchema2(ctx, reqManifest, payload, d, repoKey, headers, info)
	case *ocischema.DeserializedManifest:
		return l.dbPutManifestOCI(ctx, reqManifest, payload, d, repoKey, headers, info)
	case *manifestlist.DeserializedManifestList:
		return l.dbPutManifestList(ctx, reqManifest, payload, d, repoKey, headers, info)
	case *ocischema.DeserializedImageIndex:
		return l.dbPutImageIndex(ctx, reqManifest, payload, d, repoKey, headers, info)
	default:
		return errcode.ErrorCodeManifestInvalid.WithDetail("manifest type unsupported")
	}
}

func (l *manifestService) dbPutManifestSchema2(
	ctx context.Context,
	manifest *schema2.DeserializedManifest,
	payload []byte,
	d digest.Digest,
	repoKey string,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	return l.dbPutManifestV2(ctx, manifest, payload, false, d, repoKey, headers, info)
}

func (l *manifestService) dbPutManifestV2(
	ctx context.Context,
	mfst manifest.ManifestV2,
	payload []byte,
	nonConformant bool,
	digest digest.Digest,
	repoKey string,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	// find target repository
	dbRepo, err := l.registryDao.GetByParentIDAndName(ctx, info.ParentID, repoKey)
	if err != nil {
		return err
	}
	if dbRepo == nil {
		return errors.New("repository not found in database")
	}

	// Find the config now to ensure that the config's blob is associated with the repository.
	dbCfgBlob, err := l.DBFindRepositoryBlob(ctx, mfst.Config(), dbRepo.ID, info)
	if err != nil {
		return err
	}

	dgst, err := types.NewDigest(digest)
	if err != nil {
		return err
	}

	dbManifest, err := l.manifestDao.FindManifestByDigest(ctx, dbRepo.ID, info.Image, dgst)
	if err != nil && !errors.Is(err, store2.ErrResourceNotFound) {
		return err
	}

	if dbManifest != nil {
		return nil
	}

	log.Debug().Msgf("manifest not found in database")

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
			ocim.Annotations(), dbRepo, m, headers, info,
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
		dbBlob, err := l.DBFindRepositoryBlob(ctx, reqLayer, dbRepo.ID, info)
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
	repoID int64, info pkg.RegistryInfo,
) (*types.Blob, error) {
	image := info.Image
	b, err := l.blobRepo.FindByDigestAndRepoID(ctx, desc.Digest, repoID, image)
	if err != nil {
		if errors.Is(err, store2.ErrResourceNotFound) {
			return nil, fmt.Errorf("blob not found in database")
		}
		return nil, err
	}
	return b, nil
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
		if err != nil && !errors.Is(err, store2.ErrResourceNotFound) {
			return err
		}

		if errors.Is(err, store2.ErrResourceNotFound) {
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
	repoKey string,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	return l.dbPutManifestV2(ctx, manifest, payload, false, d, repoKey, headers, info)
}

func (l *manifestService) dbPutManifestList(
	ctx context.Context,
	manifestList *manifestlist.DeserializedManifestList,
	payload []byte,
	digest digest.Digest,
	repoKey string,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	if LikelyBuildxCache(manifestList) {
		return l.dbPutBuildkitIndex(ctx, manifestList, payload, digest, repoKey, headers, info)
	}

	r, err := l.registryDao.GetByParentIDAndName(ctx, info.ParentID, repoKey)
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

	ml, err := l.manifestDao.FindManifestByDigest(ctx, r.ID, info.Image, dgst)
	if err != nil && !errors.Is(err, store2.ErrResourceNotFound) {
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

	mm := make([]*types.Manifest, 0, len(manifestList.Manifests))
	ids := make([]int64, 0, len(mm))
	for _, desc := range manifestList.Manifests {
		m, err := l.dbFindManifestListManifest(ctx, r, info.Image, desc.Digest)
		if err != nil {
			return err
		}
		mm = append(mm, m)
		ids = append(ids, m.ID)
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
			return nil
		},
	)

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to create manifest list in database")
	}
	return err
}

func (l *manifestService) dbPutImageIndex(
	ctx context.Context,
	imageIndex *ocischema.DeserializedImageIndex,
	payload []byte,
	digest digest.Digest,
	repoKey string,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	r, err := l.registryDao.GetByParentIDAndName(ctx, info.ParentID, repoKey)
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
	if err != nil && !errors.Is(err, store2.ErrResourceNotFound) {
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

	mm := make([]*types.Manifest, 0, len(imageIndex.Manifests))
	ids := make([]int64, 0, len(mm))
	for _, desc := range imageIndex.Manifests {
		m, err := l.dbFindManifestListManifest(ctx, r, info.Image, desc.Digest)
		if err != nil {
			return err
		}
		mm = append(mm, m)
		ids = append(ids, m.ID)
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
	repoKey string,
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
	return l.dbPutManifestV2(ctx, m, payload, true, digest, repoKey, headers, info)
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
		if errors.Is(err, store2.ErrResourceNotFound) {
			return nil, fmt.Errorf(
				"manifest %s not found for %s/%s", digest.String(),
				repository.Name, imageName,
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

	found, err := l.tagDao.DeleteTagByName(ctx, registry.ID, tag)
	if err != nil {
		return false, fmt.Errorf("failed to delete tag in database: %w", err)
	}
	if !found {
		return false, distribution.ErrTagUnknown{Tag: tag}
	}

	return true, nil
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
		if errors.Is(err, store2.ErrResourceNotFound) {
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
