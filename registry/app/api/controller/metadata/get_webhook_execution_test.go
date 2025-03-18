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
	"testing"
	"time"

	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	gitnesstypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

//nolint:errcheck
func TestGetWebhookExecution_Success(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
	}

	regInfo := &RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "reg",
		ParentRef:          "root/parent",
	}
	space := &gitnesstypes.SpaceCore{ID: 2}
	var permissionChecks []gitnesstypes.PermissionCheck
	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", ctx, "", "reg").Return(regInfo, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(space, nil)
	mockRegistryMetadataHelper.On("GetPermissionChecks", space,
		regInfo.RegistryIdentifier, enum.PermissionRegistryView).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(true, nil)
	mockWebhooksExecutionRepository.On("Find", ctx, int64(1)).Return(&gitnesstypes.WebhookExecutionCore{
		ID:       1,
		Created:  time.Now().Unix(),
		Duration: 100,
		Error:    "none",
		Request: gitnesstypes.WebhookExecutionRequest{
			Body: "{}", Headers: "headers", URL: "http://example.com",
		},
		Response: gitnesstypes.WebhookExecutionResponse{
			Body: "{}", Headers: "headers", Status: "200 OK", StatusCode: 200,
		},
		RetriggerOf:   nil,
		Retriggerable: true,
		WebhookID:     4,
		Result:        enum.WebhookExecutionResultSuccess,
		TriggerType:   enum.WebhookTriggerArtifactCreated}, nil)

	r := api.GetWebhookExecutionRequestObject{
		RegistryRef:        "reg",
		WebhookIdentifier:  "webhook",
		WebhookExecutionId: "1",
	}

	response, err := controller.GetWebhookExecution(ctx, r)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, api.StatusSUCCESS, response.(api.GetWebhookExecution200JSONResponse).Status)
	assert.Equal(t, int64(1), *response.(api.GetWebhookExecution200JSONResponse).Data.Id)
	assert.Equal(t, "none", *response.(api.GetWebhookExecution200JSONResponse).Data.Error)
	assert.Equal(t, "{}", *response.(api.GetWebhookExecution200JSONResponse).Data.Request.Body)
	assert.Equal(t, "headers", *response.(api.GetWebhookExecution200JSONResponse).Data.Request.Headers)
	assert.Equal(t, "http://example.com", *response.(api.GetWebhookExecution200JSONResponse).Data.Request.Url)
	assert.Equal(t, "{}", *response.(api.GetWebhookExecution200JSONResponse).Data.Response.Body)
	assert.Equal(t, "headers", *response.(api.GetWebhookExecution200JSONResponse).Data.Response.Headers)
	assert.Equal(t, "200 OK", *response.(api.GetWebhookExecution200JSONResponse).Data.Response.Status)
	assert.Equal(t, 200, *response.(api.GetWebhookExecution200JSONResponse).Data.Response.StatusCode)
	assert.Equal(t, api.WebhookExecResultSUCCESS, *response.(api.GetWebhookExecution200JSONResponse).Data.Result)
	assert.Equal(t, api.TriggerARTIFACTCREATION, *response.(api.GetWebhookExecution200JSONResponse).Data.TriggerType)
	assert.Equal(t, api.StatusSUCCESS, response.(api.GetWebhookExecution200JSONResponse).Status)

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
	mockWebhooksExecutionRepository.AssertExpectations(t)
}

//nolint:errcheck
func TestGetWebhookExecution_PermissionCheckFails(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
	}

	regInfo := &RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "reg",
		ParentRef:          "root/parent",
	}
	space := &gitnesstypes.SpaceCore{ID: 2}
	var permissionChecks []gitnesstypes.PermissionCheck
	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", ctx, "", "reg").Return(regInfo, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(space, nil)
	mockRegistryMetadataHelper.On(
		"GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryView,
	).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(false, nil)

	r := api.GetWebhookExecutionRequestObject{
		RegistryRef:        "reg",
		WebhookIdentifier:  "webhook",
		WebhookExecutionId: "1",
	}

	response, err := controller.GetWebhookExecution(ctx, r)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.GetWebhookExecution403JSONResponse{}, response)
	assert.Equal(t, "not authorized", response.(api.GetWebhookExecution403JSONResponse).Message)

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
}

//nolint:errcheck
func TestGetWebhookExecution_InvalidExecutionIdentifier(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
	}

	regInfo := &RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "reg",
		ParentRef:          "root/parent",
	}
	space := &gitnesstypes.SpaceCore{ID: 2}
	var permissionChecks []gitnesstypes.PermissionCheck
	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", ctx, "", "reg").Return(regInfo, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(space, nil)
	mockRegistryMetadataHelper.On(
		"GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryView,
	).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(true, nil)

	r := api.GetWebhookExecutionRequestObject{
		RegistryRef:        "reg",
		WebhookIdentifier:  "webhook",
		WebhookExecutionId: "invalid",
	}

	response, err := controller.GetWebhookExecution(ctx, r)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.GetWebhookExecution400JSONResponse{}, response)
	assert.Equal(t,
		"invalid webhook execution identifier: invalid, err: strconv.ParseInt: parsing \"invalid\": invalid syntax",
		response.(api.GetWebhookExecution400JSONResponse).Message)

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
}

//nolint:errcheck
func TestGetWebhookExecution_FindExecutionError(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
	}

	regInfo := &RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "reg",
		ParentRef:          "root/parent",
	}
	space := &gitnesstypes.SpaceCore{ID: 2}
	var permissionChecks []gitnesstypes.PermissionCheck
	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", ctx, "", "reg").Return(regInfo, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(space, nil)
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier,
		enum.PermissionRegistryView).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(true, nil)
	mockWebhooksExecutionRepository.On("Find", ctx, int64(1)).
		Return(nil, fmt.Errorf("error"))

	r := api.GetWebhookExecutionRequestObject{
		RegistryRef:        "reg",
		WebhookIdentifier:  "webhook",
		WebhookExecutionId: "1",
	}

	response, err := controller.GetWebhookExecution(ctx, r)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.GetWebhookExecution500JSONResponse{}, response)
	assert.Equal(t, "failed to find webhook execution: error",
		response.(api.GetWebhookExecution500JSONResponse).Message)

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
	mockWebhooksExecutionRepository.AssertExpectations(t)
}
