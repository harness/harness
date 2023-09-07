// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
