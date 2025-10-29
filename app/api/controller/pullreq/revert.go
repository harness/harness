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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type RevertInput struct {
	Title   string `json:"title"`
	Message string `json:"message"`

	// RevertBranch is the name of new branch that will be created on which the revert commit will be put.
	// It's optional, if no value has been provided the default ("revert-pullreq-<number>") would be used.
	RevertBranch string `json:"revert_branch"`
}

func (in *RevertInput) sanitize() {
	in.Title = strings.TrimSpace(in.Title)
	in.Message = strings.TrimSpace(in.Message)
	in.RevertBranch = strings.TrimSpace(in.RevertBranch)
}

func (c *Controller) Revert(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	in *RevertInput,
) (*types.RevertResponse, error) {
	in.sanitize()

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	if pr.State != enum.PullReqStateMerged {
		return nil, usererror.BadRequest("Only merged pull requests can be reverted.")
	}

	readParams := git.CreateReadParams(repo)
	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	revertBranch := in.RevertBranch
	if revertBranch == "" {
		revertBranch = "revert-pullreq-" + strconv.FormatInt(pullreqNum, 10)
	}

	_, err = c.git.GetBranch(ctx, &git.GetBranchParams{
		ReadParams: readParams,
		BranchName: revertBranch,
	})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get revert branch: %w", err)
	}
	if err == nil {
		return nil, errors.InvalidArgumentf("Branch %q already exists.", revertBranch)
	}

	title := in.Title
	message := in.Message
	if title == "" {
		title = fmt.Sprintf("Revert Pull Request #%d %q", pullreqNum, pr.Title)
	}
	commitMessage := git.CommitMessage(title, message)

	author := controller.IdentityFromPrincipalInfo(*session.Principal.ToPrincipalInfo())
	committer := controller.SystemServicePrincipalInfo()

	now := time.Now()

	result, err := c.git.Revert(ctx, &git.RevertParams{
		WriteParams:     writeParams,
		ParentCommitSHA: sha.Must(*pr.MergeSHA),
		FromCommitSHA:   sha.Must(pr.MergeBaseSHA),
		ToCommitSHA:     sha.Must(pr.SourceSHA),
		RevertBranch:    revertBranch,
		Message:         commitMessage,
		Committer:       committer,
		CommitterDate:   &now,
		Author:          author,
		AuthorDate:      &now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to revert pull request: %w", err)
	}

	gitCommit, err := c.git.GetCommit(ctx, &git.GetCommitParams{
		ReadParams: readParams,
		Revision:   result.CommitSHA.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get revert commit: %w", err)
	}

	return &types.RevertResponse{
		Branch: revertBranch,
		Commit: *controller.MapCommit(&gitCommit.Commit),
	}, nil
}
