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
	"fmt"
	"strconv"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type PullReqGitInput struct {
	Author        *git.Identity
	Committer     *git.Identity
	RefUpdates    []git.RefUpdate
	CommitMessage string
	SourceSHA     sha.SHA
}

func (*Service) PreparePullReqMergeInput(
	pr *types.PullReq,
	sourceRepo *types.RepositoryCore,
	targetSHA sha.SHA,
	principal *types.PrincipalInfo,
	method enum.MergeMethod,
	title string,
	message string,
) (PullReqGitInput, error) {
	var author *git.Identity

	switch method {
	case enum.MergeMethodMerge:
		author = controller.IdentityFromPrincipalInfo(*principal)
	case enum.MergeMethodSquash:
		author = controller.IdentityFromPrincipalInfo(pr.Author)
	case enum.MergeMethodRebase, enum.MergeMethodFastForward:
		author = nil // Not important for these merge methods: the author info in the commits will be preserved.
	}

	var committer *git.Identity

	switch method {
	case enum.MergeMethodMerge, enum.MergeMethodSquash:
		committer = controller.SystemServicePrincipalInfo()
	case enum.MergeMethodRebase:
		committer = controller.IdentityFromPrincipalInfo(*principal)
	case enum.MergeMethodFastForward:
		committer = nil // Not important for fast-forward merge
	}

	if title == "" {
		switch method {
		case enum.MergeMethodMerge:
			if sourceRepo == nil {
				title = fmt.Sprintf("Merge branch '%s' of deleted fork (#%d)",
					pr.SourceBranch, pr.Number)
			} else {
				title = fmt.Sprintf("Merge branch '%s' of %s (#%d)", pr.SourceBranch,
					sourceRepo.Path, pr.Number)
			}
		case enum.MergeMethodSquash:
			title = fmt.Sprintf("%s (#%d)", pr.Title, pr.Number)
		case enum.MergeMethodRebase, enum.MergeMethodFastForward:
			// Not used.
		}
	}

	// create merge commit(s)

	sourceBranchSHA, err := sha.New(pr.SourceSHA)
	if err != nil {
		return PullReqGitInput{}, fmt.Errorf("failed to convert source SHA: %w", err)
	}

	refTargetBranch, err := git.GetRefPath(pr.TargetBranch, gitenum.RefTypeBranch)
	if err != nil {
		return PullReqGitInput{}, fmt.Errorf("failed to generate target branch ref name: %w", err)
	}

	prNumber := strconv.FormatInt(pr.Number, 10)

	refPullReqHead, err := git.GetRefPath(prNumber, gitenum.RefTypePullReqHead)
	if err != nil {
		return PullReqGitInput{}, fmt.Errorf("failed to generate pull request head ref name: %w", err)
	}

	refPullReqMerge, err := git.GetRefPath(prNumber, gitenum.RefTypePullReqMerge)
	if err != nil {
		return PullReqGitInput{}, fmt.Errorf("failed to generate pull requert merge ref name: %w", err)
	}

	refUpdates := make([]git.RefUpdate, 0, 4)

	// Update the target branch to the result of the merge.
	refUpdates = append(refUpdates, git.RefUpdate{
		Name: refTargetBranch,
		Old:  targetSHA,
		New:  sha.SHA{}, // update to the result of the merge.
	})

	// Make sure the PR head ref points to the correct commit after the merge.
	refUpdates = append(refUpdates, git.RefUpdate{
		Name: refPullReqHead,
		Old:  sha.SHA{}, // don't care about the old value.
		New:  sourceBranchSHA,
	})

	// Delete the PR merge reference.
	refUpdates = append(refUpdates, git.RefUpdate{
		Name: refPullReqMerge,
		Old:  sha.SHA{},
		New:  sha.Nil,
	})

	return PullReqGitInput{
		Author:        author,
		Committer:     committer,
		RefUpdates:    refUpdates,
		CommitMessage: git.CommitMessage(title, message),
		SourceSHA:     sourceBranchSHA,
	}, nil
}
