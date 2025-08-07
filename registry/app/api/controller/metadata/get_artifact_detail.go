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
	"encoding/json"
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/types/enum"
)

func (c *APIController) GetArtifactDetails(
	ctx context.Context,
	r artifact.GetArtifactDetailsRequestObject,
) (artifact.GetArtifactDetailsResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.GetArtifactDetails400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetArtifactDetails400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space,
		regInfo.RegistryIdentifier, enum.PermissionRegistryView)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return artifact.GetArtifactDetails403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	image := string(r.Artifact)
	version := string(r.Version)

	registry, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.ParentID, regInfo.RegistryIdentifier)

	if err != nil {
		return artifact.GetArtifactDetails500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	var artifactType *artifact.ArtifactType
	if r.Params.ArtifactType != nil {
		artifactType, err = ValidateAndGetArtifactType(registry.PackageType, string(*r.Params.ArtifactType))
		if err != nil {
			return artifact.GetArtifactDetails400JSONResponse{
				BadRequestJSONResponse: artifact.BadRequestJSONResponse(
					*GetErrorResponse(http.StatusBadRequest, err.Error()),
				),
			}, nil
		}
	}
	img, err := c.ImageStore.GetByNameAndType(ctx, regInfo.RegistryID, image, artifactType)

	if err != nil {
		return artifact.GetArtifactDetails500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	art, err := c.ArtifactStore.GetByName(ctx, img.ID, version)

	if err != nil {
		return artifact.GetArtifactDetails500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	downloadCount, err := c.DownloadStatRepository.GetTotalDownloadsForArtifactID(ctx, art.ID)

	if err != nil {
		return artifact.GetArtifactDetails500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	var artifactDetails artifact.ArtifactDetail

	// FIXME: Arvind: Unify the metadata structure to avoid this type checking
	//nolint:exhaustive
	switch registry.PackageType {
	case artifact.PackageTypeMAVEN:
		var metadata metadata.MavenMetadata
		err := json.Unmarshal(art.Metadata, &metadata)
		if err != nil {
			return artifact.GetArtifactDetails500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifactDetails = GetMavenArtifactDetail(img, art, metadata)
	case artifact.PackageTypeGENERIC:
		var metadata metadata.GenericMetadata
		err := json.Unmarshal(art.Metadata, &metadata)
		if err != nil {
			return artifact.GetArtifactDetails500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifactDetails = GetGenericArtifactDetail(img, art, metadata)
	case artifact.PackageTypePYTHON:
		var result map[string]interface{}
		err := json.Unmarshal(art.Metadata, &result)
		if err != nil {
			return artifact.GetArtifactDetails500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifactDetails = GetPythonArtifactDetail(img, art, result)

	case artifact.PackageTypeNPM:
		var result map[string]interface{}
		err := json.Unmarshal(art.Metadata, &result)
		if err != nil {
			return artifact.GetArtifactDetails500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifactDetails = GetNPMArtifactDetail(img, art, result, downloadCount)
	case artifact.PackageTypeRPM:
		var result map[string]interface{}
		err := json.Unmarshal(art.Metadata, &result)
		if err != nil {
			return artifact.GetArtifactDetails500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifactDetails = GetRPMArtifactDetail(img, art, result, downloadCount)
	case artifact.PackageTypeNUGET:
		var result map[string]interface{}
		err := json.Unmarshal(art.Metadata, &result)
		if err != nil {
			return artifact.GetArtifactDetails500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifactDetails = GetNugetArtifactDetail(img, art, result, downloadCount)
	case artifact.PackageTypeHUGGINGFACE:
		var result map[string]interface{}
		err := json.Unmarshal(art.Metadata, &result)
		if err != nil {
			return artifact.GetArtifactDetails500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifactDetails = GetHFArtifactDetail(img, art, result, downloadCount)
	case artifact.PackageTypeCARGO:
		var result map[string]interface{}
		err := json.Unmarshal(art.Metadata, &result)
		if err != nil {
			return artifact.GetArtifactDetails500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifactDetails = GetCargoArtifactDetail(img, art, result, downloadCount)
	case artifact.PackageTypeGO:
		var result map[string]interface{}
		err := json.Unmarshal(art.Metadata, &result)
		if err != nil {
			return artifact.GetArtifactDetails500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifactDetails = GetGoArtifactDetail(img, art, result, downloadCount)
	case artifact.PackageTypeDOCKER:
	case artifact.PackageTypeHELM:
	default:
		return artifact.GetArtifactDetails400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, "unsupported package type"),
			),
		}, nil
	}

	quarantinedArtifacts, err := c.QuarantineArtifactRepository.GetByFilePath(ctx, "", regInfo.RegistryID, image, version)
	if err != nil {
		return artifact.GetArtifactDetails500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	if len(quarantinedArtifacts) > 0 {
		isQuarantined := true
		artifactDetails.IsQuarantined = &isQuarantined
		artifactDetails.QuarantineReason = &quarantinedArtifacts[0].Reason
	} else {
		isQuarantined := false
		artifactDetails.IsQuarantined = &isQuarantined
	}
	return artifact.GetArtifactDetails200JSONResponse{
		ArtifactDetailResponseJSONResponse: artifact.ArtifactDetailResponseJSONResponse{
			Data:   artifactDetails,
			Status: "SUCCESS",
		},
	}, nil
}
