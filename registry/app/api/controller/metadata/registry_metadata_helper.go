// Copyright 2023 Harness, Inc.
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
	"strconv"

	"github.com/harness/gitness/app/paths"
	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/api/interfaces"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ interfaces.RegistryMetadataHelper = (*GitnessRegistryMetadataHelper)(nil)

type GitnessRegistryMetadataHelper struct {
	spacePathStore     corestore.SpacePathStore
	spaceFinder        interfaces.SpaceFinder
	registryRepository store.RegistryRepository
}

func NewRegistryMetadataHelper(
	spacePathStore corestore.SpacePathStore,
	spaceFinder interfaces.SpaceFinder,
	registryRepository store.RegistryRepository,
) interfaces.RegistryMetadataHelper {
	gitnessRegistryMetadataHelper := GitnessRegistryMetadataHelper{
		spacePathStore:     spacePathStore,
		spaceFinder:        spaceFinder,
		registryRepository: registryRepository,
	}
	return &gitnessRegistryMetadataHelper
}

func (r *GitnessRegistryMetadataHelper) GetSecretSpaceID(
	ctx context.Context,
	secretSpacePath *string,
) (int64, error) {
	if secretSpacePath == nil {
		return -1, fmt.Errorf("secret space path is missing")
	}

	path, err := r.spacePathStore.FindByPath(ctx, *secretSpacePath)
	if err != nil {
		return -1, fmt.Errorf("failed to get Space Path: %w", err)
	}
	return path.SpaceID, nil
}

// GetRegistryRequestBaseInfo returns the base info for the registry request
// One of the regRefParam or (parentRefParam + regIdentifierParam) should be provided.
func (r *GitnessRegistryMetadataHelper) GetRegistryRequestBaseInfo(
	ctx context.Context,
	parentRef string,
	regRef string,
) (*registrytypes.RegistryRequestBaseInfo, error) {
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

	rootSpace, err := r.spaceFinder.FindByRef(ctx, rootIdentifier)
	if err != nil {
		return nil, fmt.Errorf("root space not found: %w", err)
	}
	parentSpace, err := r.spaceFinder.FindByRef(ctx, parentRef)
	if err != nil {
		return nil, fmt.Errorf("parent space not found: %w", err)
	}
	rootIdentifierID := rootSpace.ID
	parentID := parentSpace.ID

	baseInfo := registrytypes.RegistryRequestBaseInfo{
		ParentRef:        parentRef,
		ParentID:         parentID,
		RootIdentifier:   rootIdentifier,
		RootIdentifierID: rootIdentifierID,
	}

	// ---------- REGISTRY  ------------
	if !commons.IsEmpty(regRef) {
		_, regIdentifier, _ := paths.DisectLeaf(regRef)

		reg, getRegistryErr := r.registryRepository.GetByParentIDAndName(ctx, parentID, regIdentifier)
		if getRegistryErr != nil {
			return nil, fmt.Errorf("registry not found: %w", getRegistryErr)
		}

		baseInfo.RegistryRef = regRef
		baseInfo.RegistryIdentifier = regIdentifier
		baseInfo.RegistryID = reg.ID
		baseInfo.RegistryType = reg.Type
		baseInfo.PackageType = reg.PackageType
	}

	return &baseInfo, nil
}

func (r *GitnessRegistryMetadataHelper) GetPermissionChecks(
	space *types.SpaceCore,
	registryIdentifier string,
	permission enum.Permission,
) []types.PermissionCheck {
	var permissionChecks []types.PermissionCheck
	permissionCheck := &types.PermissionCheck{
		Scope:      types.Scope{SpacePath: space.Path},
		Resource:   types.Resource{Type: enum.ResourceTypeRegistry, Identifier: registryIdentifier},
		Permission: permission,
	}
	permissionChecks = append(permissionChecks, *permissionCheck)
	return permissionChecks
}

func (r *GitnessRegistryMetadataHelper) MapToWebhookCore(
	ctx context.Context,
	webhookRequest api.WebhookRequest,
	regInfo *registrytypes.RegistryRequestBaseInfo,
) (*types.WebhookCore, error) {
	webhook := &types.WebhookCore{
		DisplayName: webhookRequest.Name,
		ParentType:  enum.WebhookParentRegistry,
		ParentID:    regInfo.RegistryID,
		Scope:       webhookScopeRegistry,
		Identifier:  webhookRequest.Identifier,
		URL:         webhookRequest.Url,
		Enabled:     webhookRequest.Enabled,
		Insecure:    webhookRequest.Insecure,
	}

	if webhookRequest.Triggers != nil {
		triggers, err := r.MapToInternalWebhookTriggers(*webhookRequest.Triggers)
		if err != nil {
			return nil, fmt.Errorf("failed to map to internal webhook triggers: %w", err)
		}
		webhook.Triggers = deduplicateTriggers(triggers)
	}

	if webhookRequest.Description != nil {
		webhook.Description = *webhookRequest.Description
	}
	if webhookRequest.SecretIdentifier != nil {
		webhook.SecretIdentifier = *webhookRequest.SecretIdentifier
	}
	if webhookRequest.SecretSpacePath != nil && len(*webhookRequest.SecretSpacePath) > 0 {
		secretSpaceID, err := r.GetSecretSpaceID(ctx, webhookRequest.SecretSpacePath)
		if err != nil {
			return nil, err
		}
		webhook.SecretSpaceID = secretSpaceID
	} else if webhookRequest.SecretSpaceId != nil {
		webhook.SecretSpaceID = *webhookRequest.SecretSpaceId
	}
	if webhookRequest.ExtraHeaders != nil {
		webhook.ExtraHeaders = mapToDTOHeaders(webhookRequest.ExtraHeaders)
	}

	return webhook, nil
}

func mapToDTOHeaders(extraHeaders *[]api.ExtraHeader) []types.ExtraHeader {
	var headers []types.ExtraHeader
	for _, h := range *extraHeaders {
		masked := false
		if h.Masked != nil {
			masked = *h.Masked
		}
		headers = append(headers, types.ExtraHeader{Key: h.Key, Value: h.Value, Masked: masked})
	}
	return headers
}

func (r *GitnessRegistryMetadataHelper) MapToWebhookResponseEntity(
	ctx context.Context,
	createdWebhook *types.WebhookCore,
) (*api.Webhook, error) {
	createdAt := strconv.FormatInt(createdWebhook.Created, 10)
	modifiedAt := strconv.FormatInt(createdWebhook.Updated, 10)
	triggers := r.MapToAPIWebhookTriggers(createdWebhook.Triggers)

	webhookResponseEntity := &api.Webhook{
		Identifier:  createdWebhook.Identifier,
		Name:        createdWebhook.DisplayName,
		Description: &createdWebhook.Description,
		Url:         createdWebhook.URL,
		Version:     &createdWebhook.Version,
		Enabled:     createdWebhook.Enabled,
		Insecure:    createdWebhook.Insecure,
		Triggers:    &triggers,
		CreatedBy:   &createdWebhook.CreatedBy,
		CreatedAt:   &createdAt,
		ModifiedAt:  &modifiedAt,
	}
	isInternal := false
	if createdWebhook.Type == enum.WebhookTypeInternal {
		isInternal = true
	} else {
		isInternal = false
	}
	webhookResponseEntity.Internal = &isInternal

	if createdWebhook.LatestExecutionResult != nil {
		result := r.MapToAPIExecutionResult(*createdWebhook.LatestExecutionResult)
		webhookResponseEntity.LatestExecutionResult = &result
	}

	if createdWebhook.ExtraHeaders != nil {
		extraHeaders := r.MapToAPIExtraHeaders(createdWebhook.ExtraHeaders)
		webhookResponseEntity.ExtraHeaders = &extraHeaders
	}
	secretSpacePath := ""
	if createdWebhook.SecretSpaceID > 0 {
		primary, err := r.spacePathStore.FindPrimaryBySpaceID(ctx, createdWebhook.SecretSpaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret space path: %w", err)
		}
		secretSpacePath = primary.Value
	}
	if createdWebhook.SecretIdentifier != "" {
		webhookResponseEntity.SecretIdentifier = &createdWebhook.SecretIdentifier
	}
	if secretSpacePath != "" {
		webhookResponseEntity.SecretSpacePath = &secretSpacePath
	}
	if createdWebhook.SecretSpaceID > 0 {
		webhookResponseEntity.SecretSpaceId = &createdWebhook.SecretSpaceID
	}

	return webhookResponseEntity, nil
}

func (r *GitnessRegistryMetadataHelper) MapToInternalWebhookTriggers(
	triggers []api.Trigger,
) ([]enum.WebhookTrigger, error) {
	var webhookTriggers = make([]enum.WebhookTrigger, 0)
	var invalidTriggers []string

	for _, trigger := range triggers {
		switch trigger {
		case api.TriggerARTIFACTCREATION:
			webhookTriggers = append(webhookTriggers, enum.WebhookTriggerArtifactCreated)
		case api.TriggerARTIFACTDELETION:
			webhookTriggers = append(webhookTriggers, enum.WebhookTriggerArtifactDeleted)
		default:
			invalidTriggers = append(invalidTriggers, string(trigger))
		}
	}

	if len(invalidTriggers) > 0 {
		return nil, fmt.Errorf("invalid webhook triggers: %v", invalidTriggers)
	}

	return webhookTriggers, nil
}

func (r *GitnessRegistryMetadataHelper) MapToAPIExecutionResult(
	result enum.WebhookExecutionResult,
) api.WebhookExecResult {
	switch result {
	case enum.WebhookExecutionResultSuccess:
		return api.WebhookExecResultSUCCESS
	case enum.WebhookExecutionResultRetriableError:
		return api.WebhookExecResultRETRIABLEERROR
	case enum.WebhookExecutionResultFatalError:
		return api.WebhookExecResultFATALERROR
	}

	return ""
}

func (r *GitnessRegistryMetadataHelper) MapToAPIWebhookTriggers(triggers []enum.WebhookTrigger) []api.Trigger {
	var webhookTriggers = make([]api.Trigger, 0)
	for _, trigger := range triggers {
		//nolint:exhaustive
		switch trigger {
		case enum.WebhookTriggerArtifactCreated:
			webhookTriggers = append(webhookTriggers, api.TriggerARTIFACTCREATION)
		case enum.WebhookTriggerArtifactDeleted:
			webhookTriggers = append(webhookTriggers, api.TriggerARTIFACTDELETION)
		}
	}
	return webhookTriggers
}

func (r *GitnessRegistryMetadataHelper) MapToAPIExtraHeaders(headers []types.ExtraHeader) []api.ExtraHeader {
	apiHeaders := make([]api.ExtraHeader, 0)
	for _, h := range headers {
		masked := h.Masked
		apiHeaders = append(apiHeaders, api.ExtraHeader{Key: h.Key, Value: h.Value, Masked: &masked})
	}
	return apiHeaders
}
