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

package commit

import (
	"context"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
)

type service struct {
	git git.Interface
}

func newService(git git.Interface) Service {
	return &service{git: git}
}

// FindRef finds information about a commit in Harness for the git ref.
// This is using the branch only as the ref at the moment, can be changed
// when needed to take any ref (like sha, tag).
func (f *service) FindRef(
	ctx context.Context,
	repo *types.RepositoryCore,
	branch string,
) (*types.Commit, error) {
	readParams := git.ReadParams{
		RepoUID: repo.GitUID,
	}
	branchOutput, err := f.git.GetBranch(ctx, &git.GetBranchParams{
		ReadParams: readParams,
		BranchName: branch,
	})
	if err != nil {
		return nil, err
	}

	// convert the RPC commit output to a types.Commit.
	return controller.MapCommit(branchOutput.Branch.Commit)
}

// FindCommit finds information about a commit in Harness for the git SHA.
func (f *service) FindCommit(
	ctx context.Context,
	repo *types.RepositoryCore,
	rawSHA string,
) (*types.Commit, error) {
	readParams := git.ReadParams{
		RepoUID: repo.GitUID,
	}
	commitOutput, err := f.git.GetCommit(ctx, &git.GetCommitParams{
		ReadParams: readParams,
		Revision:   rawSHA,
	})
	if err != nil {
		return nil, err
	}

	// convert the RPC commit output to a types.Commit.
	return controller.MapCommit(&commitOutput.Commit)
}
