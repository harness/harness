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
	"strconv"
	"strings"

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
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		//nolint:nilerr
		return artifact.DeleteRegistry400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}
	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		//nolint:nilerr
		return artifact.DeleteRegistry400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space,
		regInfo.RegistryIdentifier, enum.PermissionRegistryDelete)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		//nolint:nilerr
		return artifact.DeleteRegistry403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	repoEntity, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.ParentID, regInfo.RegistryIdentifier)
	if err != nil {
		//nolint:nilerr
		return artifact.DeleteRegistry404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "registry doesn't exist with this key"),
			),
		}, nil
	}

	err = c.checkIfRegistryUsedAsUpstream(
		ctx, regInfo, repoEntity.Name, repoEntity.ID,
	)

	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to delete upstream proxies for registry: %s with error: %s",
			regInfo.RegistryIdentifier, err)
		//nolint:nilerr
		return artifact.DeleteRegistry400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	err = c.tx.WithTx(
		ctx, func(ctx context.Context) error {
			err = c.deleteRegistryWithAudit(ctx, regInfo, repoEntity, session.Principal, regInfo.ParentRef)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("failed to delete registry: %s with error: %s",
					regInfo.RegistryIdentifier, err)
				return fmt.Errorf("failed to delete registry: %w", err)
			}
			return nil
		},
	)

	if err != nil {
		if strings.Contains(err.Error(), "delete query failed") {
			msg := "Internal Error"
			err = fmt.Errorf("failed to delete registry: %s", msg)
		}
		//nolint:nilerr
		return throwDeleteRegistry500Error(err), nil
	}
	return artifact.DeleteRegistry200JSONResponse{
		SuccessJSONResponse: artifact.SuccessJSONResponse(*GetSuccessResponse()),
	}, nil
}

func (c *APIController) checkIfRegistryUsedAsUpstream(
	ctx context.Context,
	regInfo *registrytypes.RegistryRequestBaseInfo,
	registryName string,
	registryID int64,
) error {
	registryIDs, err := c.RegistryRepository.FetchRegistriesIDByUpstreamProxyID(
		ctx, strconv.FormatInt(registryID, 10), regInfo.RootIdentifierID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to fetch registryIDs: %s", err)
		return fmt.Errorf("failed to fetch registryIDs IDs: %w", err)
	}
	if len(registryIDs) > 0 {
		registries, err := c.RegistryRepository.GetByIDIn(ctx, registryIDs)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("failed to fetch registries: %s", err)
			return fmt.Errorf("failed to fetch registries: %w", err)
		}
		var registryScopeMappings []string
		for _, registry := range *registries {
			name := registry.Name
			space, err := c.SpaceFinder.FindByID(ctx, registry.ParentID)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("failed to fetch space details: %s", err)
				continue
			}
			path := space.Path
			registryScopeMappings = append(registryScopeMappings, name+" ("+path+")")
		}
		return fmt.Errorf(
			"upstream Proxy: [%s] is being used inside Registry: [%s]",
			registryName, strings.Join(registryScopeMappings, ", "),
		)
	}

	return nil
}

func (c *APIController) deleteRegistryWithAudit(
	ctx context.Context, regInfo *registrytypes.RegistryRequestBaseInfo,
	registry *registrytypes.Registry, principal types.Principal, parentRef string,
) error {
	err := c.PublicAccess.Delete(ctx, enum.PublicResourceTypeRepo, parentRef+"/"+registry.Name)
	if err != nil {
		return fmt.Errorf("failed to delete public access for repo: %w", err)
	}
	err = c.RegFinder.Delete(ctx, regInfo.ParentID, regInfo.RegistryIdentifier)
	if err != nil {
		return err
	}

	typeRegistry := audit.ResourceTypeRegistry
	if registry.Type == artifact.RegistryTypeUPSTREAM {
		typeRegistry = audit.ResourceTypeRegistryUpstreamProxy
	}
	auditErr := c.AuditService.Log(
		ctx,
		principal,
		audit.NewResource(typeRegistry, registry.Name),
		audit.ActionDeleted,
		parentRef,
		audit.WithOldObject(
			audit.RegistryObject{
				Registry: *registry,
			},
		),
		audit.WithData("registry name", registry.Name),
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
