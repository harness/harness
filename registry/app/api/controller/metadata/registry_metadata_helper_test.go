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
	"strconv"
	"testing"
	"time"

	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	gitnesstypes "github.com/harness/gitness/types"
	gitnessenum "github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
)

//nolint:lll
func TestGetRegistryRequestBaseInfo(t *testing.T) {
	ctx := context.Background()
	mockSpacePathStore := new(MockSpacePathStore)
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	helper := &GitnessRegistryMetadataHelper{
		spacePathStore:     mockSpacePathStore,
		spaceFinder:        mockSpaceFinder,
		registryRepository: mockRegistryRepository,
	}

	mockSpaceFinder.On("FindByRef", ctx, "root").Return(&gitnesstypes.SpaceCore{ID: 1}, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(&gitnesstypes.SpaceCore{ID: 2}, nil)
	mockRegistryRepository.On("GetByParentIDAndName", ctx, int64(2), "reg").Return(&types.Registry{ID: 3, Type: api.RegistryTypeVIRTUAL}, nil)

	baseInfo, err := helper.GetRegistryRequestBaseInfo(ctx, "root/parent", "reg")
	assert.NoError(t, err)
	assert.NotNil(t, baseInfo)
	assert.Equal(t, int64(1), baseInfo.rootIdentifierID)
	assert.Equal(t, int64(2), baseInfo.parentID)
	assert.Equal(t, int64(3), baseInfo.RegistryID)
	assert.Equal(t, api.RegistryTypeVIRTUAL, baseInfo.RegistryType)
}

func TestGetRegistryRequestBaseInfo_EmptyParentRef(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	helper := &GitnessRegistryMetadataHelper{
		spaceFinder:        mockSpaceFinder,
		registryRepository: mockRegistryRepository,
	}

	_, err := helper.GetRegistryRequestBaseInfo(ctx, "", "reg")
	assert.Error(t, err)
	assert.Equal(t, "parent reference is required", err.Error())
}

//nolint:lll
func TestGetRegistryRequestBaseInfo_InvalidParentRef(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	helper := &GitnessRegistryMetadataHelper{
		spaceFinder:        mockSpaceFinder,
		registryRepository: mockRegistryRepository,
	}

	_, err := helper.GetRegistryRequestBaseInfo(ctx, "/", "reg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid parent reference")
}

func TestGetRegistryRequestBaseInfo_RootSpaceNotFound(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	helper := &GitnessRegistryMetadataHelper{
		spaceFinder:        mockSpaceFinder,
		registryRepository: mockRegistryRepository,
	}

	mockSpaceFinder.On("FindByRef", ctx, "root").Return(nil, fmt.Errorf("not found"))

	_, err := helper.GetRegistryRequestBaseInfo(ctx, "root/parent", "reg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "root space not found")
}

func TestGetRegistryRequestBaseInfo_ParentSpaceNotFound(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	helper := &GitnessRegistryMetadataHelper{
		spaceFinder:        mockSpaceFinder,
		registryRepository: mockRegistryRepository,
	}

	mockSpaceFinder.On("FindByRef", ctx, "root").Return(&gitnesstypes.SpaceCore{ID: 1}, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(nil, fmt.Errorf("not found"))

	_, err := helper.GetRegistryRequestBaseInfo(ctx, "root/parent", "reg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parent space not found")
}

func TestGetRegistryRequestBaseInfo_RegistryNotFound(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	helper := &GitnessRegistryMetadataHelper{
		spaceFinder:        mockSpaceFinder,
		registryRepository: mockRegistryRepository,
	}

	mockSpaceFinder.On("FindByRef", ctx, "root").Return(&gitnesstypes.SpaceCore{ID: 1}, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(&gitnesstypes.SpaceCore{ID: 2}, nil)
	mockRegistryRepository.On("GetByParentIDAndName", ctx, int64(2), "reg").Return(nil, fmt.Errorf("not found"))

	_, err := helper.GetRegistryRequestBaseInfo(ctx, "root/parent", "reg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry not found")
}

func TestGetPermissionChecks(t *testing.T) {
	helper := &GitnessRegistryMetadataHelper{}
	space := &gitnesstypes.SpaceCore{Path: "space/path"}
	permissionChecks := helper.GetPermissionChecks(space, "registry", gitnessenum.PermissionRegistryEdit)

	assert.Len(t, permissionChecks, 1)
	assert.Equal(t, "space/path", permissionChecks[0].Scope.SpacePath)
	assert.Equal(t, "registry", permissionChecks[0].Resource.Identifier)
	assert.Equal(t, gitnessenum.PermissionRegistryEdit, permissionChecks[0].Permission)
}

//nolint:goconst
func TestMapToWebhook(t *testing.T) {
	ctx := context.Background()
	mockSpacePathStore := new(MockSpacePathStore)
	helper := &GitnessRegistryMetadataHelper{
		spacePathStore: mockSpacePathStore,
	}

	key := "key"
	value := "value"
	extraHeaders := []api.ExtraHeader{{Key: key, Value: value}}
	description := "Test webhook"
	webhookRequest := api.WebhookRequest{
		Identifier:   "webhook",
		Url:          "http://example.com",
		Enabled:      true,
		Insecure:     false,
		Triggers:     &[]api.Trigger{api.TriggerARTIFACTCREATION, api.TriggerARTIFACTCREATION, api.TriggerARTIFACTDELETION},
		Description:  &description,
		ExtraHeaders: &extraHeaders,
	}
	regInfo := &RegistryRequestBaseInfo{
		RegistryID: 1,
	}

	webhook, err := helper.MapToWebhookCore(ctx, webhookRequest, regInfo)
	assert.NoError(t, err)
	assert.NotNil(t, webhook)
	assert.Equal(t, "webhook", webhook.Identifier)
	assert.Equal(t, "http://example.com", webhook.URL)
	assert.Equal(t, true, webhook.Enabled)
	assert.Equal(t, false, webhook.Insecure)
	assert.Len(t, webhook.Triggers, 2)
	assert.Contains(t, webhook.Triggers, gitnessenum.WebhookTriggerArtifactCreated)
	assert.Contains(t, webhook.Triggers, gitnessenum.WebhookTriggerArtifactDeleted)
	assert.Equal(t, "Test webhook", webhook.Description)
	assert.Equal(t, extraHeaders[0].Key, key)
	assert.Equal(t, extraHeaders[0].Value, value)
}

//nolint:lll
func TestMapToWebhook_WithSecretSpacePath(t *testing.T) {
	ctx := context.Background()
	mockSpacePathStore := new(MockSpacePathStore)
	helper := &GitnessRegistryMetadataHelper{
		spacePathStore: mockSpacePathStore,
	}

	secretSpacePath := "secret/path"
	webhookRequest := api.WebhookRequest{
		Identifier:      "webhook",
		Url:             "http://example.com",
		Enabled:         true,
		Insecure:        false,
		Triggers:        &[]api.Trigger{api.TriggerARTIFACTCREATION},
		SecretSpacePath: &secretSpacePath,
	}
	regInfo := &RegistryRequestBaseInfo{
		RegistryID: 1,
	}

	mockSpacePathStore.On("FindByPath", ctx, "secret/path").Return(&gitnesstypes.SpacePath{Value: "secret/path", SpaceID: 2}, nil)

	webhook, err := helper.MapToWebhookCore(ctx, webhookRequest, regInfo)
	assert.NoError(t, err)
	assert.NotNil(t, webhook)
	assert.Equal(t, "webhook", webhook.Identifier)
	assert.Equal(t, "http://example.com", webhook.URL)
	assert.Equal(t, true, webhook.Enabled)
	assert.Equal(t, false, webhook.Insecure)
	assert.Len(t, webhook.Triggers, 1)
	assert.Equal(t, gitnessenum.WebhookTriggerArtifactCreated, webhook.Triggers[0])
	assert.Equal(t, int64(2), webhook.SecretSpaceID)
}

func TestMapToWebhook_WithInexistentSecretSpacePath(t *testing.T) {
	ctx := context.Background()
	mockSpacePathStore := new(MockSpacePathStore)
	helper := &GitnessRegistryMetadataHelper{
		spacePathStore: mockSpacePathStore,
	}

	secretSpacePath := "secret/path"
	webhookRequest := api.WebhookRequest{
		Identifier:      "webhook",
		Url:             "http://example.com",
		Enabled:         true,
		Insecure:        false,
		Triggers:        &[]api.Trigger{api.TriggerARTIFACTCREATION},
		SecretSpacePath: &secretSpacePath,
	}
	regInfo := &RegistryRequestBaseInfo{
		RegistryID: 1,
	}

	mockSpacePathStore.On("FindByPath", ctx, "secret/path").Return(nil, fmt.Errorf("not found"))

	webhook, err := helper.MapToWebhookCore(ctx, webhookRequest, regInfo)
	assert.Nil(t, webhook)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Space Path: not found")
}

func TestMapToWebhookResponseEntity(t *testing.T) {
	ctx := context.Background()
	mockSpacePathStore := new(MockSpacePathStore)
	helper := &GitnessRegistryMetadataHelper{
		spacePathStore: mockSpacePathStore,
	}

	createdAt := time.Now().Unix()
	updatedAt := time.Now().Unix()
	description := "Test webhook"
	secretIdentifier := "secret-id"
	extraHeaders := []gitnesstypes.ExtraHeader{{Key: "key", Value: "value"}}
	latestExecutionResult := gitnessenum.WebhookExecutionResultSuccess
	createdWebhook := gitnesstypes.WebhookCore{
		Type:                  gitnessenum.WebhookTypeInternal,
		Identifier:            "webhook",
		DisplayName:           "webhook",
		URL:                   "http://example.com",
		Enabled:               true,
		Insecure:              false,
		Triggers:              []gitnessenum.WebhookTrigger{gitnessenum.WebhookTriggerArtifactCreated},
		Created:               createdAt,
		Updated:               updatedAt,
		Description:           description,
		SecretIdentifier:      secretIdentifier,
		SecretSpaceID:         1,
		ExtraHeaders:          extraHeaders,
		LatestExecutionResult: &latestExecutionResult,
	}

	mockSpacePathStore.On("FindPrimaryBySpaceID", ctx, int64(1)).Return(&gitnesstypes.SpacePath{Value: "secret/path"}, nil)

	webhookResponseEntity, err := helper.MapToWebhookResponseEntity(ctx, &createdWebhook)
	assert.NoError(t, err)
	assert.NotNil(t, webhookResponseEntity)
	assert.Equal(t, "webhook", webhookResponseEntity.Identifier)
	assert.Equal(t, "webhook", webhookResponseEntity.Name)
	assert.Equal(t, "http://example.com", webhookResponseEntity.Url)
	assert.Equal(t, true, webhookResponseEntity.Enabled)
	assert.Equal(t, false, webhookResponseEntity.Insecure)
	assert.Len(t, *webhookResponseEntity.Triggers, 1)
	assert.Equal(t, api.TriggerARTIFACTCREATION, (*webhookResponseEntity.Triggers)[0])
	assert.Equal(t, &description, webhookResponseEntity.Description)
	assert.Equal(t, &createdWebhook.Version, webhookResponseEntity.Version)
	assert.True(t, *webhookResponseEntity.Internal)
	assert.Equal(t, &createdWebhook.CreatedBy, webhookResponseEntity.CreatedBy)
	assert.Equal(t, strconv.FormatInt(createdAt, 10), *webhookResponseEntity.CreatedAt)
	assert.Equal(t, strconv.FormatInt(updatedAt, 10), *webhookResponseEntity.ModifiedAt)
	assert.Equal(t, api.WebhookExecResultSUCCESS, *webhookResponseEntity.LatestExecutionResult)
	assert.Equal(t, "key", extraHeaders[0].Key)
	assert.Equal(t, "value", extraHeaders[0].Value)
	assert.Equal(t, "secret-id", *webhookResponseEntity.SecretIdentifier)
	assert.Equal(t, "secret/path", *webhookResponseEntity.SecretSpacePath)
	assert.Equal(t, int64(1), *webhookResponseEntity.SecretSpaceId)
}

//nolint:lll
func TestMapToWebhookResponseEntity_FindPrimaryBySpaceIDError(t *testing.T) {
	ctx := context.Background()
	mockSpacePathStore := new(MockSpacePathStore)
	helper := &GitnessRegistryMetadataHelper{
		spacePathStore: mockSpacePathStore,
	}

	createdAt := time.Now().Unix()
	updatedAt := time.Now().Unix()
	description := "Test webhook"
	secretIdentifier := "secret-id"
	extraHeaders := []gitnesstypes.ExtraHeader{{Key: "key", Value: "value"}}
	latestExecutionResult := gitnessenum.WebhookExecutionResultSuccess
	createdWebhook := gitnesstypes.WebhookCore{
		Identifier:            "webhook",
		DisplayName:           "webhook",
		URL:                   "http://example.com",
		Enabled:               true,
		Insecure:              false,
		Triggers:              []gitnessenum.WebhookTrigger{gitnessenum.WebhookTriggerArtifactCreated},
		Created:               createdAt,
		Updated:               updatedAt,
		Description:           description,
		SecretIdentifier:      secretIdentifier,
		SecretSpaceID:         1,
		ExtraHeaders:          extraHeaders,
		LatestExecutionResult: &latestExecutionResult,
	}

	mockSpacePathStore.On("FindPrimaryBySpaceID", ctx, int64(1)).Return(nil, fmt.Errorf("error finding primary by space ID"))

	webhookResponseEntity, err := helper.MapToWebhookResponseEntity(ctx, &createdWebhook)
	assert.Error(t, err)
	assert.Nil(t, webhookResponseEntity)
	assert.Contains(t, err.Error(), "failed to get secret space path: error finding primary by space ID")
}

func TestMapToInternalWebhookTriggers(t *testing.T) {
	helper := &GitnessRegistryMetadataHelper{}

	apiTriggers := []api.Trigger{
		api.TriggerARTIFACTCREATION,
		api.TriggerARTIFACTMODIFICATION,
		api.TriggerARTIFACTDELETION,
	}

	expectedInternalTriggers := []gitnessenum.WebhookTrigger{
		gitnessenum.WebhookTriggerArtifactCreated,
		gitnessenum.WebhookTriggerArtifactUpdated,
		gitnessenum.WebhookTriggerArtifactDeleted,
	}

	internalTriggers := helper.MapToInternalWebhookTriggers(apiTriggers)
	assert.Equal(t, expectedInternalTriggers, internalTriggers)
}

func TestMapToAPIWebhookTriggers(t *testing.T) {
	helper := &GitnessRegistryMetadataHelper{}

	internalTriggers := []gitnessenum.WebhookTrigger{
		gitnessenum.WebhookTriggerArtifactCreated,
		gitnessenum.WebhookTriggerArtifactUpdated,
		gitnessenum.WebhookTriggerArtifactDeleted,
	}

	expectedAPITriggers := []api.Trigger{
		api.TriggerARTIFACTCREATION,
		api.TriggerARTIFACTMODIFICATION,
		api.TriggerARTIFACTDELETION,
	}

	apiTriggers := helper.MapToAPIWebhookTriggers(internalTriggers)
	assert.Equal(t, expectedAPITriggers, apiTriggers)
}
