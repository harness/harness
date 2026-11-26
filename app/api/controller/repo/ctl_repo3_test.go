// Copyright 2026 Harness, Inc.
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

package repo

import (
	"context"
	"testing"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const ctlCommitSHA = "0123456789abcdef0123456789abcdef01234567"

func TestCtlRepoController_CreateBranch(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	// Dry run does not touch git.
	out, violations, err := e.ctrl.CreateBranch(
		ctx, s, "root/repo1", &CreateBranchInput{Name: "feature", DryRunRules: true})
	require.NoError(t, err)
	require.Nil(t, violations)
	require.True(t, out.DryRunRules)

	var params *git.CreateBranchParams
	e.git.On("CreateBranch", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		var ok bool
		params, ok = args.Get(1).(*git.CreateBranchParams)
		require.True(t, ok)
	}).Return(&git.CreateBranchOutput{Branch: git.Branch{Name: "feature", SHA: sha.Must(ctlCommitSHA)}}, nil).Once()

	out, violations, err = e.ctrl.CreateBranch(ctx, s, "root/repo1", &CreateBranchInput{Name: "feature"})
	require.NoError(t, err)
	require.Nil(t, violations)
	require.Equal(t, "feature", out.Branch.Name)
	require.Equal(t, "main", params.Target, "target defaults to the default branch")
	require.Equal(t, e.repo.GitUID, params.RepoUID)

	e.git.On("CreateBranch", mock.Anything, mock.Anything).Return(nil, errCtlGit).Once()
	_, _, err = e.ctrl.CreateBranch(ctx, s, "root/repo1", &CreateBranchInput{Name: "broken", Target: "abc"})
	require.ErrorIs(t, err, errCtlGit)
}

func TestCtlRepoController_CreateAndDeleteTag(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	out, _, err := e.ctrl.CreateCommitTag(ctx, s, "root/repo1", &CreateCommitTagInput{Name: "v1", DryRunRules: true})
	require.NoError(t, err)
	require.True(t, out.DryRunRules)

	e.git.On("CreateCommitTag", mock.Anything, mock.Anything).Return(&git.CreateCommitTagOutput{
		CommitTag: git.CommitTag{Name: "v1", SHA: sha.Must(ctlCommitSHA), IsAnnotated: true, Message: "msg"},
	}, nil).Once()
	out, violations, err := e.ctrl.CreateCommitTag(ctx, s, "root/repo1", &CreateCommitTagInput{
		Name: "v1", Message: "msg",
	})
	require.NoError(t, err)
	require.Nil(t, violations)
	require.Equal(t, "v1", out.CommitTag.Name)
	require.True(t, out.CommitTag.IsAnnotated)

	e.git.On("CreateCommitTag", mock.Anything, mock.Anything).Return(nil, errCtlGit).Once()
	_, _, err = e.ctrl.CreateCommitTag(ctx, s, "root/repo1", &CreateCommitTagInput{Name: "v2", Target: "abc"})
	require.ErrorIs(t, err, errCtlGit)

	dry, _, err := e.ctrl.DeleteCommitTag(ctx, s, "root/repo1", "v1", false, true)
	require.NoError(t, err)
	require.True(t, dry.DryRunRules)

	e.git.On("DeleteTag", mock.Anything, mock.Anything).Return(nil).Once()
	_, violations, err = e.ctrl.DeleteCommitTag(ctx, s, "root/repo1", "v1", false, false)
	require.NoError(t, err)
	require.Nil(t, violations)

	e.git.On("DeleteTag", mock.Anything, mock.Anything).Return(errCtlGit).Once()
	_, _, err = e.ctrl.DeleteCommitTag(ctx, s, "root/repo1", "v1", false, false)
	require.ErrorIs(t, err, errCtlGit)
}

func TestCtlRepoController_DefaultBranchDeleteBlocked(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)

	_, _, err := e.ctrl.DeleteBranch(ctx, e.session(), "root/repo1", "main", false, false)
	require.Error(t, err, "the default branch can't be deleted")
}
