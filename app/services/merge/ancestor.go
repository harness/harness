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

package merge

import (
	"context"
	"fmt"
	"strconv"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
)

// GetTargetSourceSHAs returns the current target commit SHA, source commit SHA and
// if the latest commit on target branch is an ancestor of the latest commit on the source branch.
func (s *Service) GetTargetSourceSHAs(
	ctx context.Context,
	targetRepo *types.RepositoryCore,
	pr *types.PullReq,
) (sha.SHA, sha.SHA, bool, error) {
	getHeadRef, err := s.git.GetRef(ctx, git.GetRefParams{
		ReadParams: git.ReadParams{RepoUID: targetRepo.GitUID},
		Name:       strconv.FormatInt(pr.Number, 10),
		Type:       gitenum.RefTypePullReqHead,
	})
	if err != nil {
		return sha.SHA{}, sha.SHA{}, false, fmt.Errorf("failed to get pull request head ref: %w", err)
	}

	sourceSHA, err := sha.New(pr.SourceSHA)
	if err != nil {
		return sha.SHA{}, sha.SHA{}, false, fmt.Errorf("pull request source SHA not valid: %w", err)
	}

	if getHeadRef.SHA != sourceSHA {
		return sha.SHA{}, sha.SHA{}, false,
			usererror.BadRequest("The pull request head ref doesn't match the source SHA.")
	}

	targetBranch, err := s.git.GetBranch(ctx, &git.GetBranchParams{
		ReadParams: git.ReadParams{RepoUID: targetRepo.GitUID},
		BranchName: pr.TargetBranch,
	})
	if err != nil {
		return sha.SHA{}, sha.SHA{}, false, fmt.Errorf("failed to get pull request target branch: %w", err)
	}

	targetSHA := targetBranch.Branch.SHA

	isAncestorResponse, err := s.git.IsAncestor(ctx, git.IsAncestorParams{
		ReadParams:          git.ReadParams{RepoUID: targetRepo.GitUID},
		AncestorCommitSHA:   targetSHA,
		DescendantCommitSHA: sourceSHA,
	})
	if err != nil {
		return sha.SHA{}, sha.SHA{}, false, fmt.Errorf("failed to check pull request commit ancestry: %w", err)
	}

	return targetSHA, sourceSHA, isAncestorResponse.Ancestor, nil
}
