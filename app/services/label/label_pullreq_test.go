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

package label_test

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/refcache"
	mockstore "github.com/harness/gitness/mocks/store"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnassignFromPullReq(t *testing.T) {
	t.Parallel()

	const (
		repoID    = int64(1)
		parentID  = int64(10)
		pullreqID = int64(55)
		labelID   = int64(123)
	)

	tests := []struct {
		name         string
		deleted      bool
		wantActivity enum.PullReqLabelActivityType
	}{
		{
			name:         "assigned label produces unassign activity",
			deleted:      true,
			wantActivity: enum.LabelActivityUnassign,
		},
		{
			name:         "unassigned label is a no-op",
			deleted:      false,
			wantActivity: enum.LabelActivityNoop,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			labelStore := &mockstore.LabelStore{}
			labelValueStore := &mockstore.LabelValueStore{}
			assignmentStore := &mockstore.PullReqLabelAssignmentStore{}

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

			repoIDCopy := repoID
			lbl := &types.Label{ID: labelID, RepoID: &repoIDCopy, Key: "sd"}
			labelStore.On("FindByID", labelID).Return(lbl, nil).Once()
			assignmentStore.On("FindValueByLabelID", pullreqID, labelID).
				Return(nil, basestore.ErrResourceNotFound).Once()
			assignmentStore.On("Unassign", pullreqID, labelID).Return(tt.deleted, nil).Once()

			gotLabel, gotValue, gotActivity, err := labelSvc.UnassignFromPullReq(
				context.Background(), repoID, parentID, pullreqID, labelID)
			require.NoError(t, err)
			assert.Equal(t, lbl, gotLabel)
			assert.Nil(t, gotValue)
			assert.Equal(t, tt.wantActivity, gotActivity)

			labelStore.AssertExpectations(t)
			assignmentStore.AssertExpectations(t)
		})
	}
}
