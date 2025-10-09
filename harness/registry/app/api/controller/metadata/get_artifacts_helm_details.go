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

func (c *APIController) GetHelmArtifactDetails(
	ctx context.Context,
	r artifact.GetHelmArtifactDetailsRequestObject,
) (artifact.GetHelmArtifactDetailsResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.GetHelmArtifactDetails400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}
	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetHelmArtifactDetails400JSONResponse{
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
		return artifact.GetHelmArtifactDetails403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	image := string(r.Artifact)
	version := string(r.Version)

	registry, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.ParentID, regInfo.RegistryIdentifier)

	if err != nil {
		return artifact.GetHelmArtifactDetails500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	var m *types.Manifest
	var artifactDetails *artifact.HelmArtifactDetailResponseJSONResponse
	//nolint:nestif
	if c.UntaggedImagesEnabled(ctx) && !isHelmVersionTag(r) {
		dgst, err := types.NewDigest(digest.Digest(version))
		if err != nil {
			return getHelmArtifactDetailsErrResponse(err)
		}
		m, err = c.ManifestStore.FindManifestByDigest(ctx, registry.ID, image, dgst)
		if err != nil {
			if errors.Is(err, store2.ErrResourceNotFound) {
				return getHelmArtifactDetailsErrResponse(fmt.Errorf("manifest not found"))
			}
			return getHelmArtifactDetailsErrResponse(err)
		}
		registryURL := c.URLProvider.RegistryURL(ctx, regInfo.RootIdentifier, regInfo.RegistryIdentifier)
		artifactDetails = GetHelmArtifactDetails(
			registry, m.ImageName, m.Digest.String(), m.CreatedAt, m.CreatedAt, m, registryURL)
		pullCommandByDigest := GetHelmPullCommand(m.ImageName, m.Digest.String(), registryURL, false)
		artifactDetails.Data.PullCommandByDigest = &pullCommandByDigest
		tags, err := c.TagStore.GetTagsByManifestID(ctx, m.ID)
		if err != nil {
			return getHelmArtifactDetailsErrResponse(err)
		}
		if tags != nil {
			artifactDetails.Data.Metadata = &artifact.ArtifactEntityMetadata{
				"tags": tags,
			}
		}
	} else {
		tag, err := c.TagStore.GetTagDetail(ctx, registry.ID, image, version)
		if err != nil {
			return getHelmArtifactDetailsErrResponse(err)
		}
		m, err = c.ManifestStore.FindManifestByTagName(ctx, registry.ID, image, version)
		if err != nil {
			if errors.Is(err, store2.ErrResourceNotFound) {
				return getHelmArtifactDetailsErrResponse(fmt.Errorf("manifest not found"))
			}
			return getHelmArtifactDetailsErrResponse(err)
		}
		registryURL := c.URLProvider.RegistryURL(ctx, regInfo.RootIdentifier, regInfo.RegistryIdentifier)
		artifactDetails = GetHelmArtifactDetails(
			registry, m.ImageName, m.Digest.String(), m.CreatedAt, m.CreatedAt, m, registryURL)
		pullCommand := GetHelmPullCommand(m.ImageName, tag.Name, registryURL, true)
		artifactDetails.Data.PullCommand = &pullCommand
		if c.UntaggedImagesEnabled(ctx) {
			pullCommandByDigest := GetHelmPullCommand(m.ImageName, m.Digest.String(), registryURL, false)
			artifactDetails.Data.PullCommandByDigest = &pullCommandByDigest
		}
	}

	parsedDigest, err := types.NewDigest(m.Digest)
	if err != nil {
		return getHelmArtifactDetailsErrResponse(err)
	}

	art, err := c.ArtifactStore.GetArtifactMetadata(ctx, registry.ParentID, registry.Name, image, parsedDigest.String(),
		nil)
	if err != nil {
		return getHelmArtifactDetailsErrResponse(err)
	}
	artifactDetails.Data.DownloadsCount = &art.DownloadCount

	return artifact.GetHelmArtifactDetails200JSONResponse{
		HelmArtifactDetailResponseJSONResponse: *artifactDetails,
	}, nil
}

func getHelmArtifactDetailsErrResponse(err error) (artifact.GetHelmArtifactDetailsResponseObject, error) {
	return artifact.GetHelmArtifactDetails500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, nil
}

func isHelmVersionTag(r artifact.GetHelmArtifactDetailsRequestObject) bool {
	return r.Params.VersionType != nil && *r.Params.VersionType == artifact.GetHelmArtifactDetailsParamsVersionTypeTAG
}
