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

package pullreq

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/auth/authz/authztest"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/refcache"
	mockstore "github.com/harness/gitness/mocks/store"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUnassignLabel(t *testing.T) {
	t.Parallel()

	const (
		repoID     = int64(1)
		parentID   = int64(10)
		pullreqID  = int64(55)
		pullreqNum = int64(7)
		labelID    = int64(123)
	)

	tests := []struct {
		name         string
		deleted      bool
		wantActivity bool
	}{
		{
			name:         "assigned label writes unassign activity",
			deleted:      true,
			wantActivity: true,
		},
		{
			name:         "unassigned label skips activity",
			deleted:      false,
			wantActivity: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &types.RepositoryCore{
				ID: repoID, ParentID: parentID, Path: "space/repo", State: enum.RepoStateActive,
			}
			pullreqStore := &mockstore.PullReqStore{}
			labelStore := &mockstore.LabelStore{}
			labelValueStore := &mockstore.LabelValueStore{}
			assignmentStore := &mockstore.PullReqLabelAssignmentStore{}
			activityStore := &mockstore.PullReqActivityStore{}

			labelSvc := label.New(
				nil,
				nil,
				labelStore,
				labelValueStore,
				assignmentStore,
				nil,
				nil,
				refcache.SpaceFinder{},
			)

			pullreq := &types.PullReq{ID: pullreqID, Number: pullreqNum}
			pullreqStore.On("FindByNumber", repoID, pullreqNum).Return(pullreq, nil).Once()

			repoIDCopy := repoID
			labelStore.On("FindByID", labelID).
				Return(&types.Label{ID: labelID, RepoID: &repoIDCopy, Key: "sd"}, nil).Once()
			assignmentStore.On("FindValueByLabelID", pullreqID, labelID).
				Return(nil, basestore.ErrResourceNotFound).Once()
			assignmentStore.On("Unassign", pullreqID, labelID).Return(tt.deleted, nil).Once()

			if tt.wantActivity {
				pullreqStore.On("UpdateActivitySeq", pullreq).Return(pullreq, nil).Once()
				activityStore.
					On("CreateWithPayload", pullreq, mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						payload, ok := args.Get(2).(*types.PullRequestActivityLabel)
						require.True(t, ok, "payload should be a label activity")
						require.Equal(t, enum.LabelActivityUnassign, payload.Type)
					}).
					Return(&types.PullReqActivity{}, nil).
					Once()
			}

			ctrl := &Controller{
				authorizer:    authztest.AllowAuthorizer{},
				repoFinder:    testRepoFinder(repo),
				pullreqStore:  pullreqStore,
				labelSvc:      labelSvc,
				activityStore: activityStore,
			}

			err := ctrl.UnassignLabel(context.Background(), testSession(), "1", pullreqNum, labelID)
			require.NoError(t, err)

			if !tt.wantActivity {
				pullreqStore.AssertNotCalled(t, "UpdateActivitySeq", mock.Anything)
				activityStore.AssertNotCalled(t, "CreateWithPayload",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			}

			pullreqStore.AssertExpectations(t)
			labelStore.AssertExpectations(t)
			assignmentStore.AssertExpectations(t)
			activityStore.AssertExpectations(t)
		})
	}
}
