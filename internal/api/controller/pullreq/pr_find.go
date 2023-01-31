// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/sync/errgroup"
)

// Find returns a pull request from the provided repository.
func (c *Controller) Find(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
) (*types.PullReq, error) {
	if pullreqNum <= 0 {
		return nil, usererror.BadRequest("A valid pull request number must be provided.")
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to the repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return nil, err
	}

	pr.Stats, err = c.getStats(ctx, repo, pr)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

func (c *Controller) getStats(
	ctx context.Context,
	repo *types.Repository,
	pr *types.PullReq,
) (types.PullReqStats, error) {
	// declare variables which will be used in go routines,
	// no need for atomic operations because writing and reading variable
	// doesn't happen at the same time
	var (
		totalConvs   int64
		totalCommits int
		totalFiles   int
	)

	gitRef := pr.SourceBranch
	afterRef := pr.TargetBranch
	if pr.State == enum.PullReqStateMerged {
		gitRef = *pr.MergeHeadSHA
		afterRef = *pr.MergeBaseSHA
	}

	errGroup, groupCtx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		// return conversations
		var err error
		filter := &types.PullReqActivityFilter{
			Types: []enum.PullReqActivityType{
				enum.PullReqActivityTypeComment,
				enum.PullReqActivityTypeCodeComment,
			},
		}
		totalConvs, err = c.activityStore.Count(groupCtx, pr.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count pull request comments: %w", err)
		}
		return nil
	})

	errGroup.Go(func() error {
		// read total commits
		options := &gitrpc.GetCommitDivergencesParams{
			ReadParams: gitrpc.CreateRPCReadParams(repo),
			Requests: []gitrpc.CommitDivergenceRequest{
				{
					From: gitRef,
					To:   afterRef,
				},
			},
		}

		rpcOutput, err := c.gitRPCClient.GetCommitDivergences(groupCtx, options)
		if err != nil {
			return fmt.Errorf("failed to count pull request commits: %w", err)
		}
		if len(rpcOutput.Divergences) > 0 {
			totalCommits = int(rpcOutput.Divergences[0].Ahead)
		}
		return nil
	})

	errGroup.Go(func() error {
		// read short stat
		stat, err := c.gitRPCClient.DiffShortStat(groupCtx, &gitrpc.DiffParams{
			ReadParams: gitrpc.CreateRPCReadParams(repo),
			BaseRef:    afterRef,
			HeadRef:    gitRef,
			MergeBase:  true,
		})
		if err != nil {
			return fmt.Errorf("failed to count pull request file changes: %w", err)
		}
		totalFiles = stat.Files
		return nil
	})

	err := errGroup.Wait()
	if err != nil {
		return types.PullReqStats{}, err
	}

	return types.PullReqStats{
		Conversations: totalConvs,
		Commits:       totalCommits,
		FilesChanged:  totalFiles,
	}, nil
}
