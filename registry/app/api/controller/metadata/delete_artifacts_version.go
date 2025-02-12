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
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) DeleteArtifactVersion(ctx context.Context, r artifact.DeleteArtifactVersionRequestObject) (
	artifact.DeleteArtifactVersionResponseObject, error) {
	regInfo, err := c.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
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
		return artifact.DeleteArtifactVersion403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, err
	}

	repoEntity, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.parentID, regInfo.RegistryIdentifier)
	if len(repoEntity.Name) == 0 {
		return artifact.DeleteArtifactVersion404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "registry doesn't exist with this key"),
			),
		}, nil
	}
	if err != nil {
		return throwDeleteArtifactVersion500Error(err), err
	}

	err = c.deleteTagWithAudit(ctx, regInfo, repoEntity.Name, session.Principal, string(r.Artifact),
		string(r.Version))

	if err != nil {
		return throwDeleteArtifactVersion500Error(err), err
	}
	return artifact.DeleteArtifactVersion200JSONResponse{
		SuccessJSONResponse: artifact.SuccessJSONResponse(*GetSuccessResponse()),
	}, nil
}

func (c *APIController) deleteTagWithAudit(
	ctx context.Context, regInfo *RegistryRequestBaseInfo,
	registryName string, principal types.Principal, artifactName string, versionName string) error {
	err := c.TagStore.DeleteTag(ctx, regInfo.RegistryID, artifactName, versionName)
	if err != nil {
		return err
	}
	auditErr := c.AuditService.Log(
		ctx,
		principal,
		audit.NewResource(audit.ResourceTypeRegistry, artifactName),
		audit.ActionDeleted,
		regInfo.ParentRef,
		audit.WithData("registry name", registryName),
		audit.WithData("artifact name", artifactName),
		audit.WithData("version name", versionName),
	)
	if auditErr != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for delete tag operation: %s", auditErr)
	}
	return err
}

func throwDeleteArtifactVersion500Error(err error) artifact.DeleteArtifactVersion500JSONResponse {
	return artifact.DeleteArtifactVersion500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}
}
