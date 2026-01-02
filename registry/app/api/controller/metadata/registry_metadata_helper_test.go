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

//nolint:lll,revive // revive:disable:unused-parameter
package metadata

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/controller/mocks"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	gitnesstypes "github.com/harness/gitness/types"
	gitnessenum "github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test data setup.
var (
	regInfo = &types.RegistryRequestBaseInfo{
		RegistryID: 1,
	}
)

// Helper function to create string pointer.
func stringPtr(s string) *string {
	return &s
}

func TestMapToWebhook(t *testing.T) {
	tests := []struct {
		name          string
		webhookReq    api.WebhookRequest
		setupMocks    func(*mocks.SpacePathStore)
		expectedError string
		validate      func(*testing.T, *gitnesstypes.WebhookCore, error)
	}{
		{
			name: "success_case",
			webhookReq: api.WebhookRequest{
				Identifier:  "webhook",
				Name:        "webhook",
				Url:         "http://example.com",
				Enabled:     true,
				Insecure:    false,
				Triggers:    &[]api.Trigger{api.TriggerARTIFACTCREATION, api.TriggerARTIFACTDELETION},
				Description: stringPtr("Test webhook"),
				ExtraHeaders: &[]api.ExtraHeader{
					{Key: "key", Value: "value"},
				},
			},
			validate: func(t *testing.T, webhook *gitnesstypes.WebhookCore, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, webhook)
				assert.Equal(t, "webhook", webhook.Identifier)
				assert.Equal(t, "webhook", webhook.DisplayName)
				assert.Equal(t, "http://example.com", webhook.URL)
				assert.True(t, webhook.Enabled)
				assert.False(t, webhook.Insecure)
				assert.Equal(t, "Test webhook", webhook.Description)
				assert.Len(t, webhook.ExtraHeaders, 1)
				assert.Equal(t, "key", webhook.ExtraHeaders[0].Key)
				assert.Equal(t, "value", webhook.ExtraHeaders[0].Value)
				assert.Len(t, webhook.Triggers, 2)
				assert.Equal(t, gitnessenum.WebhookTriggerArtifactCreated, webhook.Triggers[0])
				assert.Equal(t, gitnessenum.WebhookTriggerArtifactDeleted, webhook.Triggers[1])
			},
		},
		{
			name: "with_secret_space_path",
			webhookReq: api.WebhookRequest{
				Identifier:      "webhook",
				Name:            "webhook",
				Url:             "http://example.com",
				Enabled:         true,
				Insecure:        false,
				Triggers:        &[]api.Trigger{api.TriggerARTIFACTCREATION},
				SecretSpacePath: stringPtr("secret/path"),
			},
			setupMocks: func(mockSpacePathStore *mocks.SpacePathStore) {
				mockSpacePathStore.On("FindByPath", mock.Anything, "secret/path").Return(
					&gitnesstypes.SpacePath{Value: "secret/path", SpaceID: 2}, nil)
			},
			validate: func(t *testing.T, webhook *gitnesstypes.WebhookCore, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, webhook)
				assert.Equal(t, int64(2), webhook.SecretSpaceID)
			},
		},
		{
			name: "invalid_secret_space_path",
			webhookReq: api.WebhookRequest{
				Identifier:      "webhook",
				Name:            "webhook",
				Url:             "http://example.com",
				Enabled:         true,
				Insecure:        false,
				Triggers:        &[]api.Trigger{api.TriggerARTIFACTCREATION},
				SecretSpacePath: stringPtr("secret/path"),
			},
			setupMocks: func(mockSpacePathStore *mocks.SpacePathStore) {
				mockSpacePathStore.On("FindByPath", mock.Anything, "secret/path").Return(nil, fmt.Errorf("not found"))
			},
			expectedError: "failed to get Space Path: not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSpacePathStore := new(mocks.SpacePathStore)

			// Create helper
			helper := &GitnessRegistryMetadataHelper{
				spacePathStore: mockSpacePathStore,
			}

			// Setup mocks if provided
			if tt.setupMocks != nil {
				tt.setupMocks(mockSpacePathStore)
			}

			// Execute test
			webhook, err := helper.MapToWebhookCore(context.Background(), tt.webhookReq, regInfo)

			// Validate results
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, webhook)
			} else if tt.validate != nil {
				tt.validate(t, webhook, err)
			}

			// Verify mock expectations
			mockSpacePathStore.AssertExpectations(t)
		})
	}
}

func TestMapToWebhook_WithSecretSpacePath(t *testing.T) {
	ctx := context.Background()
	mockSpacePathStore := new(mocks.SpacePathStore)
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockRegistryRepository := new(mocks.RegistryRepository)
	helper := NewRegistryMetadataHelper(mockSpacePathStore, mockSpaceFinder, mockRegistryRepository)

	secretSpacePath := "secret/path"
	webhookRequest := api.WebhookRequest{
		Identifier:      "webhook",
		Name:            "webhook",
		Url:             "http://example.com",
		Enabled:         true,
		Insecure:        false,
		Triggers:        &[]api.Trigger{api.TriggerARTIFACTCREATION},
		SecretSpacePath: &secretSpacePath,
	}

	mockSpacePathStore.On(
		"FindByPath",
		ctx,
		"secret/path",
	).Return(&gitnesstypes.SpacePath{
		Value:   "secret/path",
		SpaceID: 2,
	}, nil)

	webhook, err := helper.MapToWebhookCore(ctx, webhookRequest, regInfo)
	assert.NoError(t, err)
	assert.NotNil(t, webhook)
	assert.Equal(t, int64(2), webhook.SecretSpaceID)

	mockSpacePathStore.AssertExpectations(t)
}

func TestMapToWebhook_WithInexistentSecretSpacePath(t *testing.T) {
	ctx := context.Background()
	mockSpacePathStore := new(mocks.SpacePathStore)
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockRegistryRepository := new(mocks.RegistryRepository)
	helper := NewRegistryMetadataHelper(mockSpacePathStore, mockSpaceFinder, mockRegistryRepository)

	secretSpacePath := "secret/path"
	webhookRequest := api.WebhookRequest{
		Identifier:      "webhook",
		Name:            "webhook",
		Url:             "http://example.com",
		Enabled:         true,
		Insecure:        false,
		Triggers:        &[]api.Trigger{api.TriggerARTIFACTCREATION},
		SecretSpacePath: &secretSpacePath,
	}

	mockSpacePathStore.On("FindByPath", ctx, "secret/path").
		Return(nil, fmt.Errorf("error finding path"))

	webhook, err := helper.MapToWebhookCore(ctx, webhookRequest, regInfo)
	assert.Error(t, err)
	assert.Nil(t, webhook)
	assert.Equal(t, "failed to get Space Path: error finding path", err.Error())
}

func TestMapToWebhookResponseEntity(t *testing.T) {
	tests := []struct {
		name          string
		webhook       *gitnesstypes.WebhookCore
		setupMocks    func(*mocks.SpacePathStore, *mocks.SpaceFinder, *mocks.RegistryRepository)
		expectedError string
		validate      func(*testing.T, *api.Webhook, error)
	}{
		{
			name: "success_case",
			webhook: &gitnesstypes.WebhookCore{
				Type:                  gitnessenum.WebhookTypeInternal,
				Identifier:            "webhook",
				DisplayName:           "webhook",
				URL:                   "http://example.com",
				Enabled:               true,
				Insecure:              false,
				Triggers:              []gitnessenum.WebhookTrigger{gitnessenum.WebhookTriggerArtifactCreated},
				Created:               time.Now().Unix(),
				Updated:               time.Now().Unix(),
				Description:           "Test webhook",
				SecretIdentifier:      "secret-id",
				SecretSpaceID:         1,
				ExtraHeaders:          []gitnesstypes.ExtraHeader{{Key: "key", Value: "value"}},
				LatestExecutionResult: &[]gitnessenum.WebhookExecutionResult{gitnessenum.WebhookExecutionResultSuccess}[0],
			},
			setupMocks: func(mockSpacePathStore *mocks.SpacePathStore, _ *mocks.SpaceFinder, _ *mocks.RegistryRepository) {
				mockSpacePathStore.On("FindPrimaryBySpaceID", mock.Anything, int64(1)).Return(
					&gitnesstypes.SpacePath{Value: "secret/path"}, nil)
			},
			validate: func(t *testing.T, webhook *api.Webhook, err error) {
				assert.NoError(t, err, "should not return error")
				assert.NotNil(t, webhook, "webhook should not be nil")

				// Basic fields
				assert.Equal(t, "webhook", webhook.Identifier, "identifier should match")
				assert.Equal(t, "webhook", webhook.Name, "name should match")
				assert.Equal(t, "http://example.com", webhook.Url, "URL should match")
				assert.True(t, webhook.Enabled, "should be enabled")
				assert.False(t, webhook.Insecure, "should not be insecure")
				assert.Equal(t, "Test webhook", *webhook.Description, "description should match")
				assert.True(t, *webhook.Internal, "should be internal")

				// Triggers
				assert.Len(t, *webhook.Triggers, 1, "should have 1 trigger")
				assert.Equal(t, api.TriggerARTIFACTCREATION, (*webhook.Triggers)[0], "trigger should match")

				// Secret fields
				assert.Equal(t, "secret-id", *webhook.SecretIdentifier, "secret identifier should match")
				assert.Equal(t, "secret/path", *webhook.SecretSpacePath, "secret space path should match")
				assert.Equal(t, int64(1), *webhook.SecretSpaceId, "secret space ID should match")

				// Extra headers
				assert.Len(t, *webhook.ExtraHeaders, 1, "should have 1 extra header")
				assert.Equal(t, "key", (*webhook.ExtraHeaders)[0].Key, "header key should match")
				assert.Equal(t, "value", (*webhook.ExtraHeaders)[0].Value, "header value should match")

				// Latest execution result
				assert.Equal(t, api.WebhookExecResultSUCCESS, *webhook.LatestExecutionResult, "latest execution result should match")
			},
		},
		{
			name: "find_primary_by_space_id_error",
			webhook: &gitnesstypes.WebhookCore{
				SecretSpaceID: 1,
			},
			setupMocks: func(mockSpacePathStore *mocks.SpacePathStore, _ *mocks.SpaceFinder, _ *mocks.RegistryRepository) {
				mockSpacePathStore.On("FindPrimaryBySpaceID", mock.Anything, int64(1)).Return(nil, fmt.Errorf("error finding primary by space ID"))
			},
			expectedError: "failed to get secret space path: error finding primary by space ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSpacePathStore := new(mocks.SpacePathStore)
			mockSpaceFinder := new(mocks.SpaceFinder)
			mockRegistryRepository := new(mocks.RegistryRepository)

			// Create helper
			helper := &GitnessRegistryMetadataHelper{
				spacePathStore:     mockSpacePathStore,
				spaceFinder:        mockSpaceFinder,
				registryRepository: mockRegistryRepository,
			}

			// Setup mocks if provided
			if tt.setupMocks != nil {
				tt.setupMocks(mockSpacePathStore, mockSpaceFinder, mockRegistryRepository)
			}

			// Execute test
			webhook, err := helper.MapToWebhookResponseEntity(context.Background(), tt.webhook)

			// Validate results
			if tt.expectedError != "" {
				assert.Error(t, err, "should return error")
				assert.Contains(t, err.Error(), tt.expectedError, "error message should match")
				assert.Nil(t, webhook, "webhook should be nil")
			} else if tt.validate != nil {
				tt.validate(t, webhook, err)
			}

			// Verify mock expectations
			mockSpacePathStore.AssertExpectations(t)
			mockSpaceFinder.AssertExpectations(t)
			mockRegistryRepository.AssertExpectations(t)
		})
	}
}

func TestMapToWebhookResponseEntity_FindByPathError(t *testing.T) {
	// GIVEN
	mockSpacePathStore := new(mocks.SpacePathStore)
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockRegistryDao := new(mocks.RegistryRepository)

	helper := NewRegistryMetadataHelper(mockSpacePathStore, mockSpaceFinder, mockRegistryDao)

	webhook := &gitnesstypes.WebhookCore{
		ID:            1,
		ParentID:      1,
		Type:          gitnessenum.WebhookTypeInternal,
		Identifier:    "webhook",
		DisplayName:   "webhook",
		URL:           "http://example.com",
		Enabled:       true,
		Insecure:      false,
		SecretSpaceID: 1,
		Created:       1234567890,
		Updated:       1234567890,
	}

	mockSpacePathStore.On("FindPrimaryBySpaceID", mock.Anything, int64(1)).Return(nil, fmt.Errorf("space not found"))

	// WHEN
	result, err := helper.MapToWebhookResponseEntity(context.Background(), webhook)

	// THEN
	assert.Error(t, err)
	assert.Nil(t, result)
	mockSpacePathStore.AssertExpectations(t)
}

func TestGetRegistryRequestBaseInfo(t *testing.T) {
	tests := []struct {
		name          string
		parentRef     string
		registryName  string
		setupMocks    func(*mocks.SpacePathStore, *mocks.SpaceFinder, *mocks.RegistryRepository)
		expectedError string
		validate      func(*testing.T, *types.RegistryRequestBaseInfo, error)
	}{
		{
			name:         "success_case",
			parentRef:    "root/parent",
			registryName: "reg",
			setupMocks: func(_ *mocks.SpacePathStore, mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepository *mocks.RegistryRepository) {
				mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(&gitnesstypes.SpaceCore{ID: 1}, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(&gitnesstypes.SpaceCore{ID: 2}, nil)
				mockRegistryRepository.On("GetByParentIDAndName", mock.Anything, int64(2), "reg").Return(&types.Registry{
					ID:   3,
					Type: api.RegistryTypeVIRTUAL,
				}, nil)
			},
			validate: func(t *testing.T, baseInfo *types.RegistryRequestBaseInfo, err error) {
				assert.NoError(t, err, "should not return error")
				assert.NotNil(t, baseInfo, "baseInfo should not be nil")
				assert.Equal(t, int64(1), baseInfo.RootIdentifierID, "root ID should match")
				assert.Equal(t, int64(2), baseInfo.ParentID, "parent ID should match")
				assert.Equal(t, int64(3), baseInfo.RegistryID, "registry ID should match")
				assert.Equal(t, api.RegistryTypeVIRTUAL, baseInfo.RegistryType, "registry type should match")
			},
		},
		{
			name:          "empty_parent_ref",
			parentRef:     "",
			registryName:  "reg",
			expectedError: "parent reference is required",
		},
		{
			name:          "invalid_parent_ref",
			parentRef:     "/",
			registryName:  "reg",
			expectedError: "invalid parent reference",
		},
		{
			name:         "root_space_not_found",
			parentRef:    "root/parent",
			registryName: "reg",
			setupMocks: func(_ *mocks.SpacePathStore, mockSpaceFinder *mocks.SpaceFinder, _ *mocks.RegistryRepository) {
				mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(nil, fmt.Errorf("not found"))
			},
			expectedError: "root space not found",
		},
		{
			name:         "parent_space_not_found",
			parentRef:    "root/parent",
			registryName: "reg",
			setupMocks: func(_ *mocks.SpacePathStore, mockSpaceFinder *mocks.SpaceFinder, _ *mocks.RegistryRepository) {
				mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(&gitnesstypes.SpaceCore{ID: 1}, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(nil, fmt.Errorf("not found"))
			},
			expectedError: "parent space not found",
		},
		{
			name:         "registry_not_found",
			parentRef:    "root/parent",
			registryName: "reg",
			setupMocks: func(_ *mocks.SpacePathStore, mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepository *mocks.RegistryRepository) {
				mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(&gitnesstypes.SpaceCore{ID: 1}, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(&gitnesstypes.SpaceCore{ID: 2}, nil)
				mockRegistryRepository.On("GetByParentIDAndName", mock.Anything, int64(2), "reg").Return(nil, fmt.Errorf("not found"))
			},
			expectedError: "registry not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSpacePathStore := new(mocks.SpacePathStore)
			mockSpaceFinder := new(mocks.SpaceFinder)
			mockRegistryRepository := new(mocks.RegistryRepository)

			// Create helper
			helper := &GitnessRegistryMetadataHelper{
				spacePathStore:     mockSpacePathStore,
				spaceFinder:        mockSpaceFinder,
				registryRepository: mockRegistryRepository,
			}

			// Setup mocks if provided
			if tt.setupMocks != nil {
				tt.setupMocks(mockSpacePathStore, mockSpaceFinder, mockRegistryRepository)
			}

			// Execute test
			baseInfo, err := helper.GetRegistryRequestBaseInfo(context.Background(), tt.parentRef, tt.registryName)

			// Validate results
			if tt.expectedError != "" {
				assert.Error(t, err, "should return error")
				assert.Contains(t, err.Error(), tt.expectedError, "error message should match")
				assert.Nil(t, baseInfo, "baseInfo should be nil")
			} else if tt.validate != nil {
				tt.validate(t, baseInfo, err)
			}

			// Verify mock expectations
			mockSpacePathStore.AssertExpectations(t)
			mockSpaceFinder.AssertExpectations(t)
			mockRegistryRepository.AssertExpectations(t)
		})
	}
}

func TestMapToInternalWebhookTriggers(t *testing.T) {
	helper := &GitnessRegistryMetadataHelper{}

	tests := []struct {
		name      string
		triggers  []api.Trigger
		expected  []gitnessenum.WebhookTrigger
		expectErr bool
	}{
		{
			name: "success_case",
			triggers: []api.Trigger{
				api.TriggerARTIFACTCREATION,
				api.TriggerARTIFACTDELETION,
			},
			expected: []gitnessenum.WebhookTrigger{
				gitnessenum.WebhookTriggerArtifactCreated,
				gitnessenum.WebhookTriggerArtifactDeleted,
			},
			expectErr: false,
		},
		{
			name:      "empty_triggers",
			triggers:  []api.Trigger{},
			expected:  []gitnessenum.WebhookTrigger{},
			expectErr: false,
		},
		{
			name:      "invalid_trigger",
			triggers:  []api.Trigger{"INVALID_TRIGGER"},
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := helper.MapToInternalWebhookTriggers(tt.triggers)
			if tt.expectErr {
				assert.Error(t, err, "should return error for invalid triggers")
				assert.Nil(t, result, "result should be nil on error")
			} else {
				assert.NoError(t, err, "should not return error")
				assert.Equal(t, tt.expected, result, "triggers should match")
			}
		})
	}
}

func TestMapToAPIWebhookTriggers(t *testing.T) {
	helper := &GitnessRegistryMetadataHelper{}

	tests := []struct {
		name     string
		triggers []gitnessenum.WebhookTrigger
		expected []api.Trigger
	}{
		{
			name: "success_case",
			triggers: []gitnessenum.WebhookTrigger{
				gitnessenum.WebhookTriggerArtifactCreated,
				gitnessenum.WebhookTriggerArtifactDeleted,
			},
			expected: []api.Trigger{
				api.TriggerARTIFACTCREATION,
				api.TriggerARTIFACTDELETION,
			},
		},
		{
			name:     "empty_triggers",
			triggers: []gitnessenum.WebhookTrigger{},
			expected: []api.Trigger{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.MapToAPIWebhookTriggers(tt.triggers)
			assert.Equal(t, tt.expected, result, "triggers should match")
		})
	}
}
