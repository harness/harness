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
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryTypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) DeleteArtifact(ctx context.Context, r artifact.DeleteArtifactRequestObject) (
	artifact.DeleteArtifactResponseObject, error) {
	regInfo, err := c.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.DeleteArtifact400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, err
	}
	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.DeleteArtifact400JSONResponse{
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
		return artifact.DeleteArtifact403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, err
	}

	repoEntity, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.parentID, regInfo.RegistryIdentifier)
	if len(repoEntity.Name) == 0 {
		return artifact.DeleteArtifact404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "registry doesn't exist with this key"),
			),
		}, nil
	}
	if err != nil {
		return throwDeleteArtifact500Error(err), err
	}

	artifactName := string(r.Artifact)
	artifactDetails, err := c.ImageStore.GetByName(ctx, regInfo.RegistryID, artifactName)
	if err != nil || artifactDetails == nil {
		return artifact.DeleteArtifact404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "artifact doesn't exist with this key"),
			),
		}, err
	}
	if !artifactDetails.Enabled {
		return artifact.DeleteArtifact404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "artifact is already deleted"),
			),
		}, nil
	}
	err = c.tx.WithTx(
		ctx, func(ctx context.Context) error {
			err = c.disableImageStatus(
				ctx, regInfo, artifactName,
			)

			if err != nil {
				return fmt.Errorf("failed to delete artifact: %w", err)
			}

			err := c.TagStore.DeleteTagsByImageName(ctx, regInfo.RegistryID, artifactName)

			if err != nil {
				return fmt.Errorf("failed to delete artifact: %w", err)
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

			return nil
		},
	)

	if err != nil {
		return throwDeleteArtifact500Error(err), err
	}
	return artifact.DeleteArtifact200JSONResponse{
		SuccessJSONResponse: artifact.SuccessJSONResponse(*GetSuccessResponse()),
	}, nil
}

func (c *APIController) disableImageStatus(
	ctx context.Context,
	regInfo *RegistryRequestBaseInfo, artifactName string,
) error {
	image := &registryTypes.Image{
		Name:       artifactName,
		RegistryID: regInfo.RegistryID,
		Enabled:    false,
	}
	err := c.ImageStore.UpdateStatus(ctx, image)
	return err
}

func throwDeleteArtifact500Error(err error) artifact.DeleteArtifact500JSONResponse {
	return artifact.DeleteArtifact500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}
}
