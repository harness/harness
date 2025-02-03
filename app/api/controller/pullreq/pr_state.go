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
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type StateInput struct {
	State   enum.PullReqState `json:"state"`
	IsDraft bool              `json:"is_draft"`
}

func (in *StateInput) Check() error {
	state, ok := in.State.Sanitize() // Sanitize will pass through also merged state, so we must check later for it.
	if !ok {
		return usererror.BadRequest(fmt.Sprintf("Allowed states are: %s and %s",
			enum.PullReqStateOpen, enum.PullReqStateClosed))
	}

	in.State = state

	if in.State == enum.PullReqStateMerged {
		return usererror.BadRequest("Pull requests can't be merged with this API")
	}

	return nil
}

// State updates the pull request's current state.
//
//nolint:gocognit,funlen
func (c *Controller) State(ctx context.Context,
	session *auth.Session, repoRef string, pullreqNum int64, in *StateInput,
) (*types.PullReq, error) {
	if err := in.Check(); err != nil {
		return nil, err
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	id := pr.ID

	sourceRepo := targetRepo
	if pr.SourceRepoID != pr.TargetRepoID {
		sourceRepo, err = c.repoStore.Find(ctx, pr.SourceRepoID)
		if err != nil {
			return nil, fmt.Errorf("failed to get source repo by id: %w", err)
		}

		if err = apiauth.CheckRepo(ctx, c.authorizer, session, sourceRepo,
			enum.PermissionRepoView); err != nil {
			return nil, fmt.Errorf("failed to acquire access to source repo: %w", err)
		}
	}

	if pr.State == enum.PullReqStateMerged {
		return nil, usererror.BadRequest("Merged pull requests can't be modified.")
	}

	if pr.State == in.State && in.IsDraft == pr.IsDraft {
		return pr, nil // no changes are necessary: state is the same and is_draft hasn't change
	}

	oldState := pr.State
	oldDraft := pr.IsDraft

	type change int
	const (
		changeReopen change = iota + 1
		changeClose
	)

	var sourceSHA sha.SHA
	var mergeBaseSHA sha.SHA
	var stateChange change

	//nolint:nestif // refactor if needed
	if pr.State != enum.PullReqStateOpen && in.State == enum.PullReqStateOpen {
		if sourceSHA, err = c.verifyBranchExistence(ctx, sourceRepo, pr.SourceBranch); err != nil {
			return nil, err
		}

		if _, err = c.verifyBranchExistence(ctx, targetRepo, pr.TargetBranch); err != nil {
			return nil, err
		}

		err = c.checkIfAlreadyExists(ctx, pr.TargetRepoID, pr.SourceRepoID, pr.TargetBranch, pr.SourceBranch)
		if err != nil {
			return nil, err
		}

		mergeBaseResult, err := c.git.MergeBase(ctx, git.MergeBaseParams{
			ReadParams: git.ReadParams{RepoUID: sourceRepo.GitUID},
			Ref1:       pr.SourceBranch,
			Ref2:       pr.TargetBranch,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to find merge base: %w", err)
		}

		mergeBaseSHA = mergeBaseResult.MergeBaseSHA

		stateChange = changeReopen
	} else if pr.State == enum.PullReqStateOpen && in.State != enum.PullReqStateOpen {
		stateChange = changeClose
	}

	targetWriteParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, targetRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	err = controller.TxOptLock(ctx, c.tx, func(ctx context.Context) error {
		if pr == nil {
			pr, err = c.pullreqStore.Find(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to find pull request by id: %w", err)
			}
		}

		pr.State = in.State
		pr.IsDraft = in.IsDraft

		switch stateChange {
		case changeClose:
			nowMilli := time.Now().UnixMilli()

			// clear all merge (check) related fields
			pr.MergeSHA = nil
			pr.MergeTargetSHA = nil
			pr.Closed = &nowMilli
			pr.MarkAsMergeUnchecked()

			// delete the merge pull request reference
			err = c.git.UpdateRef(ctx, git.UpdateRefParams{
				WriteParams: targetWriteParams,
				Name:        strconv.FormatInt(pr.Number, 10),
				Type:        gitenum.RefTypePullReqMerge,
				NewValue:    sha.Nil,
				OldValue:    sha.None, // we don't care about the old value
			})

		case changeReopen:
			pr.SourceSHA = sourceSHA.String()
			pr.MergeBaseSHA = mergeBaseSHA.String()
			pr.Closed = nil

			// create the head pull request reference
			err = c.git.UpdateRef(ctx, git.UpdateRefParams{
				WriteParams: targetWriteParams,
				Name:        strconv.FormatInt(pr.Number, 10),
				Type:        gitenum.RefTypePullReqHead,
				NewValue:    sourceSHA,
				OldValue:    sha.None, // the request is re-opened, so anything can be the old value
			})
			if err != nil {
				return fmt.Errorf("failed to set value of PR head ref: %w", err)
			}
		}

		pr.ActivitySeq++ // because we need to add the activity entry

		err = c.pullreqStore.Update(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to update pull request: %w", err)
		}

		return nil
	}, controller.TxOptionResetFunc(func() {
		pr = nil // on the version conflict error force re-fetch of the pull request
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to update pull request: %w", err)
	}

	payload := &types.PullRequestActivityPayloadStateChange{
		Old:      oldState,
		New:      pr.State,
		OldDraft: oldDraft,
		NewDraft: pr.IsDraft,
	}
	if _, errAct := c.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID, payload, nil); errAct != nil {
		// non-critical error
		log.Ctx(ctx).Err(errAct).Msgf("failed to write pull request activity after state change")
	}

	switch stateChange {
	case changeReopen:
		c.eventReporter.Reopened(ctx, &pullreqevents.ReopenedPayload{
			Base:         eventBase(pr, &session.Principal),
			SourceSHA:    sourceSHA.String(),
			MergeBaseSHA: mergeBaseSHA.String(),
		})
	case changeClose:
		c.eventReporter.Closed(ctx, &pullreqevents.ClosedPayload{
			Base:      eventBase(pr, &session.Principal),
			SourceSHA: pr.SourceSHA,
		})
	}

	c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullReqUpdated, pr)

	return pr, nil
}
