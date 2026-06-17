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
	"strings"
	"testing"

	"github.com/harness/gitness/errors"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	linkedTestRepoID int64 = 1
	linkedTestPRNum  int64 = 7
)

func linkedPullReq() *types.PullReq {
	prType := enum.PullReqTypeLinked
	return &types.PullReq{
		ID:     55,
		Number: linkedTestPRNum,
		State:  enum.PullReqStateOpen,
		Type:   &prType,
	}
}

func newLinkedPullReqController(pr *types.PullReq) (*Controller, *mockstore.PullReqStore) {
	repo := &types.RepositoryCore{
		ID:       linkedTestRepoID,
		ParentID: 10,
		Path:     "space/repo",
		State:    enum.RepoStateActive,
	}
	pullreqStore := &mockstore.PullReqStore{}
	pullreqStore.
		On("FindByNumber", linkedTestRepoID, linkedTestPRNum).
		Return(pr, nil).
		Once()

	return &Controller{
		authorizer:   &allowAuthorizer{},
		repoFinder:   testRepoFinder(repo),
		pullreqStore: pullreqStore,
	}, pullreqStore
}

func assertForbiddenError(t *testing.T, err error, msg string) {
	t.Helper()

	require.Error(t, err)
	var forbidden *errors.Error
	require.True(t, errors.As(err, &forbidden), "expected forbidden error, got: %v", err)
	assert.Equal(t, errors.StatusForbidden, forbidden.Status)
	assert.Contains(t, forbidden.Message, msg)
}

func TestMerge_LinkedPullReqForbidden(t *testing.T) {
	t.Parallel()

	ctrl, pullreqStore := newLinkedPullReqController(linkedPullReq())

	_, _, err := ctrl.Merge(context.Background(), testSession(), "1", linkedTestPRNum, &MergeInput{
		SourceSHA:   "abc123",
		DryRunRules: true,
	})
	assertForbiddenError(t, err, "linked pull request")
	pullreqStore.AssertExpectations(t)
}

func TestState_LinkedPullReqForbidden(t *testing.T) {
	t.Parallel()

	ctrl, pullreqStore := newLinkedPullReqController(linkedPullReq())

	_, err := ctrl.State(context.Background(), testSession(), "1", linkedTestPRNum, &StateInput{
		State: enum.PullReqStateClosed,
	})
	assertForbiddenError(t, err, "linked pull request")
	pullreqStore.AssertExpectations(t)
}

func TestUpdate_LinkedPullReqForbidden(t *testing.T) {
	t.Parallel()

	ctrl, pullreqStore := newLinkedPullReqController(linkedPullReq())

	_, err := ctrl.Update(context.Background(), testSession(), "1", linkedTestPRNum, &UpdateInput{
		Title:       "Updated title",
		Description: "Updated description",
	})
	assertForbiddenError(t, err, "linked pull request")
	pullreqStore.AssertExpectations(t)
}

func TestVerifyIfAutoMergeable_LinkedPullReqForbidden(t *testing.T) {
	t.Parallel()

	err := verifyIfAutoMergeable(linkedPullReq())
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "linked pull requests"))
}
