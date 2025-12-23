// Copyright 2023 Harness, Inc.
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
	"net/http"
	"path"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/api/utils"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

func (c *APIController) QuarantineFilePath(
	ctx context.Context,
	r artifact.QuarantineFilePathRequestObject,
) (artifact.QuarantineFilePathResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.QuarantineFilePath400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.QuarantineFilePath400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space,
		regInfo.RegistryIdentifier, enum.PermissionRegistryEdit)

	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		statusCode, message := HandleAuthError(err)
		if statusCode == http.StatusUnauthorized {
			return artifact.QuarantineFilePath401JSONResponse{
				UnauthenticatedJSONResponse: artifact.UnauthenticatedJSONResponse(
					*GetErrorResponse(http.StatusUnauthorized, message),
				),
			}, nil
		}
		return artifact.QuarantineFilePath403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, message),
			),
		}, nil
	}

	artifactName := r.Body.Artifact
	version := r.Body.Version
	filePath := ""
	if r.Body.FilePath != nil {
		filePath = *r.Body.FilePath
	}
	reason := r.Body.Reason

	var artifactType *artifact.ArtifactType
	if r.Body.ArtifactType != nil {
		at := artifact.ArtifactType(*r.Body.ArtifactType)
		artifactType = &at
	}

	img, err := c.ImageStore.GetByNameAndType(ctx, regInfo.RegistryID, artifactName, artifactType)
	if err != nil {
		if errors.Is(err, store.ErrResourceNotFound) {
			return artifact.QuarantineFilePath400JSONResponse{
				BadRequestJSONResponse: artifact.BadRequestJSONResponse(
					*GetErrorResponse(http.StatusNotFound, "image not found"),
				),
			}, nil
		}
		return artifact.QuarantineFilePath500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	var versionID int64
	var rootPath string
	if version != nil { //nolint:nestif
		if regInfo.PackageType == artifact.PackageTypeDOCKER || regInfo.PackageType == artifact.PackageTypeHELM {
			parsedDigest, err := digest.Parse(*version)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to parse digest for create quarantine file path")
				return artifact.QuarantineFilePath500JSONResponse{
					InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
						*GetErrorResponse(http.StatusInternalServerError, err.Error()),
					),
				}, nil
			}
			typesDigest, err := types.NewDigest(parsedDigest)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to create types digest for create quarantine file path")
				return artifact.QuarantineFilePath500JSONResponse{
					InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
						*GetErrorResponse(http.StatusInternalServerError, err.Error()),
					),
				}, nil
			}
			digestVal := typesDigest.String()
			version = &digestVal
		}
		art, err := c.ArtifactStore.GetByName(ctx, img.ID, *version)
		if err != nil {
			if errors.Is(err, store.ErrResourceNotFound) {
				return artifact.QuarantineFilePath400JSONResponse{
					BadRequestJSONResponse: artifact.BadRequestJSONResponse(
						*GetErrorResponse(http.StatusNotFound, "version not found"),
					),
				}, nil
			}
		}
		if err != nil {
			return artifact.QuarantineFilePath500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		versionID = art.ID
	}

	var nodeID *string
	if filePath != "" {
		if version == nil {
			return artifact.QuarantineFilePath400JSONResponse{
				BadRequestJSONResponse: artifact.BadRequestJSONResponse(
					*GetErrorResponse(http.StatusBadRequest, "version not provided"),
				),
			}, nil
		}
		rootPath, err = utils.GetFilePath(regInfo.PackageType, artifactName, *version)
		if err != nil {
			return artifact.QuarantineFilePath500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		filePath = path.Join(rootPath, filePath)
		node, err := c.fileManager.GetNode(ctx, regInfo.RegistryID, filePath)
		if errors.Is(err, store.ErrResourceNotFound) {
			return artifact.QuarantineFilePath400JSONResponse{
				BadRequestJSONResponse: artifact.BadRequestJSONResponse(
					*GetErrorResponse(http.StatusNotFound, "file not found"),
				),
			}, nil
		}
		if err != nil {
			return artifact.QuarantineFilePath500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		nodeID = &node.ID
	}

	quarantineArtifact := &types.QuarantineArtifact{
		NodeID:     nodeID,
		Reason:     reason,
		RegistryID: regInfo.RegistryID,
		ArtifactID: versionID,
		ImageID:    img.ID,
	}
	err = c.QuarantineArtifactRepository.Create(ctx, quarantineArtifact)
	if err != nil {
		return artifact.QuarantineFilePath500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	// Evict cache after creating quarantine entry
	if version != nil {
		c.QuarantineFinder.EvictCache(ctx, regInfo.RegistryID, artifactName, *version, artifactType)
	}

	return artifact.QuarantineFilePath200JSONResponse{
		QuarantinePathResponseJSONResponse: *GetQuarantinePathJSONResponse(
			quarantineArtifact.ID, regInfo.RegistryID,
			img.ID, versionID, quarantineArtifact.Reason, filePath),
	}, nil
}
