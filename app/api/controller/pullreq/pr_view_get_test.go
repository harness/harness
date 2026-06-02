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
	"errors"
	"strings"
	"testing"

	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullReqViewGet_SuccessFiltersEmptyGroups(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, ParentID: 10, Path: "space/repo", State: enum.RepoStateActive}
	pullreqStore := &mockstore.PullReqStore{}
	fileGroupStore := &mockstore.PullReqFileGroupStore{}

	pullreqStore.On("FindByNumber", int64(1), int64(7)).Return(&types.PullReq{ID: 55, Number: 7}, nil).Once()
	fileGroupStore.On("List", int64(55)).Return([]*types.PullReqFileGroupWithFiles{
		{
			PullReqFileGroup: types.PullReqFileGroup{Title: "empty-group"},
			Files:            nil,
		},
		{
			PullReqFileGroup: types.PullReqFileGroup{Title: "backend"},
			Files: []*types.PullReqFileGroupFile{
				{Path: "a.txt", OldSHA: "old-a", NewSHA: "new-a"},
			},
		},
	}, nil).Once()

	ctrl := &Controller{
		authorizer:     &allowAuthorizer{},
		repoFinder:     testRepoFinder(repo),
		pullreqStore:   pullreqStore,
		fileGroupStore: fileGroupStore,
	}

	out, err := ctrl.PullReqViewGet(context.Background(), testSession(), "1", 7)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out.Groups, 1)
	assert.Equal(t, "backend", out.Groups[0].Title)
	require.Len(t, out.Groups[0].Files, 1)
	assert.Equal(t, "a.txt", out.Groups[0].Files[0].Path)

	pullreqStore.AssertExpectations(t)
	fileGroupStore.AssertExpectations(t)
}

func TestPullReqViewGet_FindByNumberError(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, ParentID: 10, Path: "space/repo", State: enum.RepoStateActive}
	pullreqStore := &mockstore.PullReqStore{}
	fileGroupStore := &mockstore.PullReqFileGroupStore{}

	pullreqStore.On("FindByNumber", int64(1), int64(7)).Return((*types.PullReq)(nil), errors.New("boom")).Once()

	ctrl := &Controller{
		authorizer:     &allowAuthorizer{},
		repoFinder:     testRepoFinder(repo),
		pullreqStore:   pullreqStore,
		fileGroupStore: fileGroupStore,
	}

	_, err := ctrl.PullReqViewGet(context.Background(), testSession(), "1", 7)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to find pull request by number"))

	pullreqStore.AssertExpectations(t)
	fileGroupStore.AssertNotCalled(t, "List", int64(55))
}

func TestPullReqViewGet_ListError(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, ParentID: 10, Path: "space/repo", State: enum.RepoStateActive}
	pullreqStore := &mockstore.PullReqStore{}
	fileGroupStore := &mockstore.PullReqFileGroupStore{}

	pullreqStore.On("FindByNumber", int64(1), int64(7)).Return(&types.PullReq{ID: 55, Number: 7}, nil).Once()
	fileGroupStore.On("List", int64(55)).Return(([]*types.PullReqFileGroupWithFiles)(nil), errors.New("boom")).Once()

	ctrl := &Controller{
		authorizer:     &allowAuthorizer{},
		repoFinder:     testRepoFinder(repo),
		pullreqStore:   pullreqStore,
		fileGroupStore: fileGroupStore,
	}

	_, err := ctrl.PullReqViewGet(context.Background(), testSession(), "1", 7)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to list pull request file groups"))

	pullreqStore.AssertExpectations(t)
	fileGroupStore.AssertExpectations(t)
}
