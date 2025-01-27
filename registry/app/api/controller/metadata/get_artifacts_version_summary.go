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
)

func (c *APIController) GetArtifactVersionSummary(
	ctx context.Context,
	r artifact.GetArtifactVersionSummaryRequestObject,
) (artifact.GetArtifactVersionSummaryResponseObject, error) {
	image, version, pkgType, isLatestTag, err := c.FetchArtifactSummary(ctx, r)
	if err != nil {
		return artifact.GetArtifactVersionSummary500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	return artifact.GetArtifactVersionSummary200JSONResponse{
		ArtifactVersionSummaryResponseJSONResponse: *GetArtifactVersionSummary(image, pkgType, version, isLatestTag),
	}, nil
}

// FetchArtifactSummary helper function for common logic.
func (c *APIController) FetchArtifactSummary(
	ctx context.Context,
	r artifact.GetArtifactVersionSummaryRequestObject,
) (string, string, artifact.PackageType, bool, error) {
	regInfo, err := c.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))

	if err != nil {
		return "", "", "", false, fmt.Errorf("failed to get registry request base info: %w", err)
	}

	space, err := c.SpaceStore.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return "", "", "", false, err
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := GetPermissionChecks(space, regInfo.RegistryIdentifier, enum.PermissionRegistryView)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return "", "", "", false, err
	}

	image := string(r.Artifact)
	version := string(r.Version)

	registry, err := c.RegistryRepository.Get(ctx, regInfo.RegistryID)
	if err != nil {
		return "", "", "", false, err
	}

	if registry.PackageType == artifact.PackageTypeDOCKER || registry.PackageType == artifact.PackageTypeHELM {
		tag, err := c.TagStore.GetTagMetadata(ctx, regInfo.parentID, regInfo.RegistryIdentifier, image, version)
		if err != nil {
			return "", "", "", false, err
		}

		latestTag, _ := c.TagStore.GetLatestTagName(ctx, regInfo.parentID, regInfo.RegistryIdentifier, image)
		isLatestTag := latestTag == version

		return image, tag.Name, tag.PackageType, isLatestTag, nil
	}
	artifact, err := c.ArtifactStore.GetArtifactMetadata(ctx, regInfo.parentID,
		regInfo.RegistryIdentifier, image, version)

	if err != nil {
		return "", "", "", false, err
	}

	latestTag, _ := c.ArtifactStore.GetLatestVersionName(ctx, regInfo.parentID, regInfo.RegistryIdentifier, image)
	isLatestTag := latestTag == version

	return image, artifact.Name, artifact.PackageType, isLatestTag, nil
}
