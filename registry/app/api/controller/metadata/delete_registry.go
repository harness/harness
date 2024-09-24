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
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) DeleteRegistry(
	ctx context.Context,
	r artifact.DeleteRegistryRequestObject,
) (artifact.DeleteRegistryResponseObject, error) {
	regInfo, err := c.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.DeleteRegistry400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, err
	}
	space, err := c.SpaceStore.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.DeleteRegistry400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, err
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := GetPermissionChecks(space, regInfo.RegistryIdentifier, enum.PermissionRegistryDelete)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return artifact.DeleteRegistry403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, err
	}

	repoEntity, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.parentID, regInfo.RegistryIdentifier)
	if len(repoEntity.Name) == 0 {
		return artifact.DeleteRegistry404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "registry doesn't exist with this key"),
			),
		}, nil
	}
	if err != nil {
		return throwDeleteRegistry500Error(err), err
	}

	if string(repoEntity.Type) == string(artifact.RegistryTypeVIRTUAL) {
		err = c.deleteRegistryWithAudit(ctx, regInfo, repoEntity, session.Principal, regInfo.ParentRef)
	} else {
		err = c.tx.WithTx(
			ctx, func(ctx context.Context) error {
				err = c.deleteUpstreamProxyWithAudit(
					ctx, regInfo, session.Principal, regInfo.ParentRef, repoEntity.Name,
				)

				if err != nil {
					return fmt.Errorf("failed to delete upstream proxy: %w", err)
				}

				err = c.deleteRegistryWithAudit(ctx, regInfo, repoEntity, session.Principal, regInfo.ParentRef)

				if err != nil {
					return fmt.Errorf("failed to delete registry: %w", err)
				}

				return nil
			},
		)
	}
	if err != nil {
		return throwDeleteRegistry500Error(err), err
	}
	return artifact.DeleteRegistry200JSONResponse{
		SuccessJSONResponse: artifact.SuccessJSONResponse(*GetSuccessResponse()),
	}, nil
}

func (c *APIController) deleteUpstreamProxyWithAudit(
	ctx context.Context,
	regInfo *RegistryRequestBaseInfo, principal types.Principal, parentRef string, registryName string,
) error {
	err := c.UpstreamProxyStore.Delete(ctx, regInfo.parentID, regInfo.RegistryIdentifier)
	if err != nil {
		return err
	}

	auditErr := c.AuditService.Log(
		ctx,
		principal,
		audit.NewResource(audit.ResourceTypeRegistryUpstreamProxy, registryName),
		audit.ActionDeleted,
		parentRef,
		audit.WithData("registry name", registryName),
	)
	if auditErr != nil {
		log.Ctx(ctx).Warn().Msgf(
			"failed to insert audit log for delete upstream proxy config operation: %s", auditErr,
		)
	}

	return err
}

func (c *APIController) deleteRegistryWithAudit(
	ctx context.Context, regInfo *RegistryRequestBaseInfo,
	registry *registrytypes.Registry, principal types.Principal, parentRef string,
) error {
	err := c.RegistryRepository.Delete(ctx, regInfo.parentID, regInfo.RegistryIdentifier)
	if err != nil {
		return err
	}
	auditErr := c.AuditService.Log(
		ctx,
		principal,
		audit.NewResource(audit.ResourceTypeRegistry, registry.Name),
		audit.ActionDeleted,
		parentRef,
		audit.WithOldObject(
			audit.RegistryObject{
				Registry: *registry,
			},
		),
	)
	if auditErr != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for delete registry operation: %s", auditErr)
	}
	return err
}

func throwDeleteRegistry500Error(err error) artifact.DeleteRegistry500JSONResponse {
	return artifact.DeleteRegistry500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}
}
