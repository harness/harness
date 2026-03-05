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
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) DeleteArtifactVersion(ctx context.Context, r artifact.DeleteArtifactVersionRequestObject) (
	artifact.DeleteArtifactVersionResponseObject, error,
) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.DeleteArtifactVersion400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, err
	}
	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.DeleteArtifactVersion400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, err
	}

	session, _ := request.AuthSessionFrom(ctx)
	if err = apiauth.CheckSpaceScope(
		ctx,
		c.Authorizer,
		session,
		space,
		enum.ResourceTypeRegistry,
		enum.PermissionArtifactsDelete,
	); err != nil {
		statusCode, message := HandleAuthError(err)
		if statusCode == http.StatusUnauthorized {
			return artifact.DeleteArtifactVersion401JSONResponse{
				UnauthenticatedJSONResponse: artifact.UnauthenticatedJSONResponse(
					*GetErrorResponse(http.StatusUnauthorized, message),
				),
			}, nil
		}
		return artifact.DeleteArtifactVersion403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, message),
			),
		}, nil
	}

	repoEntity, err := c.RegistryRepository.GetByParentIDAndName(
		ctx,
		regInfo.ParentID,
		regInfo.RegistryIdentifier,
		types.WithAllDeleted(),
	)
	if err != nil {
		//nolint:nilerr
		return artifact.DeleteArtifactVersion404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(
					http.StatusNotFound,
					fmt.Sprintf("registry %s doesn't exist", regInfo.RegistryIdentifier),
				),
			),
		}, nil
	}

	artifactName := string(r.Artifact)
	versionName := string(r.Version)
	registryName := repoEntity.Name

	_, err = c.ImageStore.GetByName(ctx, repoEntity.ID, artifactName, types.WithAllDeleted())
	if err != nil {
		//nolint:nilerr
		return artifact.DeleteArtifactVersion404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "image doesn't exist with this key"),
			),
		}, nil
	}

	// Delete artifact version.
	err = c.DeletionService.DeleteArtifactVersionByPackageType(
		ctx, regInfo, artifactName, versionName, &session.Principal.ID, registryName,
	)
	if err != nil {
		if errors.Is(err, store.ErrResourceNotFound) {
			return artifact.DeleteArtifactVersion404JSONResponse{
				NotFoundJSONResponse: artifact.NotFoundJSONResponse(
					*GetErrorResponse(
						http.StatusNotFound,
						fmt.Sprintf("artifact version '%s' not found for artifact '%s'", versionName, artifactName),
					),
				),
			}, nil
		}
		return throwDeleteArtifactVersion500Error(err), nil
	}

	auditErr := c.AuditService.Log(
		ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeRegistry, artifactName),
		audit.ActionDeleted,
		regInfo.ParentRef,
		audit.WithData("registry name", registryName),
		audit.WithData("artifact name", artifactName),
		audit.WithData("version name", versionName),
	)
	if auditErr != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for delete artifact operation: %s", auditErr)
	}

	return artifact.DeleteArtifactVersion200JSONResponse{
		SuccessJSONResponse: artifact.SuccessJSONResponse(*GetSuccessResponse()),
	}, nil
}

func throwDeleteArtifactVersion500Error(err error) artifact.DeleteArtifactVersion500JSONResponse {
	return artifact.DeleteArtifactVersion500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}
}
