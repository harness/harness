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
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types/enum"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

// ArtifactSummary holds the summary information for an artifact version.
type ArtifactSummary struct {
	Image            string
	Version          string
	PackageType      artifact.PackageType
	IsQuarantined    bool
	QuarantineReason string
	ArtifactType     *artifact.ArtifactType
	DeletedAt        *time.Time
	ArtifactUUID     string
	RegistryUUID     string
}

func (c *APIController) GetArtifactVersionSummary(
	ctx context.Context,
	r artifact.GetArtifactVersionSummaryRequestObject,
) (artifact.GetArtifactVersionSummaryResponseObject, error) {
	summary, err := c.FetchArtifactSummary(ctx, r)
	if err != nil {
		if errors.Is(err, apiauth.ErrUnauthorized) {
			return artifact.GetArtifactVersionSummary401JSONResponse{
				UnauthenticatedJSONResponse: artifact.UnauthenticatedJSONResponse(
					*GetErrorResponse(http.StatusUnauthorized, err.Error()),
				),
			}, nil
		}
		if errors.Is(err, apiauth.ErrForbidden) {
			return artifact.GetArtifactVersionSummary403JSONResponse{
				UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
					*GetErrorResponse(http.StatusForbidden, err.Error()),
				),
			}, nil
		}
		return artifact.GetArtifactVersionSummary500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	return artifact.GetArtifactVersionSummary200JSONResponse{
		ArtifactVersionSummaryResponseJSONResponse: *GetArtifactVersionSummary(summary.Image,
			summary.PackageType, summary.Version, summary.IsQuarantined, summary.QuarantineReason,
			summary.ArtifactType, summary.DeletedAt, summary.ArtifactUUID, summary.RegistryUUID),
	}, nil
}

// FetchArtifactSummary helper function for common logic.
func (c *APIController) FetchArtifactSummary(
	ctx context.Context,
	r artifact.GetArtifactVersionSummaryRequestObject,
) (*ArtifactSummary, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))

	if err != nil {
		return nil, fmt.Errorf("failed to get registry request base info: %w", err)
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return nil, err
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
		return nil, err
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

	registry, err := c.RegistryRepository.Get(ctx, regInfo.RegistryID, types.WithAllDeleted())
	if err != nil {
		return nil, err
	}

	var artifactType *artifact.ArtifactType
	if r.Params.ArtifactType != nil {
		artifactType, err = ValidateAndGetArtifactType(registry.PackageType, string(*r.Params.ArtifactType))
		if err != nil {
			return nil, err
		}
	}

	quarantinePath, err := c.QuarantineArtifactRepository.GetByFilePath(ctx,
		"", regInfo.RegistryID, image, artifactVersion, artifactType)

	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to get quarantine path")
	}

	var quarantineReason string
	var isQuarantined bool
	if len(quarantinePath) > 0 {
		isQuarantined = true
		quarantineReason = quarantinePath[0].Reason
	}

	//nolint:nestif
	if registry.PackageType == artifact.PackageTypeDOCKER || registry.PackageType == artifact.PackageTypeHELM {
		var ociVersion *types.OciVersionMetadata
		if c.UntaggedImagesEnabled(ctx) {
			var d string
			if r.Params.Digest != nil && strings.TrimSpace(string(*r.Params.Digest)) != "" {
				d = string(*r.Params.Digest)
			} else {
				d = version
			}
			parsedDigest, err := types.NewDigest(digest.Digest(d))
			if err != nil {
				return nil, err
			}
			art, err := c.ArtifactStore.GetArtifactMetadata(ctx, regInfo.ParentID, regInfo.RegistryIdentifier, image,
				parsedDigest.String(), artifactType, types.WithAllDeleted())
			if err != nil {
				return nil, err
			}

			return &ArtifactSummary{
				Image:            image,
				Version:          version,
				PackageType:      art.PackageType,
				IsQuarantined:    isQuarantined,
				QuarantineReason: quarantineReason,
				ArtifactType:     art.ArtifactType,
				DeletedAt:        art.DeletedAt,
				ArtifactUUID:     art.UUID,
				RegistryUUID:     registry.UUID,
			}, nil
		}

		ociVersion, err = c.TagStore.GetTagMetadata(ctx, regInfo.ParentID, regInfo.RegistryIdentifier, image, version)
		if err != nil {
			return nil, err
		}

		var deletedAt *time.Time
		if ociVersion.ArtifactDeletedAt != nil {
			deletedAt = ociVersion.ArtifactDeletedAt
		}

		// Artifact UUID is fetched directly from the database via join in GetOCIVersionMetadata
		return &ArtifactSummary{
			Image:            image,
			Version:          ociVersion.Name,
			PackageType:      ociVersion.PackageType,
			IsQuarantined:    isQuarantined,
			QuarantineReason: quarantineReason,
			ArtifactType:     nil,
			DeletedAt:        deletedAt,
			ArtifactUUID:     ociVersion.ArtifactUUID,
			RegistryUUID:     registry.UUID,
		}, nil
	}
	metadata, err := c.ArtifactStore.GetArtifactMetadata(
		ctx, regInfo.ParentID, regInfo.RegistryIdentifier, image, version, artifactType, types.WithAllDeleted())

	if err != nil {
		return nil, err
	}

	return &ArtifactSummary{
		Image:            image,
		Version:          metadata.Name,
		PackageType:      metadata.PackageType,
		IsQuarantined:    isQuarantined,
		QuarantineReason: quarantineReason,
		ArtifactType:     metadata.ArtifactType,
		DeletedAt:        metadata.DeletedAt,
		ArtifactUUID:     metadata.UUID,
		RegistryUUID:     registry.UUID,
	}, nil
}
