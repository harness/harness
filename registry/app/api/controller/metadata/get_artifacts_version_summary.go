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
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types/enum"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

func (c *APIController) GetArtifactVersionSummary(
	ctx context.Context,
	r artifact.GetArtifactVersionSummaryRequestObject,
) (artifact.GetArtifactVersionSummaryResponseObject, error) {
	image, version, pkgType, isQuarantine, quarantineReason, artifactType, err := c.FetchArtifactSummary(ctx, r)
	if err != nil {
		return artifact.GetArtifactVersionSummary500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	return artifact.GetArtifactVersionSummary200JSONResponse{
		ArtifactVersionSummaryResponseJSONResponse: *GetArtifactVersionSummary(image,
			pkgType, version, isQuarantine, quarantineReason, artifactType),
	}, nil
}

// FetchArtifactSummary helper function for common logic.
func (c *APIController) FetchArtifactSummary(
	ctx context.Context,
	r artifact.GetArtifactVersionSummaryRequestObject,
) (string, string, artifact.PackageType, bool, string, *artifact.ArtifactType, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))

	if err != nil {
		return "", "", "", false, "", nil, fmt.Errorf("failed to get registry request base info: %w", err)
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return "", "", "", false, "", nil, err
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
		return "", "", "", false, "", nil, err
	}

	image := string(r.Artifact)
	version := string(r.Version)
	artifactVersion := version

	if r.Params.Digest != nil && strings.TrimSpace(string(*r.Params.Digest)) != "" {
		parsedDigest, err := types.NewDigest(digest.Digest(*r.Params.Digest))
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("Failed to parse digest")
		}
		artifactVersion = parsedDigest.String()
	}

	registry, err := c.RegistryRepository.Get(ctx, regInfo.RegistryID)
	if err != nil {
		return "", "", "", false, "", nil, err
	}

	quarantinePath, err := c.QuarantineArtifactRepository.GetByFilePath(ctx,
		"", regInfo.RegistryID, image, artifactVersion)

	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to get quarantine path")
	}

	var quarantineReason string
	var isQuarantined bool
	if len(quarantinePath) > 0 {
		isQuarantined = true
		quarantineReason = quarantinePath[0].Reason
	}
	var artifactType *artifact.ArtifactType
	if r.Params.ArtifactType != nil {
		artifactType, err = ValidateAndGetArtifactType(registry.PackageType, string(*r.Params.ArtifactType))
		if err != nil {
			return "", "", "", false, "", nil, err
		}
	}

	if registry.PackageType == artifact.PackageTypeDOCKER || registry.PackageType == artifact.PackageTypeHELM {
		tag, err := c.TagStore.GetTagMetadata(ctx, regInfo.ParentID, regInfo.RegistryIdentifier, image, version)
		if err != nil {
			return "", "", "", false, "", nil, err
		}

		return image, tag.Name, tag.PackageType, isQuarantined, quarantineReason, nil, nil
	}
	art, err := c.ArtifactStore.GetArtifactMetadata(ctx, regInfo.ParentID, regInfo.RegistryIdentifier, image,
		version, artifactType)

	if err != nil {
		return "", "", "", false, "", nil, err
	}

	return image, art.Name, art.PackageType, isQuarantined, quarantineReason, art.ArtifactType, nil
}
