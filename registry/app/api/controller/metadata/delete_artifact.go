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

func (c *APIController) DeleteArtifact(ctx context.Context, r artifact.DeleteArtifactRequestObject) (
	artifact.DeleteArtifactResponseObject, error,
) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.DeleteArtifact400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}
	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.DeleteArtifact400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
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
			return artifact.DeleteArtifact401JSONResponse{
				UnauthenticatedJSONResponse: artifact.UnauthenticatedJSONResponse(
					*GetErrorResponse(http.StatusUnauthorized, message),
				),
			}, nil
		}
		return artifact.DeleteArtifact403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(
					http.StatusForbidden,
					message,
				),
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
		if errors.Is(err, store.ErrResourceNotFound) {
			//nolint:nilerr
			return artifact.DeleteArtifact404JSONResponse{
				NotFoundJSONResponse: artifact.NotFoundJSONResponse(
					*GetErrorResponse(http.StatusNotFound,
						fmt.Sprintf("registry %s doesn't exist", regInfo.RegistryIdentifier)),
				),
			}, nil
		}
		//nolint:nilerr
		return artifact.DeleteArtifact500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, "failed to get registry"),
			),
		}, nil
	}

	artifactName := string(r.Artifact)
	_, err = c.ImageStore.GetByName(ctx, regInfo.RegistryID, artifactName, types.WithAllDeleted())
	if err != nil {
		//nolint:nilerr
		return artifact.DeleteArtifact404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "artifact doesn't exist with this key"),
			),
		}, nil
	}

	err = c.DeletionService.DeleteImageByPackageType(ctx, regInfo, regInfo.PackageType, artifactName)
	if err != nil {
		return artifact.DeleteArtifact500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, err
	}

	auditErr := c.AuditService.Log(
		ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeRegistryArtifact, string(r.Artifact)),
		audit.ActionDeleted,
		regInfo.ParentRef,
		audit.WithData("registry name", repoEntity.Name),
		audit.WithData("artifact name", string(r.Artifact)),
	)
	if auditErr != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for delete tag operation: %s", auditErr)
	}

	return artifact.DeleteArtifact200JSONResponse{
		SuccessJSONResponse: artifact.SuccessJSONResponse(*GetSuccessResponse()),
	}, nil
}
