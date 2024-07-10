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
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/contextutil"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

type SuggestionReference struct {
	CommentID int64  `json:"comment_id"`
	CheckSum  string `json:"check_sum"`
}

func (e *SuggestionReference) sanitize() error {
	if e.CommentID <= 0 {
		return usererror.BadRequest("Comment ID has to be a positive number.")
	}

	e.CheckSum = strings.TrimSpace(e.CheckSum)
	if e.CheckSum == "" {
		return usererror.BadRequest("Check sum has to be provided.")
	}

	return nil
}

type CommentApplySuggestionsInput struct {
	Suggestions []SuggestionReference `json:"suggestions"`

	Title   string `json:"title"`
	Message string `json:"message"`

	DryRunRules bool `json:"dry_run_rules"`
	BypassRules bool `json:"bypass_rules"`
}

func (i *CommentApplySuggestionsInput) sanitize() error {
	if len(i.Suggestions) == 0 {
		return usererror.BadRequest("No suggestions provided.")
	}
	for _, suggestion := range i.Suggestions {
		if err := suggestion.sanitize(); err != nil {
			return err
		}
	}

	// cleanup title / message (NOTE: git doesn't support white space only)
	i.Title = strings.TrimSpace(i.Title)
	i.Message = strings.TrimSpace(i.Message)

	return nil
}

type CommentApplySuggestionsOutput struct {
	CommitID string `json:"commit_id"`

	DryRunRules    bool                   `json:"dry_run_rules,omitempty"`
	RuleViolations []types.RuleViolations `json:"rule_violations,omitempty"`
}

// CommentApplySuggestions applies suggestions for code comments.
//
//nolint:gocognit,gocyclo,cyclop
func (c *Controller) CommentApplySuggestions(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *CommentApplySuggestionsInput,
) (CommentApplySuggestionsOutput, []types.RuleViolations, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return CommentApplySuggestionsOutput{}, nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return CommentApplySuggestionsOutput{}, nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	if err := in.sanitize(); err != nil {
		return CommentApplySuggestionsOutput{}, nil, err
	}

	// verify branch rules
	isRepoOwner, err := apiauth.IsRepoOwner(ctx, c.authorizer, session, repo)
	if err != nil {
		return CommentApplySuggestionsOutput{}, nil, fmt.Errorf("failed to determine if user is repo owner: %w", err)
	}
	protectionRules, err := c.protectionManager.ForRepository(ctx, repo.ID)
	if err != nil {
		return CommentApplySuggestionsOutput{}, nil, fmt.Errorf(
			"failed to fetch protection rules for the repository: %w", err)
	}
	violations, err := protectionRules.RefChangeVerify(ctx, protection.RefChangeVerifyInput{
		Actor:       &session.Principal,
		AllowBypass: in.BypassRules,
		IsRepoOwner: isRepoOwner,
		Repo:        repo,
		RefAction:   protection.RefActionUpdate,
		RefType:     protection.RefTypeBranch,
		RefNames:    []string{pr.SourceBranch},
	})
	if err != nil {
		return CommentApplySuggestionsOutput{}, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if in.DryRunRules {
		return CommentApplySuggestionsOutput{
			DryRunRules:    true,
			RuleViolations: violations,
		}, nil, nil
	}

	if protection.IsCritical(violations) {
		return CommentApplySuggestionsOutput{}, violations, nil
	}

	actions := []git.CommitFileAction{}
	type activityUpdate struct {
		act      *types.PullReqActivity
		resolve  bool
		checksum string
	}
	activityUpdates := map[int64]activityUpdate{}

	// cache file shas to reduce number of git calls (use commit as some code comments can be temp out of sync)
	getFileSHAKey := func(commitID string, path string) string { return commitID + ":" + path }
	fileSHACache := map[string]sha.SHA{}

	for _, suggestionEntry := range in.Suggestions {
		activity, err := c.getCommentForPR(ctx, pr, suggestionEntry.CommentID)
		if err != nil {
			return CommentApplySuggestionsOutput{}, nil, fmt.Errorf(
				"failed to find activity %d: %w", suggestionEntry.CommentID, err)
		}

		var ccActivity *types.PullReqActivity
		if activity.IsValidCodeComment() {
			ccActivity = activity
		} else if activity.ParentID != nil {
			parentActivity, err := c.activityStore.Find(ctx, *activity.ParentID)
			if err != nil {
				return CommentApplySuggestionsOutput{}, nil, fmt.Errorf(
					"failed to find parent activity %d: %w", *activity.ParentID, err)
			}
			if parentActivity.IsValidCodeComment() {
				ccActivity = parentActivity
			}
		}
		if ccActivity == nil {
			return CommentApplySuggestionsOutput{}, nil, usererror.BadRequest(
				"Only code comments or replies on code comments support applying suggestions.")
		}

		// code comment can't be part of multiple suggestions being applied
		if _, ok := activityUpdates[ccActivity.ID]; ok {
			return CommentApplySuggestionsOutput{}, nil, usererror.BadRequestf(
				"Code comment %d is part of multiple suggestions being applied.",
				ccActivity.ID,
			)
		}

		// retrieve and verify code comment data
		cc := ccActivity.AsCodeComment()

		if cc.Outdated {
			return CommentApplySuggestionsOutput{}, nil, usererror.BadRequest(
				"Suggestions by outdated code comments cannot be applied.")
		}

		// retrieve and verify code comment payload
		payload, err := ccActivity.GetPayload()
		if err != nil {
			return CommentApplySuggestionsOutput{}, nil, fmt.Errorf(
				"failed to get payload of related code comment activity %d: %w", ccActivity.ID, err)
		}
		ccPayload, ok := payload.(*types.PullRequestActivityPayloadCodeComment)
		if !ok {
			return CommentApplySuggestionsOutput{}, nil, fmt.Errorf(
				"provided code comment activity %d has payload of wrong type %T", ccActivity.ID, payload)
		}

		if !ccPayload.LineStartNew || !ccPayload.LineEndNew {
			return CommentApplySuggestionsOutput{}, nil, usererror.BadRequest(
				"Only suggestions on the PR source branch can be applied.")
		}

		suggestions := parseSuggestions(activity.Text)
		var suggestionToApply *suggestion
		for i := range suggestions {
			if strings.EqualFold(suggestions[i].checkSum, suggestionEntry.CheckSum) {
				suggestionToApply = &suggestions[i]
				break
			}
		}
		if suggestionToApply == nil {
			return CommentApplySuggestionsOutput{}, nil, usererror.NotFoundf(
				"No suggestion found for activity %d that matches check sum %q.",
				suggestionEntry.CommentID,
				suggestionEntry.CheckSum,
			)
		}

		// use file-sha for optimistic locking on file to avoid any racing conditions.
		fileSHAKey := getFileSHAKey(cc.SourceSHA, cc.Path)
		fileSHA, ok := fileSHACache[fileSHAKey]
		if !ok {
			node, err := c.git.GetTreeNode(ctx, &git.GetTreeNodeParams{
				ReadParams:          git.CreateReadParams(repo),
				GitREF:              cc.SourceSHA,
				Path:                cc.Path,
				IncludeLatestCommit: false,
			})
			if err != nil {
				return CommentApplySuggestionsOutput{}, nil, fmt.Errorf(
					"failed to read tree node for commit %q path %q: %w",
					cc.SourceSHA,
					cc.Path,
					err,
				)
			}
			// TODO: git api should return sha.SHA type
			fileSHA = sha.Must(node.Node.SHA)
			fileSHACache[fileSHAKey] = fileSHA
		}

		// add suggestion to actions
		actions = append(actions,
			git.CommitFileAction{
				Action: git.PatchTextAction,
				Path:   cc.Path,
				SHA:    fileSHA,
				Payload: []byte(fmt.Sprintf(
					"%d:%d\u0000%s",
					cc.LineNew,
					cc.LineNew+cc.SpanNew,
					suggestionToApply.code,
				)),
			})

		activityUpdates[activity.ID] = activityUpdate{
			act:      activity,
			checksum: suggestionToApply.checkSum,
			resolve:  ccActivity == activity,
		}
		if ccActivity != activity {
			activityUpdates[ccActivity.ID] = activityUpdate{
				act:     ccActivity,
				resolve: true,
			}
		}
	}

	// we want to complete the operation independent of request cancel - start with new, time restricted context.
	// TODO: This is a small change to reduce likelihood of dirty state (e.g. git work done but db canceled).
	// We still require a proper solution to handle an application crash or very slow execution times
	const timeout = 1 * time.Minute
	ctx, cancel := context.WithTimeout(
		contextutil.WithNewValues(context.Background(), ctx),
		timeout,
	)
	defer cancel()

	// Create internal write params. Note: This will skip the pre-commit protection rules check.
	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return CommentApplySuggestionsOutput{}, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	// backfill title if not provided (keeping it basic for now, user can provide more detailed title)
	if in.Title == "" {
		in.Title = "Apply code review suggestions"
	}

	now := time.Now()
	commitOut, err := c.git.CommitFiles(ctx, &git.CommitFilesParams{
		WriteParams:   writeParams,
		Title:         in.Title,
		Message:       in.Message,
		Branch:        pr.SourceBranch,
		Committer:     identityFromPrincipalInfo(*bootstrap.NewSystemServiceSession().Principal.ToPrincipalInfo()),
		CommitterDate: &now,
		Author:        identityFromPrincipalInfo(*session.Principal.ToPrincipalInfo()),
		AuthorDate:    &now,
		Actions:       actions,
	})
	if err != nil {
		return CommentApplySuggestionsOutput{}, nil, fmt.Errorf("failed to commit changes: %w", err)
	}

	// update activities (use UpdateOptLock as it can have racing condition with comment migration)
	resolved := ptr.Of(now.UnixMilli())
	resolvedBy := &session.Principal.ID
	resolvedActivities := map[int64]struct{}{}
	for _, update := range activityUpdates {
		_, err = c.activityStore.UpdateOptLock(ctx, update.act, func(act *types.PullReqActivity) error {
			// only resolve where required (can happen in case of parallel resolution of activity)
			if update.resolve && act.Resolved == nil {
				act.Resolved = resolved
				act.ResolvedBy = resolvedBy
				resolvedActivities[act.ID] = struct{}{}
			} else {
				delete(resolvedActivities, act.ID)
			}

			if update.checksum != "" {
				act.UpdateMetadata(types.WithPullReqActivitySuggestionsMetadataUpdate(
					func(s *types.PullReqActivitySuggestionsMetadata) {
						s.AppliedCheckSum = update.checksum
						s.AppliedCommitSHA = commitOut.CommitID.String()
					}))
			}

			return nil
		})
		if err != nil {
			// best effort - commit already happened
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to update activity %d after applying suggestions", update.act.ID)
		}
	}

	// This is a best effort approach as in case of sqlite a transaction is likely to be blocked
	// by parallel event-triggered db writes from the above commit.
	// WARNING: This could cause the count to diverge (similar to create / delete).
	// TODO: Use transaction once sqlite issue has been addressed.
	pr, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		pr.UnresolvedCount -= len(resolvedActivities)
		return nil
	})
	if err != nil {
		return CommentApplySuggestionsOutput{}, nil,
			fmt.Errorf("failed to update pull request's unresolved comment count: %w", err)
	}

	if err = c.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypePullRequestUpdated, pr); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to publish PR changed event")
	}

	return CommentApplySuggestionsOutput{
		CommitID:       commitOut.CommitID.String(),
		RuleViolations: violations,
	}, nil, nil
}
