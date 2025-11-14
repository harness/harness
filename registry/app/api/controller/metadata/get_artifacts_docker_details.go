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
	"github.com/rs/zerolog/log"
)

func (c *APIController) GetDockerArtifactDetails(
	ctx context.Context,
	r artifact.GetDockerArtifactDetailsRequestObject,
) (artifact.GetDockerArtifactDetailsResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
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
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space,
		regInfo.RegistryIdentifier, enum.PermissionRegistryView)
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

	registry, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.ParentID, regInfo.RegistryIdentifier)

	if err != nil {
		return artifact.GetDockerArtifactDetails500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	dgst, err := types.NewDigest(digest.Digest(manifestDigest))
	if err != nil {
		return getArtifactDetailsErrResponse(ctx, err)
	}
	art, err := c.ArtifactStore.GetArtifactMetadata(ctx, registry.ParentID, registry.Name, image, dgst.String(),
		nil)
	if err != nil {
		return getArtifactDetailsErrResponse(ctx, err)
	}

	m, err := c.ManifestStore.FindManifestByDigest(ctx, registry.ID, image, dgst)

	if err != nil {
		if errors.Is(err, store2.ErrResourceNotFound) {
			return getArtifactDetailsErrResponse(ctx, fmt.Errorf("manifest not found"))
		}
		return getArtifactDetailsErrResponse(ctx, err)
	}

	quarantineArtifacts, err := c.QuarantineArtifactRepository.GetByFilePath(ctx, "",
		regInfo.RegistryID, image, dgst.String(), nil)
	if err != nil {
		return getArtifactDetailsErrResponse(ctx, err)
	}
	var isQuarantined bool
	var quarantineReason *string
	if len(quarantineArtifacts) > 0 {
		isQuarantined = true
		quarantineReason = &quarantineArtifacts[0].Reason
	}

	//nolint:nestif
	if c.UntaggedImagesEnabled(ctx) && !isDockerVersionTag(r) {
		dockerArtifactDetails := GetDockerArtifactDetails(
			registry, m.ImageName, m.Digest.String(), m.CreatedAt, m.CreatedAt, m.Digest.String(), m.TotalSize,
			c.URLProvider.RegistryURL(ctx, regInfo.RootIdentifier, registry.Name),
			art.DownloadCount, isQuarantined, quarantineReason,
		)
		pullCommandByDigest := GetDockerPullCommand(m.ImageName, m.Digest.String(),
			c.URLProvider.RegistryURL(ctx, regInfo.RootIdentifier, registry.Name), false,
		)
		dockerArtifactDetails.Data.PullCommandByDigest = &pullCommandByDigest
		tags, err := c.TagStore.GetTagsByManifestID(ctx, m.ID)
		if err != nil {
			return getArtifactDetailsErrResponse(ctx, err)
		}
		if tags != nil {
			dockerArtifactDetails.Data.Metadata = &artifact.ArtifactEntityMetadata{
				"tags": tags,
			}
		}
		return artifact.GetDockerArtifactDetails200JSONResponse{
			DockerArtifactDetailResponseJSONResponse: *dockerArtifactDetails,
		}, nil
	}
	tag, err := c.TagStore.GetTagDetail(ctx, registry.ID, image, version)
	if err != nil {
		return getArtifactDetailsErrResponse(ctx, err)
	}
	dockerArtifactDetails := GetDockerArtifactDetails(
		registry, tag.ImageName, tag.Name, tag.CreatedAt, tag.UpdatedAt, m.Digest.String(), m.TotalSize,
		c.URLProvider.RegistryURL(ctx, regInfo.RootIdentifier, registry.Name),
		art.DownloadCount, isQuarantined, quarantineReason,
	)
	pullCommandByTag := GetDockerPullCommand(tag.ImageName, tag.Name, c.URLProvider.RegistryURL(
		ctx, regInfo.RootIdentifier, registry.Name), true,
	)
	dockerArtifactDetails.Data.PullCommand = &pullCommandByTag

	if c.UntaggedImagesEnabled(ctx) {
		pullCommandByDigest := GetDockerPullCommand(m.ImageName, m.Digest.String(),
			c.URLProvider.RegistryURL(ctx, regInfo.RootIdentifier, registry.Name),
			false,
		)
		dockerArtifactDetails.Data.PullCommandByDigest = &pullCommandByDigest
	}
	return artifact.GetDockerArtifactDetails200JSONResponse{
		DockerArtifactDetailResponseJSONResponse: *dockerArtifactDetails,
	}, nil
}

func getArtifactDetailsErrResponse(
	ctx context.Context,
	err error,
) (artifact.GetDockerArtifactDetailsResponseObject, error) {
	log.Error().Ctx(ctx).Msgf("error while getting artifact details: %v", err)
	return artifact.GetDockerArtifactDetails500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, nil
}

func isDockerVersionTag(r artifact.GetDockerArtifactDetailsRequestObject) bool {
	return r.Params.VersionType != nil && *r.Params.VersionType == artifact.GetDockerArtifactDetailsParamsVersionTypeTAG
}
