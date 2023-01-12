// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/controller"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type MergeInput struct {
	Method       enum.MergeMethod `json:"method"`
	Force        bool             `json:"force,omitempty"`
	DeleteBranch bool             `json:"delete_branch,omitempty"`
}

// Merge merges the pull request.
//
//nolint:gocognit // no need to refactor
func (c *Controller) Merge(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	in *MergeInput,
) (types.MergeResponse, error) {
	var (
		sha      string
		pr       *types.PullReq
		activity *types.PullReqActivity
	)

	if in.Method == "" {
		in.Method = enum.MergeMethodMerge
	}

	method, ok := enum.ParseMergeMethod(in.Method)
	if !ok {
		return types.MergeResponse{}, usererror.BadRequest(
			fmt.Sprintf("wrong merge method type: %s", in.Method))
	}

	in.Method = method

	now := time.Now().UnixMilli()

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return types.MergeResponse{}, usererror.BadRequest(
			fmt.Sprintf("failed to acquire access to target repo: %s", err))
	}

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		// pesimistic lock for no other user can merge the same pr
		pr, err = c.pullreqStore.FindByNumberWithLock(ctx, targetRepo.ID, pullreqNum, true)
		if err != nil {
			return fmt.Errorf("failed to get pull request by number: %w", err)
		}

		if pr.Merged != nil {
			return errors.New("pull request already merged")
		}

		if pr.State != enum.PullReqStateOpen {
			return fmt.Errorf("pull request state cannot be %v", pr.State)
		}

		sourceRepo := targetRepo
		if pr.SourceRepoID != pr.TargetRepoID {
			sourceRepo, err = c.repoStore.Find(ctx, pr.SourceRepoID)
			if err != nil {
				return fmt.Errorf("failed to get source repository: %w", err)
			}
		}

		var writeParams gitrpc.WriteParams
		writeParams, err = controller.CreateRPCWriteParams(ctx, c.urlProvider, session, targetRepo)
		if err != nil {
			return fmt.Errorf("failed to create RPC write params: %w", err)
		}

		sha, err = c.gitRPCClient.MergeBranch(ctx, &gitrpc.MergeBranchParams{
			WriteParams:  writeParams,
			BaseBranch:   pr.TargetBranch,
			HeadRepoUID:  sourceRepo.GitUID,
			HeadBranch:   pr.SourceBranch,
			Force:        in.Force,
			DeleteBranch: in.DeleteBranch,
		})
		if err != nil {
			return err
		}

		activity = getMergeActivity(session, pr, in, sha)

		pr.MergeStrategy = &in.Method
		pr.Merged = &now
		pr.MergedBy = &session.Principal.ID
		pr.State = enum.PullReqStateMerged

		err = c.pullreqStore.Update(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to update pull request: %w", err)
		}

		return nil
	})
	if err != nil {
		return types.MergeResponse{}, err
	}

	err = c.writeActivity(ctx, pr, activity)
	if err != nil {
		log.Err(err).Msg("failed to write pull req activity")
	}

	return types.MergeResponse{
		SHA: sha,
	}, nil
}

func getMergeActivity(session *auth.Session, pr *types.PullReq, in *MergeInput, sha string) *types.PullReqActivity {
	now := time.Now().UnixMilli()

	act := &types.PullReqActivity{
		ID:        0, // Will be populated in the data layer
		Version:   0,
		CreatedBy: session.Principal.ID,
		Created:   now,
		Updated:   now,
		Edited:    now,
		Deleted:   nil,
		RepoID:    pr.TargetRepoID,
		PullReqID: pr.ID,
		Order:     0, // Will be filled in writeActivity
		SubOrder:  0,
		ReplySeq:  0,
		Type:      enum.PullReqActivityTypeMerge,
		Kind:      enum.PullReqActivityKindSystem,
		Text:      "",
		Payload: map[string]interface{}{
			"merge_method": in.Method,
			"sha":          sha,
		},
		Metadata:   nil,
		ResolvedBy: nil,
		Resolved:   nil,
	}

	return act
}
