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

package metadata_test

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/api/controller/mocks"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/registry/utils"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListWebhooks_SortOrder(t *testing.T) {
	sortOrderDesc := api.SortOrder("DESC")
	sortOrderDescLower := api.SortOrder("desc")
	sortOrderSqli := api.SortOrder("ASC,(SELECT CASE WHEN (1=1) THEN 1 ELSE 0 END)")

	tests := []struct {
		name                string
		sortOrderParam      *api.SortOrder
		expectedSortByOrder string
		expectedStatus      int
	}{
		{
			name:                "nil_sort_order_defaults_to_asc",
			sortOrderParam:      nil,
			expectedSortByOrder: "ASC",
			expectedStatus:      200,
		},
		{
			name:                "uppercase_desc",
			sortOrderParam:      &sortOrderDesc,
			expectedSortByOrder: "DESC",
			expectedStatus:      200,
		},
		{
			name:                "lowercase_desc",
			sortOrderParam:      &sortOrderDescLower,
			expectedSortByOrder: "DESC",
			expectedStatus:      200,
		},
		{
			name:                "sqli_attempt_normalized_to_asc",
			sortOrderParam:      &sortOrderSqli,
			expectedSortByOrder: "ASC",
			expectedStatus:      200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := &metadata.APIController{}

			mockSpaceFinder := new(mocks.SpaceFinder)
			mockMetadataHelper := new(mocks.RegistryMetadataHelper)
			mockAuthorizer := new(mocks.Authorizer)
			mockWebhooksRepo := new(mocks.WebhooksRepository)

			regInfo := &registrytypes.RegistryRequestBaseInfo{
				RegistryID:         1,
				RegistryIdentifier: "reg",
				ParentID:           2,
				ParentRef:          "root/parent",
				RegistryRef:        "reg",
			}
			space := &coretypes.SpaceCore{
				Path: "root/parent",
			}
			permissionChecks := []coretypes.PermissionCheck{
				{
					Scope: coretypes.Scope{SpacePath: "root/parent"},
					Resource: coretypes.Resource{
						Type:       enum.ResourceTypeRegistry,
						Identifier: "reg",
					},
					Permission: enum.PermissionRegistryView,
				},
			}

			mockMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
			mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
			mockMetadataHelper.On("GetPermissionChecks", space, "reg", enum.PermissionRegistryView).Return(permissionChecks)
			mockSession := &auth.Session{Principal: coretypes.Principal{ID: 123}}
			mockAuthorizer.On("CheckAll", mock.Anything, mockSession, permissionChecks[0]).Return(true, nil)

			// WebhooksRepository.ListByRegistry should be called with expectedSortByOrder
			mockWebhooksRepo.On("ListByRegistry",
				mock.Anything,
				"",
				tt.expectedSortByOrder,
				10,
				0,
				"",
				int64(1),
			).Return([]*coretypes.WebhookCore{}, nil)

			mockWebhooksRepo.On("CountAllByRegistry", mock.Anything, int64(1), "").Return(int64(0), nil)

			controller.SpaceFinder = mockSpaceFinder
			controller.RegistryMetadataHelper = mockMetadataHelper
			controller.Authorizer = mockAuthorizer
			controller.WebhooksRepository = mockWebhooksRepo

			req := api.ListWebhooksRequestObject{
				RegistryRef: "reg",
				Params: api.ListWebhooksParams{
					Size:      utils.PageSizePtr(10),
					Page:      utils.PageNumberPtr(0),
					SortOrder: tt.sortOrderParam,
				},
			}

			ctx := request.WithAuthSession(context.Background(), mockSession)
			resp, err := controller.ListWebhooks(ctx, req)
			require.NoError(t, err)

			_, ok := resp.(api.ListWebhooks200JSONResponse)
			assert.True(t, ok, "expected ListWebhooks200JSONResponse, got %T", resp)

			mockWebhooksRepo.AssertExpectations(t)
		})
	}
}
