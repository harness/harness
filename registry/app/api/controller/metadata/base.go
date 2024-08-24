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
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/app/paths"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/types"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

const MediaTypeImageConfig = "application/vnd.docker.container.image.v1+json"

var _ api.StrictServerInterface = (*APIController)(nil)

type RegistryRequestBaseInfo struct {
	rootIdentifier   string
	rootIdentifierID int64

	registryRef        string
	RegistryIdentifier string
	registryID         int64

	parentRef string
	parentID  int64
}

type RegistryRequestInfo struct {
	RegistryRequestBaseInfo
	packageTypes []string
	sortByField  string
	sortByOrder  string
	offset       int
	limit        int
	pageNumber   int64
	searchTerm   string
	labels       []string
}

// GetRegistryRequestBaseInfo returns the base info for the registry request
// One of the regRefParam or (parentRefParam + regIdentifierParam) should be provided.
func (c *APIController) GetRegistryRequestBaseInfo(
	ctx context.Context,
	parentRef string,
	regRef string,
) (*RegistryRequestBaseInfo, error) {
	// ---------- CHECKS ------------
	if commons.IsEmpty(parentRef) && !commons.IsEmpty(regRef) {
		parentRef, _, _ = paths.DisectLeaf(regRef)
	}

	// ---------- PARENT ------------
	if commons.IsEmpty(parentRef) {
		return nil, fmt.Errorf("parent reference is required")
	}
	rootIdentifier, _, err := paths.DisectRoot(parentRef)
	if err != nil {
		return nil, fmt.Errorf("invalid parent reference: %w", err)
	}

	rootSpace, err := c.spaceStore.FindByRef(ctx, rootIdentifier)
	if err != nil {
		return nil, fmt.Errorf("root space not found: %w", err)
	}
	parentSpace, err := c.spaceStore.FindByRef(ctx, parentRef)
	if err != nil {
		return nil, fmt.Errorf("parent space not found: %w", err)
	}
	rootIdentifierID := rootSpace.ID
	parentID := parentSpace.ID

	baseInfo := &RegistryRequestBaseInfo{
		parentRef:        parentRef,
		parentID:         parentID,
		rootIdentifier:   rootIdentifier,
		rootIdentifierID: rootIdentifierID,
	}

	// ---------- REGISTRY  ------------
	if !commons.IsEmpty(regRef) {
		_, regIdentifier, _ := paths.DisectLeaf(regRef)

		reg, getRegistryErr := c.RegistryRepository.GetByParentIDAndName(ctx, parentID, regIdentifier)
		if getRegistryErr != nil {
			return nil, fmt.Errorf("registry not found: %w", err)
		}

		baseInfo.registryRef = regRef
		baseInfo.RegistryIdentifier = regIdentifier
		baseInfo.registryID = reg.ID
	}

	return baseInfo, nil
}

func (c *APIController) GetRegistryRequestInfo(
	ctx context.Context,
	packageTypesParam *api.PackageTypeParam,
	page *api.PageNumber,
	size *api.PageSize,
	search *api.SearchTerm,
	resource string,
	parentRef string,
	regRef string,
	labelsParam *api.LabelsParam,
	sortOrder *api.SortOrder,
	sortField *api.SortField,
) (*RegistryRequestInfo, error) {
	packageTypes := []string{}
	if packageTypesParam != nil {
		packageTypes = *packageTypesParam
	}
	sortByField := ""
	sortByOrder := ""
	if sortOrder != nil {
		sortByOrder = string(*sortOrder)
	}

	if sortField != nil {
		sortByField = string(*sortField)
	}

	labels := []string{}

	if labelsParam != nil {
		labels = *labelsParam
	}

	sortByField = GetSortByField(sortByField, resource)
	sortByOrder = GetSortByOrder(sortByOrder)

	offset := GetOffset(size, page)
	limit := GetPageLimit(size)
	pageNumber := GetPageNumber(page)

	searchTerm := ""
	if search != nil {
		searchTerm = string(*search)
	}

	baseInfo, err := c.GetRegistryRequestBaseInfo(ctx, parentRef, regRef)
	if err != nil {
		return nil, err
	}

	return &RegistryRequestInfo{
		RegistryRequestBaseInfo: *baseInfo,
		packageTypes:            packageTypes,
		sortByField:             sortByField,
		sortByOrder:             sortByOrder,
		offset:                  offset,
		limit:                   limit,
		pageNumber:              pageNumber,
		searchTerm:              searchTerm,
		labels:                  labels,
	}, nil
}

func getManifestConfig(
	ctx context.Context,
	digest digest.Digest,
	rootRef string,
	driver storagedriver.StorageDriver,
) (*manifestConfig, error) {
	var config manifestConfig
	path, err := storage.PathFn(rootRef, digest)
	if err != nil {
		return nil, fmt.Errorf("failed to get path: %w", err)
	}

	content, err := driver.GetContent(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get content for image config: %w", err)
	}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest config: %w", err)
	}

	return &config, nil
}

func (c *APIController) setUpstreamProxyIDs(
	ctx context.Context,
	registry *types.Registry,
	dto api.RegistryRequest,
	parentID int64,
) error {
	if dto.Config.Type != api.RegistryTypeVIRTUAL {
		return fmt.Errorf("invalid call to set upstream proxy ids for parentID: %d", parentID)
	}
	virtualConfig, err := dto.Config.AsVirtualConfig()
	if err != nil {
		return fmt.Errorf("failed to get virtualConfig: %w", err)
	}
	if nil == virtualConfig.UpstreamProxies || commons.IsEmpty(*(virtualConfig.UpstreamProxies)) {
		log.Ctx(ctx).Debug().Msgf("Nothing to do for registryRequest: %s", dto.Identifier)
		return nil
	}

	upstreamProxies, err := c.RegistryRepository.FetchUpstreamProxyIDs(
		ctx,
		*virtualConfig.UpstreamProxies,
		parentID,
	)
	if err != nil {
		return fmt.Errorf("failed to fectch upstream proxy IDs :%w", err)
	}
	registry.UpstreamProxies = upstreamProxies
	return nil
}

func (c *APIController) getUpstreamProxyKeys(ctx context.Context, ids []int64) []string {
	repoKeys, _ := c.RegistryRepository.FetchUpstreamProxyKeys(ctx, ids)
	return repoKeys
}

type manifestConfig struct {
	CreatedAt  *string        `json:"created,omitempty"`
	Digest     string         `json:"digest,omitempty"`
	History    []historyEntry `json:"history"`
	ModifiedAt *string        `json:"modified,omitempty"`
	Os         string         `json:"os"`
	Arch       string         `json:"architecture,omitempty"`
}

type historyEntry struct {
	Created    string `json:"created"`
	CreatedBy  string `json:"created_by"`
	EmptyLayer bool   `json:"empty_layer"`
	Comment    string `json:"comment,omitempty"`
}

func getRepoEntityFields(dto api.RegistryRequest) ([]string, []string, string, []string) {
	allowedPattern := []string{}
	if dto.AllowedPattern != nil {
		allowedPattern = *dto.AllowedPattern
	}
	blockedPattern := []string{}
	if dto.BlockedPattern != nil {
		blockedPattern = *dto.BlockedPattern
	}
	description := ""
	if dto.Description != nil {
		description = *dto.Description
	}
	labels := []string{}
	if dto.Labels != nil {
		labels = *dto.Labels
	}
	return allowedPattern, blockedPattern, description, labels
}

func CreateVirtualRepositoryResponse(
	registry *types.Registry,
	upstreamProxyKeys []string,
	cleanupPolicies *[]types.CleanupPolicy,
	rootIdentifier string,
	registryURL string,
) *api.RegistryResponseJSONResponse {
	createdAt := GetTimeInMs(registry.CreatedAt)
	modifiedAt := GetTimeInMs(registry.UpdatedAt)
	allowedPattern := registry.AllowedPattern
	blockedPattern := registry.BlockedPattern
	labels := registry.Labels

	config := api.RegistryConfig{}
	_ = config.FromVirtualConfig(api.VirtualConfig{UpstreamProxies: &upstreamProxyKeys})
	response := &api.RegistryResponseJSONResponse{
		Data: api.Registry{
			Identifier:     registry.Name,
			Description:    &registry.Description,
			Url:            GetRepoURL(rootIdentifier, registry.Name, registryURL),
			PackageType:    registry.PackageType,
			AllowedPattern: &allowedPattern,
			BlockedPattern: &blockedPattern,
			CreatedAt:      &createdAt,
			ModifiedAt:     &modifiedAt,
			CleanupPolicy:  CreateCleanupPolicyResponse(cleanupPolicies),
			Config:         &config,
			Labels:         &labels,
		},
		Status: api.StatusSUCCESS,
	}
	return response
}

func CreateUpstreamProxyResponseJSONResponse(upstreamproxy *types.UpstreamProxy) *api.RegistryResponseJSONResponse {
	createdAt := GetTimeInMs(upstreamproxy.CreatedAt)
	modifiedAt := GetTimeInMs(upstreamproxy.UpdatedAt)
	allowedPattern := upstreamproxy.AllowedPattern
	blockedPattern := upstreamproxy.BlockedPattern
	configAuth := &api.UpstreamConfig_Auth{}

	if api.AuthType(upstreamproxy.RepoAuthType) == api.AuthTypeUserPassword {
		auth := api.UserPassword{}
		auth.UserName = upstreamproxy.UserName
		auth.SecretIdentifier = &upstreamproxy.SecretIdentifier.String
		auth.SecretSpaceId = nil
		if upstreamproxy.SecretSpaceID.Valid {
			// Convert int32 to int and assign to the expected field
			secretSpaceID := int(upstreamproxy.SecretSpaceID.Int32)
			auth.SecretSpaceId = &secretSpaceID
		}
		_ = configAuth.FromUserPassword(auth)
	}

	source := api.UpstreamConfigSource(upstreamproxy.Source)

	config := api.UpstreamConfig{
		AuthType: api.AuthType(upstreamproxy.RepoAuthType),
		Auth:     configAuth,
		Source:   &source,
		Url:      &upstreamproxy.RepoURL,
	}
	registryConfig := &api.RegistryConfig{}
	_ = registryConfig.FromUpstreamConfig(config)

	response := &api.RegistryResponseJSONResponse{
		Data: api.Registry{
			Identifier:     upstreamproxy.RepoKey,
			PackageType:    upstreamproxy.PackageType,
			Url:            upstreamproxy.RepoURL,
			AllowedPattern: &allowedPattern,
			BlockedPattern: &blockedPattern,
			CreatedAt:      &createdAt,
			ModifiedAt:     &modifiedAt,
			Config:         registryConfig,
		},
		Status: api.StatusSUCCESS,
	}
	return response
}
