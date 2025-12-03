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
	"fmt"
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) GetOciArtifactTags(
	ctx context.Context,
	r artifact.GetOciArtifactTagsRequestObject,
) (artifact.GetOciArtifactTagsResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return c.getOciArtifactTagsBadRequestResponse(
			fmt.Errorf("failed to get registry request base info: %w", err)), nil
	}
	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return c.getOciArtifactTagsBadRequestResponse(err), nil
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
		return artifact.GetOciArtifactTags403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	image := string(r.Artifact)
	_, err = c.ImageStore.GetByName(ctx, regInfo.RegistryID, image)
	if err != nil {
		return getOCIArtifacts404Error(ctx, fmt.Errorf("failed to get image by name: [%s], error: %w", image, err))
	}

	offset := GetOffset(r.Params.Size, r.Params.Page)
	limit := GetPageLimit(r.Params.Size)
	page := GetPageNumber(r.Params.Page)

	searchTerm := ""
	if r.Params.SearchTerm != nil {
		searchTerm = string(*r.Params.SearchTerm)
	}

	tagsInfo, err := c.TagStore.GetOciTagsInfo(ctx, regInfo.RegistryID, image, limit, offset, searchTerm)
	if err != nil {
		return getOCIArtifacts500Error(ctx, err)
	}

	var count int64
	count, err = c.TagStore.CountAllTagsByRepoAndImage(
		ctx, regInfo.ParentID, regInfo.RegistryIdentifier,
		image, searchTerm,
	)
	if err != nil {
		return getOCIArtifacts500Error(ctx, err)
	}

	pageCount := GetPageCount(count, limit)
	var tags []artifact.OciArtifactTag
	for _, t := range *tagsInfo {
		tags = append(tags, artifact.OciArtifactTag{
			Name:   t.Name,
			Digest: t.Digest,
		})
	}
	return artifact.GetOciArtifactTags200JSONResponse{
		ListOciArtifactTagsResponseJSONResponse: artifact.ListOciArtifactTagsResponseJSONResponse{
			Data: artifact.ListOciArtifactTags{
				ItemCount:       &count,
				PageCount:       &pageCount,
				PageIndex:       &page,
				PageSize:        &limit,
				OciArtifactTags: tags,
			},
			Status: artifact.StatusSUCCESS,
		},
	}, nil
}

func (c *APIController) getOciArtifactTagsBadRequestResponse(err error) artifact.GetOciArtifactTagsResponseObject {
	return artifact.GetOciArtifactTags400JSONResponse{
		BadRequestJSONResponse: artifact.BadRequestJSONResponse(
			*GetErrorResponse(http.StatusBadRequest, err.Error()),
		),
	}
}

func getOCIArtifacts500Error(ctx context.Context, err error) (artifact.GetOciArtifactTagsResponseObject, error) {
	wrappedErr := fmt.Errorf("internal server error: %w", err)
	log.Error().Ctx(ctx).Msgf("error while getting artifact tags details: %v", wrappedErr)
	return artifact.GetOciArtifactTags500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, wrappedErr.Error()),
		),
	}, nil
}

func getOCIArtifacts404Error(ctx context.Context, err error) (artifact.GetOciArtifactTagsResponseObject, error) {
	log.Error().Ctx(ctx).Msgf("error while getting artifact tags details: %v", err)
	return artifact.GetOciArtifactTags404JSONResponse{
		NotFoundJSONResponse: artifact.NotFoundJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, nil
}
