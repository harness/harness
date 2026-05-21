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

	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/refcache"
	mockstore "github.com/harness/gitness/mocks/store"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type txAssertStub struct {
	inTx   bool
	called bool
}

func (t *txAssertStub) WithTx(ctx context.Context, txFn func(ctx context.Context) error, _ ...any) error {
	t.called = true
	t.inTx = true
	defer func() {
		t.inTx = false
	}()

	return txFn(ctx)
}

func TestAssignLabel_DeleteSuggestionNotFound_InsideTransaction(t *testing.T) {
	t.Parallel()

	repoID := int64(1)
	parentID := int64(10)
	pullreqID := int64(55)
	pullreqNum := int64(7)
	labelID := int64(123)

	tx := &txAssertStub{}
	repo := &types.RepositoryCore{ID: repoID, ParentID: parentID, Path: "space/repo", State: enum.RepoStateActive}
	pullreqStore := &mockstore.PullReqStore{}
	labelStore := &mockstore.LabelStore{}
	labelValueStore := &mockstore.LabelValueStore{}
	assignmentStore := &mockstore.PullReqLabelAssignmentStore{}
	suggestionStore := &mockstore.PullReqLabelSuggestionStore{}

	labelSvc := label.New(
		tx,
		nil,
		labelStore,
		labelValueStore,
		assignmentStore,
		suggestionStore,
		nil,
		refcache.SpaceFinder{},
	)

	pullreq := &types.PullReq{ID: pullreqID, Number: pullreqNum}
	pullreqStore.On("FindByNumber", repoID, pullreqNum).Return(pullreq, nil).Once()

	labelStore.On("FindByID", labelID).Return(&types.Label{ID: labelID, RepoID: &repoID}, nil).Once()
	assignmentStore.On("FindByLabelID", pullreqID, labelID).Return(&types.PullReqLabel{
		PullReqID: pullreqID,
		LabelID:   labelID,
	}, nil).Once()

	suggestionStore.
		On("Delete", pullreqID, labelID).
		Run(func(_ mock.Arguments) {
			require.True(t, tx.inTx, "suggestion deletion should happen inside transaction")
		}).
		Return(basestore.ErrResourceNotFound).
		Once()

	ctrl := &Controller{
		tx:                   tx,
		authorizer:           &allowAuthorizer{},
		repoFinder:           testRepoFinder(repo),
		pullreqStore:         pullreqStore,
		labelSvc:             labelSvc,
		labelSuggestionStore: suggestionStore,
	}

	out, err := ctrl.AssignLabel(
		context.Background(),
		testSession(),
		"1",
		pullreqNum,
		&types.PullReqLabelAssignInput{LabelID: labelID},
	)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, pullreqID, out.PullReqID)
	assert.Equal(t, labelID, out.LabelID)
	assert.True(t, tx.called)

	pullreqStore.AssertExpectations(t)
	labelStore.AssertExpectations(t)
	assignmentStore.AssertExpectations(t)
	suggestionStore.AssertExpectations(t)
}
