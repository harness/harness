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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type ChangeTargetBranchInput struct {
	BranchName string `json:"branch_name"`
}

func (c *Controller) ChangeTargetBranch(ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	in *ChangeTargetBranchInput,
) (*types.PullReq, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	if _, err = c.verifyBranchExistence(ctx, repo, in.BranchName); err != nil {
		return nil,
			fmt.Errorf("failed to verify branch existence: %w", err)
	}

	if pr.TargetBranch == in.BranchName {
		return pr, nil
	}
	if pr.SourceBranch == in.BranchName {
		return nil,
			errors.InvalidArgument("source branch %q is same as new target branch", pr.SourceBranch)
	}

	ref1, err := git.GetRefPath(pr.SourceBranch, gitenum.RefTypeBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get ref path: %w", err)
	}
	ref2, err := git.GetRefPath(in.BranchName, gitenum.RefTypeBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get ref path: %w", err)
	}
	mergeBase, err := c.git.MergeBase(ctx, git.MergeBaseParams{
		ReadParams: git.ReadParams{RepoUID: repo.GitUID},
		Ref1:       ref1,
		Ref2:       ref2,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find merge base: %w", err)
	}

	if mergeBase.MergeBaseSHA.String() == pr.SourceSHA {
		return nil,
			usererror.BadRequest("The source branch doesn't contain any new commits")
	}

	oldTargetBranch := pr.TargetBranch
	oldMergeBaseSHA := pr.MergeBaseSHA

	_, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		// clear merge and stats related fields
		pr.MergeSHA = nil
		pr.MergeTargetSHA = nil
		pr.Stats.DiffStats = types.DiffStats{}

		pr.MergeBaseSHA = mergeBase.MergeBaseSHA.String()
		pr.TargetBranch = in.BranchName

		pr.MarkAsMergeUnchecked()

		pr.ActivitySeq++

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update PR target branch in db with error: %w", err)
	}

	_, err = c.activityStore.CreateWithPayload(
		ctx, pr, session.Principal.ID,
		&types.PullRequestActivityPayloadBranchChangeTarget{
			Old: oldTargetBranch,
			New: in.BranchName,
		},
		nil,
	)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to write pull request activity for successful branch restore")
	}

	err = c.instrumentation.Track(ctx, instrument.Event{
		Type:      instrument.EventTypeChangeTargetBranch,
		Principal: session.Principal.ToPrincipalInfo(),
		Path:      repo.Path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:   repo.ID,
			instrument.PropertyRepositoryName: repo.Identifier,
			instrument.PropertyTargetBranch:   in.BranchName,
			instrument.PropertyPullRequestID:  pr.Number,
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for create branch operation: %s", err)
	}

	c.eventReporter.TargetBranchChanged(ctx, &pullreqevents.TargetBranchChangedPayload{
		Base:            eventBase(pr, &session.Principal),
		SourceSHA:       pr.SourceSHA,
		OldTargetBranch: oldTargetBranch,
		NewTargetBranch: in.BranchName,
		OldMergeBaseSHA: oldMergeBaseSHA,
		NewMergeBaseSHA: mergeBase.MergeBaseSHA.String(),
	})

	return pr, nil
}
