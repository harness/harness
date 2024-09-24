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

func (c *APIController) UpdateArtifactLabels(
	ctx context.Context,
	r artifact.UpdateArtifactLabelsRequestObject,
) (artifact.UpdateArtifactLabelsResponseObject, error) {
	regInfo, _ := c.GetRegistryRequestInfo(
		ctx, nil, nil, nil, nil,
		ArtifactVersionResource, "", string(r.RegistryRef), nil, nil, nil,
	)

	space, err := c.SpaceStore.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.UpdateArtifactLabels400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := GetPermissionChecks(space, regInfo.RegistryIdentifier, enum.PermissionRegistryEdit)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return artifact.UpdateArtifactLabels403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	a := string(r.Artifact)

	artifactEntity, err := c.ImageStore.GetByRepoAndName(ctx, regInfo.parentID, regInfo.RegistryIdentifier, a)

	if len(artifactEntity.Name) == 0 {
		return artifact.UpdateArtifactLabels404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "artifact doesn't exist with this name"),
			),
		}, nil
	}
	if err != nil {
		return throwModifyArtifact400Error(err), nil
	}
	existingArtifact, err := AttachLabels(artifact.ArtifactLabelRequest(*r.Body), artifactEntity)
	if err != nil {
		return throwModifyArtifact400Error(err), nil
	}

	err = c.ImageStore.Update(ctx, existingArtifact)

	if err != nil {
		return throwModifyArtifact400Error(err), nil
	}

	tag, err := c.TagStore.GetLatestTagMetadata(ctx, regInfo.parentID, regInfo.RegistryIdentifier, a)

	if err != nil {
		return artifact.UpdateArtifactLabels500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	return artifact.UpdateArtifactLabels200JSONResponse{
		ArtifactLabelResponseJSONResponse: *getArtifactSummary(*tag),
	}, nil
}

func throwModifyArtifact400Error(err error) artifact.UpdateArtifactLabels400JSONResponse {
	return artifact.UpdateArtifactLabels400JSONResponse{
		BadRequestJSONResponse: artifact.BadRequestJSONResponse(
			*GetErrorResponse(http.StatusBadRequest, err.Error()),
		),
	}
}

func AttachLabels(
	dto artifact.ArtifactLabelRequest,
	existingArtifact *types.Image,
) (*types.Image, error) {
	return &types.Image{
		ID:         existingArtifact.ID,
		RegistryID: existingArtifact.RegistryID,
		Name:       existingArtifact.Name,
		Labels:     dto.Labels,
		CreatedAt:  existingArtifact.CreatedAt,
	}, nil
}

func getArtifactSummary(t types.ArtifactMetadata) *artifact.ArtifactLabelResponseJSONResponse {
	downloads := int64(0)
	createdAt := GetTimeInMs(t.CreatedAt)
	modifiedAt := GetTimeInMs(t.ModifiedAt)
	artifactVersionSummary := &artifact.ArtifactSummary{
		CreatedAt:      &createdAt,
		ModifiedAt:     &modifiedAt,
		DownloadsCount: &downloads,
		ImageName:      t.Name,
		Labels:         &t.Labels,
		PackageType:    t.PackageType,
	}
	response := &artifact.ArtifactLabelResponseJSONResponse{
		Data:   *artifactVersionSummary,
		Status: artifact.StatusSUCCESS,
	}
	return response
}
