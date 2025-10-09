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
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types/enum"
)

func (c *APIController) GetArtifactSummary(
	ctx context.Context,
	r artifact.GetArtifactSummaryRequestObject,
) (artifact.GetArtifactSummaryResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.GetArtifactSummary400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetArtifactSummary400JSONResponse{
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
		return artifact.GetArtifactSummary403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	image := string(r.Artifact)
	registry, err := c.RegistryRepository.Get(ctx, regInfo.RegistryID)

	if err != nil {
		return artifact.GetArtifactSummary500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	var artifactType *artifact.ArtifactType
	if r.Params.ArtifactType != nil {
		artifactType, err = ValidateAndGetArtifactType(registry.PackageType, string(*r.Params.ArtifactType))
		if err != nil {
			return artifact.GetArtifactSummary400JSONResponse{
				BadRequestJSONResponse: artifact.BadRequestJSONResponse(
					*GetErrorResponse(http.StatusBadRequest, err.Error()),
				),
			}, nil
		}
	}
	metadata, err := c.getImageMetadata(ctx, registry, image, artifactType)
	if err != nil {
		return artifact.GetArtifactSummary500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	return artifact.GetArtifactSummary200JSONResponse{
		ArtifactSummaryResponseJSONResponse: *GetArtifactSummary(*metadata),
	}, nil
}

func (c *APIController) getImageMetadata(
	ctx context.Context, registry *types.Registry, image string,
	artifactType *artifact.ArtifactType,
) (*types.ImageMetadata, error) {
	img, err := c.ImageStore.GetByNameAndType(ctx, registry.ID, image, artifactType)
	if err != nil {
		return nil, err
	}
	downloadCount, err := c.DownloadStatRepository.GetTotalDownloadsForImage(ctx, img.ID)
	if err != nil {
		return nil, err
	}
	imgMetadata := &types.ImageMetadata{
		Name:          image,
		DownloadCount: downloadCount,
		RepoName:      registry.Name,
		PackageType:   registry.PackageType,
		CreatedAt:     img.CreatedAt,
		ArtifactType:  img.ArtifactType,
	}
	//nolint:nestif
	if registry.PackageType == artifact.PackageTypeDOCKER || registry.PackageType == artifact.PackageTypeHELM {
		if c.UntaggedImagesEnabled(ctx) {
			latestManifest, err := c.ManifestStore.GetLatestManifest(ctx, registry.ID, image)
			if err != nil {
				return nil, err
			}
			imgMetadata.LatestVersion = latestManifest.Digest.String()
			imgMetadata.ModifiedAt = latestManifest.CreatedAt
		} else {
			latestTag, err := c.TagStore.GetLatestTag(ctx, registry.ID, image)
			if err != nil {
				return nil, err
			}
			imgMetadata.LatestVersion = latestTag.Name
			imgMetadata.ModifiedAt = latestTag.UpdatedAt
		}
	} else {
		latestArtifact, err := c.ArtifactStore.GetLatestByImageID(ctx, img.ID)
		if err != nil {
			return nil, err
		}
		imgMetadata.LatestVersion = latestArtifact.Version
		imgMetadata.ModifiedAt = latestArtifact.UpdatedAt
	}
	return imgMetadata, nil
}
