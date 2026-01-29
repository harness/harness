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
	"math"
	"strconv"
	"strings"

	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/types"
	gitnessenum "github.com/harness/gitness/types/enum"

	digest "github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

var _ api.StrictServerInterface = (*APIController)(nil)

type RegistryRequestInfo struct {
	*types.RegistryRequestBaseInfo
	packageTypes []string
	sortByField  string
	sortByOrder  string
	offset       int
	limit        int
	pageNumber   int64
	searchTerm   string
	labels       []string
	registryIDs  []string
	recursive    bool
	scope        string
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
	recursive         *api.RecursiveParam
	scope             *api.GetAllRegistriesParamsScope
}

type ArtifactFilesRequestInfo struct {
	*types.RegistryRequestBaseInfo
	sortByField string
	sortByOrder string
	offset      int
	limit       int
	pageNumber  int64
	searchTerm  string
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

	baseInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, registryRequestParams.ParentRef,
		registryRequestParams.RegRef)
	if err != nil {
		return nil, err
	}

	scope := ""
	if registryRequestParams.scope != nil {
		scope = string(*registryRequestParams.scope)
	}

	recursive := false
	if registryRequestParams.recursive != nil {
		recursive = bool(*registryRequestParams.recursive)
	}

	return &RegistryRequestInfo{
		RegistryRequestBaseInfo: baseInfo,
		packageTypes:            packageTypes,
		sortByField:             sortByField,
		sortByOrder:             sortByOrder,
		offset:                  offset,
		limit:                   limit,
		pageNumber:              pageNumber,
		searchTerm:              searchTerm,
		labels:                  labels,
		registryIDs:             registryIDs,
		recursive:               recursive,
		scope:                   scope,
	}, nil
}

// TODO(Arvind) - route via Local.go of docker instead of directly working with storage layer
func (c *APIController) getManifestConfig(
	ctx context.Context,
	digest digest.Digest,
	info *types.RegistryRequestBaseInfo,
) (*manifestConfig, error) {
	blobsContext := c.app.GetBlobsContext(ctx, pkg.RegistryInfo{
		ArtifactInfo: &pkg.ArtifactInfo{
			RegIdentifier: info.RegistryIdentifier,
			BaseInfo: &pkg.BaseInfo{
				RootIdentifier: info.RootIdentifier,
			},
		},
		Digest: digest.String(),
	}, types.BlobLocator{
		Digest:       digest,
		RegistryID:   info.RegistryID,
		RootParentID: info.RootIdentifierID,
	})
	var config manifestConfig

	content, err := blobsContext.OciBlobStore.Get(ctx, strings.ToLower(info.RootIdentifier), digest)
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

	parentIDs, err := c.SpaceStore.GetAncestorIDs(ctx, parentID)
	if err != nil {
		return fmt.Errorf("failed to get ancestors upstream proxies: %w", err)
	}
	repos, err := c.RegistryRepository.GetAll(
		ctx,
		parentIDs,
		[]string{string(registry.PackageType)},
		"id",
		"",
		math.MaxInt,
		0,
		"",
		"",
	)

	if repos == nil || err != nil {
		err := fmt.Errorf("no repositories found for parentID: %d", parentID)
		log.Ctx(ctx).Debug().Err(err).Msg("Failed to fetch repositories")
		return err
	}

	var upstreamProxies []int64
	for _, proxy := range *virtualConfig.UpstreamProxies {
		for _, repo := range *repos {
			if repo.RegIdentifier == proxy {
				regID, err := strconv.ParseInt(repo.RegID, 10, 64)
				if err != nil {
					continue
				}
				// cycle detection: ensure adding regID as upstream to registry.ID doesn't create a cycle
				if err := c.assertNoCycleOnAdd(ctx, registry.ID, regID, registry.Name); err != nil {
					return err
				}
				upstreamProxies = append(upstreamProxies, regID)
			}
		}
	}

	registry.UpstreamProxies = upstreamProxies
	return nil
}

func (c *APIController) assertNoCycleOnAdd(
	ctx context.Context,
	registryID int64, newUpstreamID int64, registryName string,
) error {
	return c.assertNoCycleOnAddHelper(ctx, registryID, newUpstreamID, registryName)
}

func (c *APIController) assertNoCycleOnAddHelper(
	ctx context.Context,
	rootRegistryID int64, newUpstreamID int64, registryName string,
) error {
	if newUpstreamID == rootRegistryID {
		return fmt.Errorf("cycle detected while setting up the upstream proxy for registry: [%s]", registryName)
	}
	upstreamProxies, err := c.getUpstreamProxyIDs(ctx, newUpstreamID)
	if err != nil {
		return err
	}

	for _, upstreamProxyID := range upstreamProxies {
		if err := c.assertNoCycleOnAddHelper(ctx, rootRegistryID, upstreamProxyID, registryName); err != nil {
			return err
		}
	}
	return nil
}

func (c *APIController) getUpstreamProxyIDs(ctx context.Context, registryID int64) ([]int64, error) {
	registry, err := c.RegistryRepository.Get(ctx, registryID)
	if err != nil {
		return nil, err
	}
	return registry.UpstreamProxies, nil
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
	RootFS     rootFS         `json:"rootfs"`
}

type historyEntry struct {
	Created    string `json:"created"`
	CreatedBy  string `json:"created_by"`  //nolint:tagliatelle
	EmptyLayer bool   `json:"empty_layer"` //nolint:tagliatelle
	Comment    string `json:"comment,omitempty"`
}

type rootFS struct {
	RootFsType string   `json:"type"`
	DiffIDs    []string `json:"diff_ids"` //nolint:tagliatelle
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

func (c *APIController) CreateVirtualRepositoryResponse(
	ctx context.Context,
	registry *types.Registry,
	upstreamProxyKeys []string,
	cleanupPolicies *[]types.CleanupPolicy,
	registryURL string,
	registryRef string,
) (*api.RegistryResponseJSONResponse, error) {
	isPublic, err := c.PublicAccess.Get(ctx, gitnessenum.PublicResourceTypeRegistry, registryRef)
	if err != nil {
		return nil, fmt.Errorf("error in fetching public access for registry %s: %w", registryRef, err)
	}
	createdAt := GetTimeInMs(registry.CreatedAt)
	modifiedAt := GetTimeInMs(registry.UpdatedAt)
	allowedPattern := registry.AllowedPattern
	blockedPattern := registry.BlockedPattern
	labels := registry.Labels

	config := api.RegistryConfig{}
	_ = config.FromVirtualConfig(api.VirtualConfig{UpstreamProxies: &upstreamProxyKeys})
	response := &api.RegistryResponseJSONResponse{
		Data: api.Registry{
			Uuid:           registry.UUID,
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
			IsPublic:       isPublic,
		},
		Status: api.StatusSUCCESS,
	}
	return response, nil
}

func (c *APIController) CreateUpstreamProxyResponseJSONResponse(
	ctx context.Context,
	upstreamproxy *types.UpstreamProxy,
	registryRef string,
) (*api.RegistryResponseJSONResponse, error) {
	isPublic, err := c.PublicAccess.Get(ctx, gitnessenum.PublicResourceTypeRegistry, registryRef)
	if err != nil {
		return nil, fmt.Errorf("error in fetching public access for registry %s: %w", registryRef, err)
	}
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
	} else if api.AuthType(upstreamproxy.RepoAuthType) == api.AuthTypeAccessKeySecretKey {
		auth := api.AccessKeySecretKey{}
		auth.AccessKey = &upstreamproxy.UserName
		auth.AccessKeySecretIdentifier = &upstreamproxy.UserNameSecretIdentifier
		auth.AccessKeySecretSpacePath = &upstreamproxy.UserNameSecretSpacePath
		auth.SecretKeyIdentifier = upstreamproxy.SecretIdentifier
		auth.SecretKeySpacePath = &upstreamproxy.SecretSpacePath
		err := configAuth.FromAccessKeySecretKey(auth)
		if err != nil {
			return nil, fmt.Errorf("error in converting auth config to access and secret key: %w", err)
		}
	}

	source := api.UpstreamConfigSource(upstreamproxy.Source)

	config := api.UpstreamConfig{
		AuthType: api.AuthType(upstreamproxy.RepoAuthType),
		Auth:     configAuth,
		Source:   &source,
		Url:      &upstreamproxy.RepoURL,
	}
	if upstreamproxy.FirewallMode != nil {
		config.UpstreamProxyConfigFirewallMode = (*api.UpstreamProxyConfigFirewallMode)(upstreamproxy.FirewallMode)
	}

	if upstreamproxy.Config != nil && upstreamproxy.Config.RemoteUrlSuffix != "" {
		config.RemoteUrlSuffix = &upstreamproxy.Config.RemoteUrlSuffix
	}

	registryConfig := &api.RegistryConfig{}
	_ = registryConfig.FromUpstreamConfig(config)

	response := &api.RegistryResponseJSONResponse{
		Data: api.Registry{
			Uuid:           upstreamproxy.RegistryUUID,
			Identifier:     upstreamproxy.RepoKey,
			PackageType:    upstreamproxy.PackageType,
			Url:            upstreamproxy.RepoURL,
			AllowedPattern: &allowedPattern,
			BlockedPattern: &blockedPattern,
			CreatedAt:      &createdAt,
			ModifiedAt:     &modifiedAt,
			Config:         registryConfig,
			IsPublic:       isPublic,
		},
		Status: api.StatusSUCCESS,
	}
	return response, nil
}

// deduplicateTriggers de-duplicates the triggers provided by the user.
func deduplicateTriggers(in []gitnessenum.WebhookTrigger) []gitnessenum.WebhookTrigger {
	if len(in) == 0 {
		return []gitnessenum.WebhookTrigger{}
	}

	triggerSet := make(map[gitnessenum.WebhookTrigger]bool, len(in))
	out := make([]gitnessenum.WebhookTrigger, 0, len(in))
	for _, trigger := range in {
		if triggerSet[trigger] {
			continue
		}
		triggerSet[trigger] = true
		out = append(out, trigger)
	}

	return out
}

func (c *APIController) GetArtifactFilesRequestInfo(
	ctx context.Context,
	r api.GetArtifactFilesRequestObject,
) (*ArtifactFilesRequestInfo, error) {
	sortByField := ""
	sortByOrder := ""
	if r.Params.SortOrder != nil {
		sortByOrder = string(*r.Params.SortOrder)
	}

	if r.Params.SortField != nil {
		sortByField = string(*r.Params.SortField)
	}

	sortByField = GetSortByField(sortByField, ArtifactFilesResource)
	sortByOrder = GetSortByOrder(sortByOrder)

	offset := GetOffset(r.Params.Size, r.Params.Page)
	limit := GetPageLimit(r.Params.Size)
	pageNumber := GetPageNumber(r.Params.Page)

	searchTerm := ""
	if r.Params.SearchTerm != nil {
		searchTerm = string(*r.Params.SearchTerm)
	}

	baseInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))

	if err != nil {
		return nil, err
	}

	return &ArtifactFilesRequestInfo{
		RegistryRequestBaseInfo: baseInfo,
		sortByField:             sortByField,
		sortByOrder:             sortByOrder,
		offset:                  offset,
		limit:                   limit,
		pageNumber:              pageNumber,
		searchTerm:              searchTerm,
	}, nil
}
