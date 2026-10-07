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
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	mockgit "github.com/harness/gitness/mocks/git"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	_ "unsafe" // for go:linkname
)

// bootstrapSystemServicePrincipal aliases the unexported package-level principal used by
// bootstrap.NewSystemServiceSession(), which the revert uses as the committer.
// It's seeded here so that the session lookup doesn't deref nil.
//
//go:linkname bootstrapSystemServicePrincipal github.com/harness/gitness/app/bootstrap.systemServicePrincipal
var bootstrapSystemServicePrincipal *types.Principal

func init() {
	bootstrapSystemServicePrincipal = &types.Principal{
		ID:    1,
		UID:   "harness-test",
		Email: "harness-test@local",
	}
}

// SHAs of the commits involved in a revert. The values are arbitrary, only their roles matter.
const (
	revertMergeSHAStr       = "1111111111111111111111111111111111111111" // the PR's merge commit
	revertMergeTargetSHAStr = "2222222222222222222222222222222222222222" // target branch tip before the merge
	revertMergeBaseSHAStr   = "3333333333333333333333333333333333333333" // must not be used by revert
	revertSourceSHAStr      = "4444444444444444444444444444444444444444" // must not be used by revert
	revertCommitSHAStr      = "5555555555555555555555555555555555555555" // the created revert commit
)

// revertURLProvider is a minimal url.Provider stub. Only GetInternalAPIURL is needed to
// create the git write params, so the rest of the interface is embedded and left unimplemented.
type revertURLProvider struct{ url.Provider }

func (*revertURLProvider) GetInternalAPIURL(context.Context) string { return "http://localhost" }

// mergedPullReq returns a pull request merged with the SHAs declared above.
func mergedPullReq(pullreqID, pullreqNum int64) *types.PullReq {
	return &types.PullReq{
		ID:             pullreqID,
		Number:         pullreqNum,
		Title:          "Some change",
		State:          enum.PullReqStateMerged,
		MergeSHA:       ptr.String(revertMergeSHAStr),
		MergeTargetSHA: ptr.String(revertMergeTargetSHAStr),
		MergeBaseSHA:   revertMergeBaseSHAStr,
		SourceSHA:      revertSourceSHAStr,
	}
}

func revertController(
	repo *types.RepositoryCore,
	pullreqStore *mockstore.PullReqStore,
	gitMock *mockgit.Interface,
) *Controller {
	return &Controller{
		authorizer:   authztest.AllowAuthorizer{},
		urlProvider:  &revertURLProvider{},
		repoFinder:   testRepoFinder(repo),
		pullreqStore: pullreqStore,
		git:          gitMock,
	}
}

// TestRevert_GitParams locks in the commits that the revert of a merged pull request is based on:
// the diff has to be taken between the merge commit and the target branch tip that the merge was
// based on, and it has to be applied on top of the merge commit itself.
// Using the merge base or the source branch tip instead produces a patch that was generated
// against a different tree than the one it's applied to, which fails to apply
// whenever the target branch changed the same regions.
func TestRevert_GitParams(t *testing.T) {
	t.Parallel()

	const (
		repoID     = int64(1)
		pullreqID  = int64(55)
		pullreqNum = int64(7)
	)

	repo := &types.RepositoryCore{
		ID: repoID, ParentID: 10, Path: "space/repo", GitUID: "git-uid", State: enum.RepoStateActive,
	}

	pullreqStore := &mockstore.PullReqStore{}
	pullreqStore.On("FindByNumber", repoID, pullreqNum).Return(mergedPullReq(pullreqID, pullreqNum), nil).Once()

	gitMock := &mockgit.Interface{}

	// The revert branch must not exist yet.
	gitMock.On("GetBranch", mock.Anything, mock.Anything).
		Return(nil, errors.NotFound("branch not found")).Once()

	var revertParams *git.RevertParams
	gitMock.On("Revert", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			revertParams, _ = args.Get(1).(*git.RevertParams)
		}).
		Return(git.RevertOutput{CommitSHA: sha.Must(revertCommitSHAStr)}, nil).
		Once()

	gitMock.On("GetCommit", mock.Anything, mock.Anything).
		Return(&git.GetCommitOutput{Commit: git.Commit{SHA: sha.Must(revertCommitSHAStr)}}, nil).Once()

	ctrl := revertController(repo, pullreqStore, gitMock)

	out, err := ctrl.Revert(context.Background(), testSession(), "1", pullreqNum, &RevertInput{})
	require.NoError(t, err)
	require.NotNil(t, out)

	require.NotNil(t, revertParams, "git.Revert should have been called")

	assert.Equal(t, revertMergeSHAStr, revertParams.ParentCommitSHA.String(),
		"the revert commit must be put on top of the merge commit")
	assert.Equal(t, revertMergeTargetSHAStr, revertParams.FromCommitSHA.String(),
		"the revert diff must start at the target branch tip that the merge was based on")
	assert.Equal(t, revertMergeSHAStr, revertParams.ToCommitSHA.String(),
		"the revert diff must end at the merge commit")

	// Guard against a regression to the merge base / source branch tip.
	assert.NotEqual(t, revertMergeBaseSHAStr, revertParams.FromCommitSHA.String())
	assert.NotEqual(t, revertSourceSHAStr, revertParams.ToCommitSHA.String())

	assert.Equal(t, "revert-pullreq-7", revertParams.RevertBranch, "default revert branch name")
	assert.Equal(t, "revert-pullreq-7", out.Branch)
	assert.Equal(t, revertCommitSHAStr, out.Commit.SHA.String())

	pullreqStore.AssertExpectations(t)
	gitMock.AssertExpectations(t)
}

// TestRevert_MissingSHAs makes sure that pull requests without the merge SHAs are rejected
// before any git operation. Imported pull requests are merged but have no merge SHA,
// and dereferencing it without the check panics.
func TestRevert_MissingSHAs(t *testing.T) {
	t.Parallel()

	const (
		repoID     = int64(1)
		pullreqID  = int64(55)
		pullreqNum = int64(7)
	)

	tests := []struct {
		name     string
		mutatePR func(pr *types.PullReq)
	}{
		{
			name:     "merge SHA is nil",
			mutatePR: func(pr *types.PullReq) { pr.MergeSHA = nil },
		},
		{
			name:     "merge SHA is empty",
			mutatePR: func(pr *types.PullReq) { pr.MergeSHA = ptr.String("") },
		},
		{
			name:     "merge target SHA is nil",
			mutatePR: func(pr *types.PullReq) { pr.MergeTargetSHA = nil },
		},
		{
			name:     "merge target SHA is empty",
			mutatePR: func(pr *types.PullReq) { pr.MergeTargetSHA = ptr.String("") },
		},
		{
			name:     "pull request is not merged",
			mutatePR: func(pr *types.PullReq) { pr.State = enum.PullReqStateOpen },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repo := &types.RepositoryCore{
				ID: repoID, ParentID: 10, Path: "space/repo", GitUID: "git-uid", State: enum.RepoStateActive,
			}

			pr := mergedPullReq(pullreqID, pullreqNum)
			test.mutatePR(pr)

			pullreqStore := &mockstore.PullReqStore{}
			pullreqStore.On("FindByNumber", repoID, pullreqNum).Return(pr, nil).Once()

			// No expectations are set on the git mock: any git call would fail the test.
			gitMock := &mockgit.Interface{}

			ctrl := revertController(repo, pullreqStore, gitMock)

			_, err := ctrl.Revert(context.Background(), testSession(), "1", pullreqNum, &RevertInput{})
			require.Error(t, err)

			pullreqStore.AssertExpectations(t)
			gitMock.AssertExpectations(t)
		})
	}
}
