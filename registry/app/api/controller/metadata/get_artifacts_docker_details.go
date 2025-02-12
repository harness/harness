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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"

	"github.com/opencontainers/go-digest"
)

func (c *APIController) GetDockerArtifactDetails(
	ctx context.Context,
	r artifact.GetDockerArtifactDetailsRequestObject,
) (artifact.GetDockerArtifactDetailsResponseObject, error) {
	regInfo, err := c.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.GetDockerArtifactDetails400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetDockerArtifactDetails400JSONResponse{
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
		return artifact.GetDockerArtifactDetails403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	image := string(r.Artifact)
	version := string(r.Version)
	manifestDigest := string(r.Params.Digest)

	registry, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.parentID, regInfo.RegistryIdentifier)

	if err != nil {
		return artifact.GetDockerArtifactDetails500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	tag, err := c.TagStore.GetTagDetail(ctx, registry.ID, image, version)
	if err != nil {
		return getArtifactDetailsErrResponse(err)
	}
	dgst, err := types.NewDigest(digest.Digest(manifestDigest))
	if err != nil {
		return getArtifactDetailsErrResponse(err)
	}
	m, err := c.ManifestStore.FindManifestByDigest(ctx, registry.ID, image, dgst)

	if err != nil {
		if errors.Is(err, store2.ErrResourceNotFound) {
			return getArtifactDetailsErrResponse(fmt.Errorf("manifest not found"))
		}
		return getArtifactDetailsErrResponse(err)
	}

	latestTag, err := c.TagStore.GetLatestTag(ctx, registry.ID, image)
	if err != nil {
		return getArtifactDetailsErrResponse(err)
	}

	return artifact.GetDockerArtifactDetails200JSONResponse{
		DockerArtifactDetailResponseJSONResponse: *GetDockerArtifactDetails(
			registry, tag, m,
			latestTag.ID == tag.ID, c.URLProvider.RegistryURL(ctx, regInfo.RootIdentifier, registry.Name),
		),
	}, nil
}

func getArtifactDetailsErrResponse(err error) (artifact.GetDockerArtifactDetailsResponseObject, error) {
	return artifact.GetDockerArtifactDetails500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, nil
}
