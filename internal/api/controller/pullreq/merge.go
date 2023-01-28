// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/controller"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type MergeInput struct {
	Method       enum.MergeMethod `json:"method"`
	Force        bool             `json:"force,omitempty"`
	DeleteBranch bool             `json:"delete_branch,omitempty"`
}

// Merge merges the pull request.
//
//nolint:funlen // no need to refactor
func (c *Controller) Merge(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	in *MergeInput,
) (types.MergeResponse, error) {
	var (
		sha string
		pr  *types.PullReq
	)

	method, ok := in.Method.Sanitize()
	if !ok {
		return types.MergeResponse{}, usererror.BadRequest(
			fmt.Sprintf("wrong merge method type: %s", in.Method))
	}
	in.Method = method

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return types.MergeResponse{}, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	// if two requests for merging comes at the same time then mutex will lock
	// first one and second one will wait, when first one is done then second one
	// continue with latest data from db with state merged and return error that
	// pr is already merged.
	mutex, err := c.newMutexForPR(targetRepo.GitUID, 0) // 0 means locks all PRs for this repo
	if err != nil {
		return types.MergeResponse{}, err
	}
	err = mutex.Lock(ctx)
	if err != nil {
		return types.MergeResponse{}, err
	}
	defer func() {
		_ = mutex.Unlock(ctx)
	}()

	pr, err = c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
	if err != nil {
		return types.MergeResponse{}, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	if pr.Merged != nil {
		return types.MergeResponse{}, usererror.BadRequest("Pull request already merged")
	}

	if pr.State != enum.PullReqStateOpen {
		return types.MergeResponse{}, usererror.BadRequest("Pull request must be open")
	}

	if pr.IsDraft {
		return types.MergeResponse{}, usererror.BadRequest("Draft pull requests can't be merged. Clear the draft flag first.")
	}

	sourceRepo := targetRepo
	if pr.SourceRepoID != pr.TargetRepoID {
		sourceRepo, err = c.repoStore.Find(ctx, pr.SourceRepoID)
		if err != nil {
			return types.MergeResponse{}, fmt.Errorf("failed to get source repository: %w", err)
		}
	}

	var writeParams gitrpc.WriteParams
	writeParams, err = controller.CreateRPCWriteParams(ctx, c.urlProvider, session, targetRepo)
	if err != nil {
		return types.MergeResponse{}, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	// TODO: for forking merge title might be different?
	mergeTitle := fmt.Sprintf("Merge branch '%s' of %s (#%d)", pr.SourceBranch, sourceRepo.Path, pr.Number)

	var mergeOutput gitrpc.MergeBranchOutput
	mergeOutput, err = c.gitRPCClient.MergeBranch(ctx, &gitrpc.MergeBranchParams{
		WriteParams:      writeParams,
		BaseBranch:       pr.TargetBranch,
		HeadRepoUID:      sourceRepo.GitUID,
		HeadBranch:       pr.SourceBranch,
		Title:            mergeTitle,
		Message:          "",
		Force:            in.Force,
		DeleteHeadBranch: in.DeleteBranch,
	})
	if err != nil {
		return types.MergeResponse{}, err
	}

	pr, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		now := time.Now().UnixMilli()
		pr.MergeStrategy = &in.Method
		pr.Merged = &now
		pr.MergedBy = &session.Principal.ID
		pr.State = enum.PullReqStateMerged

		pr.MergeBaseSHA = &mergeOutput.BaseSHA
		pr.MergeHeadSHA = &mergeOutput.HeadSHA
		return nil
	})
	if err != nil {
		return types.MergeResponse{}, fmt.Errorf("failed to update pull request: %w", err)
	}

	c.eventReporter.Merged(ctx, &pullreqevents.MergedPayload{
		Base:        eventBase(pr, &session.Principal),
		MergeMethod: in.Method,
		SHA:         sha,
	})

	return types.MergeResponse{
		SHA: sha,
	}, nil
}
