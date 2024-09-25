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
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"
	gitnessenum "github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) CreateRegistry(
	ctx context.Context,
	r artifact.CreateRegistryRequestObject,
) (artifact.CreateRegistryResponseObject, error) {
	registryRequest := artifact.RegistryRequest(*r.Body)
	parentRef := artifact.SpaceRefPathParam(*registryRequest.ParentRef)

	regInfo, err := c.GetRegistryRequestBaseInfo(ctx, string(parentRef), "")
	if err != nil {
		return artifact.CreateRegistry400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, err
	}

	space, err := c.SpaceStore.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.CreateRegistry400JSONResponse{
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
		gitnessenum.ResourceTypeRegistry,
		gitnessenum.PermissionRegistryEdit,
	); err != nil {
		return artifact.CreateRegistry403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, err
	}

	if registryRequest.Config.Type == artifact.RegistryTypeVIRTUAL {
		return c.createVirtualRegistry(ctx, registryRequest, regInfo, session, parentRef)
	}
	registry, upstreamproxy, err := c.CreateUpstreamProxyEntity(
		ctx,
		registryRequest,
		regInfo.parentID, regInfo.rootIdentifierID,
	)
	var registryID int64
	if err != nil {
		return throwCreateRegistry400Error(err), err
	}

	err = c.tx.WithTx(
		ctx, func(ctx context.Context) error {
			registryID, err = c.createRegistryWithAudit(ctx, registry, session.Principal, string(parentRef))

			if err != nil {
				return fmt.Errorf("failed to create registry: %w", err)
			}

			upstreamproxy.RegistryID = registryID

			_, err = c.createUpstreamProxyWithAudit(
				ctx, upstreamproxy, session.Principal, string(parentRef), registry.Name,
			)

			if err != nil {
				return fmt.Errorf("failed to create upstream proxy: %w", err)
			}
			return nil
		},
	)

	if err != nil {
		return throwCreateRegistry400Error(err), err
	}
	upstreamproxyEntity, err := c.UpstreamProxyStore.Get(ctx, registryID)
	if err != nil {
		return throwCreateRegistry400Error(err), err
	}

	return artifact.CreateRegistry201JSONResponse{
		RegistryResponseJSONResponse: *CreateUpstreamProxyResponseJSONResponse(upstreamproxyEntity),
	}, nil
}

func (c *APIController) createVirtualRegistry(
	ctx context.Context, registryRequest artifact.RegistryRequest, regInfo *RegistryRequestBaseInfo,
	session *auth.Session, parentRef artifact.SpaceRefPathParam,
) (artifact.CreateRegistryResponseObject, error) {
	registry, err := CreateRegistryEntity(registryRequest, regInfo.parentID, regInfo.rootIdentifierID)
	if err != nil {
		return throwCreateRegistry400Error(err), nil
	}
	err = c.setUpstreamProxyIDs(ctx, registry, registryRequest, regInfo.parentID)
	if err != nil {
		return throwCreateRegistry400Error(err), nil
	}
	id, err := c.createRegistryWithAudit(ctx, registry, session.Principal, string(parentRef))
	if err != nil {
		return throwCreateRegistry400Error(err), nil
	}
	repoEntity, err := c.RegistryRepository.Get(ctx, id)
	if err != nil {
		return throwCreateRegistry400Error(err), nil
	}
	cleanupPolicies, err := c.CleanupPolicyStore.GetByRegistryID(ctx, repoEntity.ID)
	if err != nil {
		return throwCreateRegistry400Error(err), nil
	}

	return artifact.CreateRegistry201JSONResponse{
		RegistryResponseJSONResponse: *CreateVirtualRepositoryResponse(
			repoEntity, c.getUpstreamProxyKeys(ctx, repoEntity.UpstreamProxies),
			cleanupPolicies, regInfo.RootIdentifier, c.URLProvider.RegistryURL(),
		),
	}, nil
}

func (c *APIController) createUpstreamProxyWithAudit(
	ctx context.Context,
	upstreamProxy *registrytypes.UpstreamProxyConfig, principal types.Principal,
	parentRef string, registryName string,
) (int64, error) {
	id, err := c.UpstreamProxyStore.Create(ctx, upstreamProxy)
	if err != nil {
		return id, err
	}
	auditErr := c.AuditService.Log(
		ctx,
		principal,
		audit.NewResource(audit.ResourceTypeRegistryUpstreamProxy, registryName),
		audit.ActionCreated,
		parentRef,
		audit.WithNewObject(
			audit.RegistryUpstreamProxyConfigObject{
				ID:         id,
				RegistryID: upstreamProxy.RegistryID,
				Source:     upstreamProxy.Source,
				URL:        upstreamProxy.URL,
				AuthType:   upstreamProxy.AuthType,
				CreatedAt:  upstreamProxy.CreatedAt,
				UpdatedAt:  upstreamProxy.UpdatedAt,
				CreatedBy:  upstreamProxy.CreatedBy,
				UpdatedBy:  upstreamProxy.UpdatedBy,
			},
		),
	)
	if auditErr != nil {
		log.Ctx(ctx).Warn().Msgf(
			"failed to insert audit log for create upstream proxy config operation: %s", auditErr,
		)
	}

	return id, err
}

func (c *APIController) createRegistryWithAudit(
	ctx context.Context, registry *registrytypes.Registry,
	principal types.Principal, parentRef string,
) (int64, error) {
	id, err := c.RegistryRepository.Create(ctx, registry)
	if err != nil {
		return id, err
	}
	auditErr := c.AuditService.Log(
		ctx,
		principal,
		audit.NewResource(audit.ResourceTypeRegistry, registry.Name),
		audit.ActionCreated,
		parentRef,
		audit.WithNewObject(
			audit.RegistryObject{
				Registry: *registry,
			},
		),
	)
	if auditErr != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for create registry operation: %s", auditErr)
	}
	return id, err
}

func throwCreateRegistry400Error(err error) artifact.CreateRegistry400JSONResponse {
	return artifact.CreateRegistry400JSONResponse{
		BadRequestJSONResponse: artifact.BadRequestJSONResponse(
			*GetErrorResponse(http.StatusBadRequest, err.Error()),
		),
	}
}

func CreateRegistryEntity(
	dto artifact.RegistryRequest, parentID int64,
	rootParentID int64,
) (*registrytypes.Registry, error) {
	allowedPattern, blockedPattern, description, labels := getRepoEntityFields(dto)
	e := ValidatePackageType(string(dto.PackageType))
	if e != nil {
		return nil, e
	}
	e = ValidateRepoType(string(dto.Config.Type))
	if e != nil {
		return nil, e
	}
	e = ValidateIdentifier(dto.Identifier)
	if e != nil {
		return nil, e
	}
	entity := &registrytypes.Registry{
		Name:           dto.Identifier,
		ParentID:       parentID,
		RootParentID:   rootParentID,
		Description:    description,
		AllowedPattern: allowedPattern,
		BlockedPattern: blockedPattern,
		PackageType:    dto.PackageType,
		Labels:         labels,
		Type:           dto.Config.Type,
	}
	return entity, nil
}

func (c *APIController) CreateUpstreamProxyEntity(
	ctx context.Context, dto artifact.RegistryRequest, parentID int64, rootParentID int64,
) (*registrytypes.Registry, *registrytypes.UpstreamProxyConfig, error) {
	allowedPattern := []string{}
	if dto.AllowedPattern != nil {
		allowedPattern = *dto.AllowedPattern
	}
	blockedPattern := []string{}
	if dto.BlockedPattern != nil {
		blockedPattern = *dto.BlockedPattern
	}
	e := ValidatePackageType(string(dto.PackageType))
	if e != nil {
		return nil, nil, e
	}
	e = ValidateUpstream(dto.Config)
	if e != nil {
		return nil, nil, e
	}
	e = ValidateIdentifier(dto.Identifier)
	if e != nil {
		return nil, nil, e
	}
	repoEntity := &registrytypes.Registry{
		Name:           dto.Identifier,
		ParentID:       parentID,
		RootParentID:   rootParentID,
		AllowedPattern: allowedPattern,
		BlockedPattern: blockedPattern,
		PackageType:    dto.PackageType,
		Type:           artifact.RegistryTypeUPSTREAM,
	}

	config, e := dto.Config.AsUpstreamConfig()
	if e != nil {
		return nil, nil, e
	}
	CleanURLPath(config.Url)
	upstreamProxyConfigEntity := &registrytypes.UpstreamProxyConfig{
		URL:      *config.Url,
		AuthType: string(config.AuthType),
	}
	if config.Source != nil && len(string(*config.Source)) > 0 {
		err := ValidateUpstreamSource(string(*config.Source))
		if err != nil {
			return nil, nil, err
		}
		upstreamProxyConfigEntity.Source = string(*config.Source)
	}
	if config.AuthType == artifact.AuthTypeUserPassword {
		res, err := config.Auth.AsUserPassword()
		if err != nil {
			return nil, nil, err
		}
		upstreamProxyConfigEntity.UserName = res.UserName
		if res.SecretIdentifier == nil {
			return nil, nil, fmt.Errorf("failed to create upstream proxy: secret_identifier missing")
		}

		upstreamProxyConfigEntity.SecretSpaceID, err = c.getSecretID(ctx, res.SecretSpaceId, res.SecretSpacePath)
		if err != nil {
			return nil, nil, err
		}

		upstreamProxyConfigEntity.SecretSpaceID = *res.SecretSpaceId
		upstreamProxyConfigEntity.SecretIdentifier = *res.SecretIdentifier
	}
	return repoEntity, upstreamProxyConfigEntity, nil
}

func (c *APIController) getSecretID(ctx context.Context, secretSpaceID *int, secretSpacePath *string) (int, error) {
	if secretSpaceID == nil && secretSpacePath == nil {
		return -1, fmt.Errorf("failed to create upstream proxy: secret space missing")
	}

	if secretSpaceID != nil {
		return *secretSpaceID, nil
	}

	path, err := c.spacePathStore.FindByPath(ctx, *secretSpacePath)
	if err != nil {
		return -1, fmt.Errorf("failed to get Space Path: %w", err)
	}
	return int(path.SpaceID), nil
}
