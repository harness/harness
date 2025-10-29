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

	"github.com/gotidy/ptr"
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

	if pr.State == enum.PullReqSortMerged {
		return nil, errors.InvalidArgument("Pull request is already merged.")
	}

	if _, err = c.verifyBranchExistence(ctx, repo, in.BranchName); err != nil {
		return nil,
			fmt.Errorf("failed to verify branch existence: %w", err)
	}

	if pr.TargetBranch == in.BranchName {
		return pr, nil
	}

	if pr.SourceRepoID != nil && pr.TargetRepoID == *pr.SourceRepoID && pr.SourceBranch == in.BranchName {
		return nil,
			errors.InvalidArgumentf("Source branch %q is same as new target branch", pr.SourceBranch)
	}

	readParams := git.CreateReadParams(repo)

	targetRef, err := c.git.GetRef(ctx, git.GetRefParams{
		ReadParams: readParams,
		Name:       in.BranchName,
		Type:       gitenum.RefTypeBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target branch reference: %w", err)
	}

	targetSHA := targetRef.SHA

	mergeBase, err := c.git.MergeBase(ctx, git.MergeBaseParams{
		ReadParams: readParams,
		Ref1:       pr.SourceSHA,
		Ref2:       targetSHA.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find merge base: %w", err)
	}

	if mergeBase.MergeBaseSHA.String() == pr.SourceSHA {
		return nil, usererror.BadRequest("The source branch doesn't contain any new commits")
	}

	diffStats, err := c.git.DiffStats(ctx, &git.DiffParams{
		ReadParams: readParams,
		BaseRef:    pr.MergeBaseSHA,
		HeadRef:    pr.SourceSHA,
	})
	if err != nil {
		return nil, fmt.Errorf("failed get diff stats: %w", err)
	}

	oldTargetBranch := pr.TargetBranch
	oldMergeBaseSHA := pr.MergeBaseSHA

	pr, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		pr.MergeSHA = nil
		pr.MarkAsMergeUnchecked()

		pr.MergeBaseSHA = mergeBase.MergeBaseSHA.String()
		pr.MergeTargetSHA = ptr.String(targetSHA.String())
		pr.TargetBranch = in.BranchName
		pr.Stats.DiffStats = types.NewDiffStats(
			diffStats.Commits,
			diffStats.FilesChanged,
			diffStats.Additions,
			diffStats.Deletions,
		)

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
