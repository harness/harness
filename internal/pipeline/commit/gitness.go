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

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/controller"
	"github.com/harness/gitness/types"
)

type service struct {
	gitRPCClient gitrpc.Interface
}

func new(gitRPCClient gitrpc.Interface) CommitService {
	return &service{gitRPCClient: gitRPCClient}
}

// FindRef finds information about a commit in gitness for the git ref.
// This is using the branch only as the ref at the moment, can be changed
// when needed to take any ref (like sha, tag).
func (f *service) FindRef(
	ctx context.Context,
	repo *types.Repository,
	branch string,
) (*types.Commit, error) {
	readParams := gitrpc.ReadParams{
		RepoUID: repo.GitUID,
	}
	branchOutput, err := f.gitRPCClient.GetBranch(ctx, &gitrpc.GetBranchParams{
		ReadParams: readParams,
		BranchName: branch,
	})
	if err != nil {
		return nil, err
	}

	// convert the RPC commit output to a types.Commit.
	return controller.MapCommit(branchOutput.Branch.Commit)
}

// FindCommit finds information about a commit in gitness for the git SHA
func (f *service) FindCommit(
	ctx context.Context,
	repo *types.Repository,
	sha string,
) (*types.Commit, error) {
	readParams := gitrpc.ReadParams{
		RepoUID: repo.GitUID,
	}
	commitOutput, err := f.gitRPCClient.GetCommit(ctx, &gitrpc.GetCommitParams{
		ReadParams: readParams,
		SHA:        sha,
	})
	if err != nil {
		return nil, err
	}

	// convert the RPC commit output to a types.Commit.
	return controller.MapCommit(&commitOutput.Commit)
}
