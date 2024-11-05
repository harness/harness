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
	"strings"

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
	RootIdentifier   string
	rootIdentifierID int64

	RegistryRef        string
	RegistryIdentifier string
	RegistryID         int64

	ParentRef string
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
	registryIDs  []string
}

type RegistryRequestParams struct {
	packageTypesParam *api.PackageTypeParam
	page              *api.PageNumber
	size              *api.PageSize
	search            *api.SearchTerm
	Resource          string
	ParentRef         string
	RegRef            string
	labelsParam       *api.LabelsParam
	sortOrder         *api.SortOrder
	sortField         *api.SortField
	registryIDsParam  *api.RegistryIdentifierParam
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

	rootSpace, err := c.SpaceStore.FindByRef(ctx, rootIdentifier)
	if err != nil {
		return nil, fmt.Errorf("root space not found: %w", err)
	}
	parentSpace, err := c.SpaceStore.FindByRef(ctx, parentRef)
	if err != nil {
		return nil, fmt.Errorf("parent space not found: %w", err)
	}
	rootIdentifierID := rootSpace.ID
	parentID := parentSpace.ID

	baseInfo := &RegistryRequestBaseInfo{
		ParentRef:        parentRef,
		parentID:         parentID,
		RootIdentifier:   rootIdentifier,
		rootIdentifierID: rootIdentifierID,
	}

	// ---------- REGISTRY  ------------
	if !commons.IsEmpty(regRef) {
		_, regIdentifier, _ := paths.DisectLeaf(regRef)

		reg, getRegistryErr := c.RegistryRepository.GetByParentIDAndName(ctx, parentID, regIdentifier)
		if getRegistryErr != nil {
			return nil, fmt.Errorf("registry not found: %w", err)
		}

		baseInfo.RegistryRef = regRef
		baseInfo.RegistryIdentifier = regIdentifier
		baseInfo.RegistryID = reg.ID
	}

	return baseInfo, nil
}

func (c *APIController) GetRegistryRequestInfo(
	ctx context.Context,
	registryRequestParams RegistryRequestParams,
) (*RegistryRequestInfo, error) {
	packageTypes := []string{}
	if registryRequestParams.packageTypesParam != nil {
		packageTypes = *registryRequestParams.packageTypesParam
	}
	registryIDs := []string{}
	if registryRequestParams.registryIDsParam != nil {
		registryIDs = *registryRequestParams.registryIDsParam
	}
	sortByField := ""
	sortByOrder := ""
	if registryRequestParams.sortOrder != nil {
		sortByOrder = string(*registryRequestParams.sortOrder)
	}

	if registryRequestParams.sortField != nil {
		sortByField = string(*registryRequestParams.sortField)
	}

	labels := []string{}

	if registryRequestParams.labelsParam != nil {
		labels = *registryRequestParams.labelsParam
	}

	sortByField = GetSortByField(sortByField, registryRequestParams.Resource)
	sortByOrder = GetSortByOrder(sortByOrder)

	offset := GetOffset(registryRequestParams.size, registryRequestParams.page)
	limit := GetPageLimit(registryRequestParams.size)
	pageNumber := GetPageNumber(registryRequestParams.page)

	searchTerm := ""
	if registryRequestParams.search != nil {
		searchTerm = string(*registryRequestParams.search)
	}

	baseInfo, err := c.GetRegistryRequestBaseInfo(ctx, registryRequestParams.ParentRef, registryRequestParams.RegRef)
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
		registryIDs:             registryIDs,
	}, nil
}

func getManifestConfig(
	ctx context.Context,
	digest digest.Digest,
	rootRef string,
	driver storagedriver.StorageDriver,
) (*manifestConfig, error) {
	var config manifestConfig
	path, err := storage.PathFn(strings.ToLower(rootRef), digest)
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
	RootFS     rootFS         `json:"rootfs,omitempty"`
}

type historyEntry struct {
	Created    string `json:"created"`
	CreatedBy  string `json:"created_by"`
	EmptyLayer bool   `json:"empty_layer"`
	Comment    string `json:"comment,omitempty"`
}

type rootFS struct {
	RootFsType string   `json:"type"`
	DiffIDs    []string `json:"diff_ids"`
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
			Url:            registryURL,
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
		auth.SecretIdentifier = &upstreamproxy.SecretIdentifier
		auth.SecretSpacePath = &upstreamproxy.SecretSpacePath
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
