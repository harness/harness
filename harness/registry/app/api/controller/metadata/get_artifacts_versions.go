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

package metadata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	ml "github.com/harness/gitness/registry/app/manifest/manifestlist"
	os "github.com/harness/gitness/registry/app/manifest/ocischema"
	s2 "github.com/harness/gitness/registry/app/manifest/schema2"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) GetAllArtifactVersions(
	ctx context.Context,
	r artifact.GetAllArtifactVersionsRequestObject,
) (artifact.GetAllArtifactVersionsResponseObject, error) {
	registryRequestParams := &RegistryRequestParams{
		packageTypesParam: nil,
		page:              r.Params.Page,
		size:              r.Params.Size,
		search:            r.Params.SearchTerm,
		Resource:          ArtifactVersionResource,
		ParentRef:         "",
		RegRef:            string(r.RegistryRef),
		labelsParam:       nil,
		sortOrder:         r.Params.SortOrder,
		sortField:         r.Params.SortField,
		registryIDsParam:  nil,
	}
	regInfo, _ := c.GetRegistryRequestInfo(ctx, *registryRequestParams)

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetAllArtifactVersions400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space, regInfo.RegistryIdentifier,
		enum.PermissionRegistryView)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return artifact.GetAllArtifactVersions403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	image := string(r.Artifact)

	registry, err := c.RegistryRepository.Get(ctx, regInfo.RegistryID)
	if err != nil {
		if errors.Is(err, store.ErrResourceNotFound) {
			return artifact.GetAllArtifactVersions404JSONResponse{
				NotFoundJSONResponse: artifact.NotFoundJSONResponse(
					*GetErrorResponse(http.StatusNotFound, err.Error()),
				),
			}, nil
		}
		return throw500Error(err)
	}
	var artifactType *artifact.ArtifactType
	if r.Params.ArtifactType != nil {
		artifactType, err = ValidateAndGetArtifactType(registry.PackageType, string(*r.Params.ArtifactType))
		if err != nil {
			return artifact.GetAllArtifactVersions400JSONResponse{
				BadRequestJSONResponse: artifact.BadRequestJSONResponse(
					*GetErrorResponse(http.StatusBadRequest, err.Error()),
				),
			}, nil
		}
	}

	img, err := c.ImageStore.GetByNameAndType(ctx, registry.ID, image, artifactType)
	if err != nil {
		if errors.Is(err, store.ErrResourceNotFound) {
			return artifact.GetAllArtifactVersions404JSONResponse{
				NotFoundJSONResponse: artifact.NotFoundJSONResponse(
					*GetErrorResponse(http.StatusNotFound, err.Error()),
				),
			}, nil
		}
		return throw500Error(err)
	}

	//nolint:nestif
	if registry.PackageType == artifact.PackageTypeDOCKER || registry.PackageType == artifact.PackageTypeHELM {
		var ociVersions *[]types.OciVersionMetadata
		if c.UntaggedImagesEnabled(ctx) {
			ociVersions, err = c.TagStore.GetAllOciVersionsByRepoAndImage(
				ctx, regInfo.ParentID, regInfo.RegistryIdentifier,
				image, regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset, regInfo.searchTerm,
			)
		} else {
			ociVersions, err = c.TagStore.GetAllTagsByRepoAndImage(
				ctx, regInfo.ParentID, regInfo.RegistryIdentifier,
				image, regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset, regInfo.searchTerm,
			)
		}
		if err != nil {
			return throw500Error(err)
		}

		var digests []string
		for _, ociVersion := range *ociVersions {
			if ociVersion.Digest != "" {
				digests = append(digests, ociVersion.Digest)
			}
		}

		counts, err := c.DownloadStatRepository.GetTotalDownloadsForManifests(ctx, digests, img.ID)
		if err != nil {
			return throw500Error(err)
		}

		err = c.updateQuarantineInfo(ctx, ociVersions, image, registry.Name, registry.ParentID)
		if err != nil {
			return throw500Error(err)
		}

		for i, ociVersion := range *ociVersions {
			if ociVersion.Digest != "" {
				(*ociVersions)[i].DownloadCount = counts[ociVersion.Digest]
			}
		}
		var count int64
		if c.UntaggedImagesEnabled(ctx) {
			count, err = c.TagStore.CountOciVersionByRepoAndImage(
				ctx, regInfo.ParentID, regInfo.RegistryIdentifier,
				image, regInfo.searchTerm,
			)
			if err != nil {
				return throw500Error(err)
			}
		} else {
			count, err = c.TagStore.CountAllTagsByRepoAndImage(
				ctx, regInfo.ParentID, regInfo.RegistryIdentifier,
				image, regInfo.searchTerm,
			)
			if err != nil {
				return throw500Error(err)
			}
		}

		err = setDigestCount(ctx, *ociVersions)
		if err != nil {
			return throw500Error(err)
		}

		return artifact.GetAllArtifactVersions200JSONResponse{
			ListArtifactVersionResponseJSONResponse: *GetAllArtifactVersionResponse(
				ctx, ociVersions, image, count, regInfo.pageNumber, regInfo.limit,
				c.URLProvider.RegistryURL(ctx, regInfo.RootIdentifier, regInfo.RegistryIdentifier),
				c.SetupDetailsAuthHeaderPrefix, c.UntaggedImagesEnabled(ctx),
			),
		}, nil
	}
	metadata, err := c.ArtifactStore.GetAllVersionsByRepoAndImage(ctx, regInfo.RegistryID, image,
		regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset,
		regInfo.searchTerm, artifactType)
	if err != nil {
		return throw500Error(err)
	}

	cnt, _ := c.ArtifactStore.CountAllVersionsByRepoAndImage(ctx, regInfo.ParentID, regInfo.RegistryIdentifier, image,
		regInfo.searchTerm, artifactType)

	registryURL := c.URLProvider.RegistryURL(ctx, regInfo.RootIdentifier, regInfo.RegistryIdentifier)
	if registry.PackageType == artifact.PackageTypeGENERIC {
		registryURL = c.URLProvider.RegistryURL(ctx, regInfo.RootIdentifier,
			strings.ToLower(string(registry.PackageType)), regInfo.RegistryIdentifier)
	}

	return artifact.GetAllArtifactVersions200JSONResponse{
		ListArtifactVersionResponseJSONResponse: *GetNonOCIAllArtifactVersionResponse(
			ctx, metadata, image, cnt, regInfo.pageNumber, regInfo.limit, registryURL,
			c.SetupDetailsAuthHeaderPrefix, string(registry.PackageType), c.PackageWrapper,
		),
	}, nil
}

func (c *APIController) updateQuarantineInfo(
	ctx context.Context,
	versions *[]types.OciVersionMetadata,
	image string,
	regName string,
	parentID int64,
) error {
	if versions == nil || len(*versions) == 0 {
		return nil
	}

	// Collect all artifact identifiers for quarantine lookup
	artifactIdentifiers := make([]types.ArtifactIdentifier, 0, len(*versions))
	artifactIndexMap := make(map[types.ArtifactIdentifier]int)

	for i, version := range *versions {
		// Only process versions that have a digest
		if version.Digest != "" {
			artifactID := types.ArtifactIdentifier{
				Name:         image,
				Version:      version.Digest,
				RegistryName: regName,
			}
			artifactIdentifiers = append(artifactIdentifiers, artifactID)
			artifactIndexMap[artifactID] = i
		}
	}

	// Get quarantine information for all versions using the existing tag store method
	quarantineMap, err := c.TagStore.GetQuarantineInfoForArtifacts(ctx, artifactIdentifiers, parentID)
	if err != nil {
		return err
	}

	// Update versions with quarantine information
	for artifactID, quarantineInfo := range quarantineMap {
		if index, exists := artifactIndexMap[artifactID]; exists {
			(*versions)[index].IsQuarantined = true
			(*versions)[index].QuarantineReason = quarantineInfo.Reason
		}
	}
	return nil
}

func setDigestCount(ctx context.Context, tags []types.OciVersionMetadata) error {
	for i := range tags {
		err := setDigestCountInTagMetadata(ctx, &tags[i])
		if err != nil {
			return err
		}
	}
	return nil
}

//nolint:unused // kept for potential future use
func setDigestCountInTagMetadata(ctx context.Context, v *types.OciVersionMetadata) error {
	m := types.Manifest{
		SchemaVersion: v.SchemaVersion,
		MediaType:     v.MediaType,
		NonConformant: v.NonConformant,
		Payload:       v.Payload,
	}
	manifest, err := docker.DBManifestToManifest(&m)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msg("Failed to convert DBManifest to Manifest")
		return err
	}
	switch reqManifest := manifest.(type) {
	case *s2.DeserializedManifest, *os.DeserializedManifest:
		v.DigestCount = 1
	case *ml.DeserializedManifestList:
		v.DigestCount = len(reqManifest.Manifests)
	default:
		err = fmt.Errorf("unknown manifest type: %T", manifest)
		log.Ctx(ctx).Error().Stack().Err(err).Msg("Failed to set digest count")
	}
	return nil
}

//nolint:unused,unparam // kept for potential future use
func throw500Error(err error) (artifact.GetAllArtifactVersionsResponseObject, error) {
	wrappedErr := fmt.Errorf("internal server error: %w", err)
	return artifact.GetAllArtifactVersions500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, wrappedErr.Error()),
		),
	}, nil
}
