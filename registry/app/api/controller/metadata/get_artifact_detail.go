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
	"github.com/harness/gitness/registry/app/store/database"
	"github.com/harness/gitness/types/enum"
)

func (c *APIController) GetArtifactDetails(
	ctx context.Context,
	r artifact.GetArtifactDetailsRequestObject,
) (artifact.GetArtifactDetailsResponseObject, error) {
	regInfo, err := c.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
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
	permissionChecks := GetPermissionChecks(space, regInfo.RegistryIdentifier, enum.PermissionRegistryView)
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

	registry, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.parentID, regInfo.RegistryIdentifier)

	if err != nil {
		return artifact.GetArtifactDetails500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	img, err := c.ImageStore.GetByName(ctx, regInfo.RegistryID, image)

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

	var artifactDetails artifact.ArtifactDetail

	if artifact.PackageTypeMAVEN == registry.PackageType {
		var metadata database.MavenMetadata
		err := json.Unmarshal(art.Metadata, &metadata)
		if err != nil {
			return artifact.GetArtifactDetails500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifactDetails = GetMavenArtifactDetail(img, art, metadata)
	} else if artifact.PackageTypeGENERIC == registry.PackageType {
		var metadata database.GenericMetadata
		err := json.Unmarshal(art.Metadata, &metadata)
		if err != nil {
			return artifact.GetArtifactDetails500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifactDetails = GetGenericArtifactDetail(img, art, metadata)
	}
	return artifact.GetArtifactDetails200JSONResponse{
		ArtifactDetailResponseJSONResponse: artifact.ArtifactDetailResponseJSONResponse{
			Data:   artifactDetails,
			Status: "SUCCESS",
		},
	}, nil
}
