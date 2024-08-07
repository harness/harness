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

package githook

import (
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	events "github.com/harness/gitness/app/events/git"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

const (
	// gitReferenceNamePrefixBranch is the prefix of references of type branch.
	gitReferenceNamePrefixBranch = "refs/heads/"

	// gitReferenceNamePrefixTag is the prefix of references of type tag.
	gitReferenceNamePrefixTag = "refs/tags/"

	// gitReferenceNamePrefixTag is the prefix of pull req references.
	gitReferenceNamePullReq = "refs/pullreq/"
)

// PostReceive executes the post-receive hook for a git repository.
func (c *Controller) PostReceive(
	ctx context.Context,
	rgit RestrictedGIT,
	session *auth.Session,
	in types.GithookPostReceiveInput,
) (hook.Output, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, in.RepoID, enum.PermissionRepoPush)
	if err != nil {
		return hook.Output{}, err
	}
	// create output object and have following messages fill its messages
	out := hook.Output{}

	// update default branch based on ref update info on empty repos.
	// as the branch could be different than the configured default value.
	c.handleEmptyRepoPush(ctx, repo, in.PostReceiveInput, &out)

	// report ref events if repo is in an active state (best effort)
	if repo.State == enum.RepoStateActive {
		c.reportReferenceEvents(ctx, rgit, repo, in.PrincipalID, in.PostReceiveInput)
	}

	// handle branch updates related to PRs - best effort
	c.handlePRMessaging(ctx, repo, in.PostReceiveInput, &out)

	err = c.postReceiveExtender.Extend(ctx, rgit, session, repo, in, &out)
	if err != nil {
		return hook.Output{}, fmt.Errorf("failed to extend post-receive hook: %w", err)
	}

	return out, nil
}

// reportReferenceEvents is reporting reference events to the event system.
// NOTE: keep best effort for now as it doesn't change the outcome of the git operation.
// TODO: in the future we might want to think about propagating errors so user is aware of events not being triggered.
func (c *Controller) reportReferenceEvents(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.Repository,
	principalID int64,
	in hook.PostReceiveInput,
) {
	for _, refUpdate := range in.RefUpdates {
		switch {
		case strings.HasPrefix(refUpdate.Ref, gitReferenceNamePrefixBranch):
			c.reportBranchEvent(ctx, rgit, repo, principalID, in.Environment, refUpdate)
		case strings.HasPrefix(refUpdate.Ref, gitReferenceNamePrefixTag):
			c.reportTagEvent(ctx, repo, principalID, refUpdate)
		default:
			// Ignore any other references in post-receive
		}
	}
}

func (c *Controller) reportBranchEvent(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.Repository,
	principalID int64,
	env hook.Environment,
	branchUpdate hook.ReferenceUpdate,
) {
	switch {
	case branchUpdate.Old.IsNil():
		c.gitReporter.BranchCreated(ctx, &events.BranchCreatedPayload{
			RepoID:      repo.ID,
			PrincipalID: principalID,
			Ref:         branchUpdate.Ref,
			SHA:         branchUpdate.New.String(),
		})
	case branchUpdate.New.IsNil():
		c.gitReporter.BranchDeleted(ctx, &events.BranchDeletedPayload{
			RepoID:      repo.ID,
			PrincipalID: principalID,
			Ref:         branchUpdate.Ref,
			SHA:         branchUpdate.Old.String(),
		})
	default:
		result, err := rgit.IsAncestor(ctx, git.IsAncestorParams{
			ReadParams: git.ReadParams{
				RepoUID:             repo.GitUID,
				AlternateObjectDirs: env.AlternateObjectDirs,
			},
			AncestorCommitSHA:   branchUpdate.Old,
			DescendantCommitSHA: branchUpdate.New,
		})
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("ref", branchUpdate.Ref).
				Msg("failed to check ancestor")
		}
		// In case of an error consider this a forced update. In post-update the branch has already been updated,
		// so there's less harm in declaring the update as forced. A force update event might trigger some additional
		// operations that aren't required for ordinary updates (force pushes alter the commit history of a branch).
		forced := err != nil || !result.Ancestor
		c.gitReporter.BranchUpdated(ctx, &events.BranchUpdatedPayload{
			RepoID:      repo.ID,
			PrincipalID: principalID,
			Ref:         branchUpdate.Ref,
			OldSHA:      branchUpdate.Old.String(),
			NewSHA:      branchUpdate.New.String(),
			Forced:      forced,
		})
	}
}

func (c *Controller) reportTagEvent(
	ctx context.Context,
	repo *types.Repository,
	principalID int64,
	tagUpdate hook.ReferenceUpdate,
) {
	switch {
	case tagUpdate.Old.IsNil():
		c.gitReporter.TagCreated(ctx, &events.TagCreatedPayload{
			RepoID:      repo.ID,
			PrincipalID: principalID,
			Ref:         tagUpdate.Ref,
			SHA:         tagUpdate.New.String(),
		})
	case tagUpdate.New.IsNil():
		c.gitReporter.TagDeleted(ctx, &events.TagDeletedPayload{
			RepoID:      repo.ID,
			PrincipalID: principalID,
			Ref:         tagUpdate.Ref,
			SHA:         tagUpdate.Old.String(),
		})
	default:
		c.gitReporter.TagUpdated(ctx, &events.TagUpdatedPayload{
			RepoID:      repo.ID,
			PrincipalID: principalID,
			Ref:         tagUpdate.Ref,
			OldSHA:      tagUpdate.Old.String(),
			NewSHA:      tagUpdate.New.String(),
			// tags can only be force updated!
			Forced: true,
		})
	}
}

// handlePRMessaging checks any single branch push for pr information and returns an according response if needed.
// TODO: If it is a new branch, or an update on a branch without any PR, it also sends out an SSE for pr creation.
func (c *Controller) handlePRMessaging(
	ctx context.Context,
	repo *types.Repository,
	in hook.PostReceiveInput,
	out *hook.Output,
) {
	// skip anything that was a batch push / isn't branch related / isn't updating/creating a branch.
	if len(in.RefUpdates) != 1 ||
		!strings.HasPrefix(in.RefUpdates[0].Ref, gitReferenceNamePrefixBranch) ||
		in.RefUpdates[0].New.IsNil() {
		return
	}

	// for now we only care about first branch that was pushed.
	branchName := in.RefUpdates[0].Ref[len(gitReferenceNamePrefixBranch):]

	c.suggestPullRequest(ctx, repo, branchName, out)

	// TODO: store latest pushed branch for user in cache and send out SSE
}

func (c *Controller) suggestPullRequest(
	ctx context.Context,
	repo *types.Repository,
	branchName string,
	out *hook.Output,
) {
	if branchName == repo.DefaultBranch {
		// Don't suggest a pull request if this is a push to the default branch.
		return
	}

	// do we have a PR related to it?
	prs, err := c.pullreqStore.List(ctx, &types.PullReqFilter{
		Page: 1,
		// without forks we expect at most one PR (keep 2 to not break when forks are introduced)
		Size:         2,
		SourceRepoID: repo.ID,
		SourceBranch: branchName,
		// we only care about open PRs - merged/closed will lead to "create new PR" message
		States: []enum.PullReqState{enum.PullReqStateOpen},
		Order:  enum.OrderAsc,
		Sort:   enum.PullReqSortCreated,
	})
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf(
			"failed to find pullrequests for branch '%s' originating from repo '%s'",
			branchName,
			repo.Path,
		)
		return
	}

	// for already existing PRs, print them to users terminal for easier access.
	if len(prs) > 0 {
		msgs := make([]string, 2*len(prs)+1)
		msgs[0] = fmt.Sprintf("Branch %q has open PRs:", branchName)
		for i, pr := range prs {
			msgs[2*i+1] = fmt.Sprintf("  (#%d) %s", pr.Number, pr.Title)
			msgs[2*i+2] = "    " + c.urlProvider.GenerateUIPRURL(ctx, repo.Path, pr.Number)
		}
		out.Messages = append(out.Messages, msgs...)
		return
	}

	// this is a new PR!
	out.Messages = append(out.Messages,
		fmt.Sprintf("Create a pull request for %q by visiting:", branchName),
		"  "+c.urlProvider.GenerateUICompareURL(ctx, repo.Path, repo.DefaultBranch, branchName),
	)
}

// handleEmptyRepoPush updates repo default branch on empty repos if push contains branches.
func (c *Controller) handleEmptyRepoPush(
	ctx context.Context,
	repo *types.Repository,
	in hook.PostReceiveInput,
	out *hook.Output,
) {
	if !repo.IsEmpty {
		return
	}

	var newDefaultBranch string
	// update default branch if corresponding branch does not exist
	for _, refUpdate := range in.RefUpdates {
		if strings.HasPrefix(refUpdate.Ref, gitReferenceNamePrefixBranch) && !refUpdate.New.IsNil() {
			branchName := refUpdate.Ref[len(gitReferenceNamePrefixBranch):]
			if branchName == repo.DefaultBranch {
				newDefaultBranch = branchName
				break
			}
			// use the first pushed branch if default branch is not present.
			if newDefaultBranch == "" {
				newDefaultBranch = branchName
			}
		}
	}
	if newDefaultBranch == "" {
		out.Error = ptr.String(usererror.ErrEmptyRepoNeedsBranch.Error())
		return
	}

	oldName := repo.DefaultBranch
	var err error
	repo, err = c.repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
		r.IsEmpty = false
		r.DefaultBranch = newDefaultBranch
		return nil
	})
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("failed to update the repo default branch to %s and is_empty to false",
			newDefaultBranch)
		return
	}

	if repo.DefaultBranch != oldName {
		c.repoReporter.DefaultBranchUpdated(ctx, &repoevents.DefaultBranchUpdatedPayload{
			RepoID:      repo.ID,
			PrincipalID: bootstrap.NewSystemServiceSession().Principal.ID,
			OldName:     oldName,
			NewName:     repo.DefaultBranch,
		})
	}
}
