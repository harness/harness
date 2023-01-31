// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/sync/errgroup"
)

func (c *Controller) GetMetaData(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
) (types.PullReqMetaData, error) {
	// declare variables which will be used in go routines,
	// no need for atomic operations because writing and reading variable
	// doesn't happen at the same time
	var (
		totalConvs   int64
		totalCommits int
		totalFiles   int
	)

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return types.PullReqMetaData{}, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return types.PullReqMetaData{}, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	gitRef := pr.SourceBranch
	afterRef := pr.TargetBranch
	if pr.State == enum.PullReqStateMerged {
		gitRef = *pr.MergeHeadSHA
		afterRef = *pr.MergeBaseSHA
	}

	errGroup, groupCtx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		// return conversations
		var errStore error
		filter := &types.PullReqActivityFilter{
			Types: []enum.PullReqActivityType{
				enum.PullReqActivityTypeComment,
				enum.PullReqActivityTypeCodeComment,
			},
		}
		totalConvs, errStore = c.activityStore.Count(groupCtx, pr.ID, filter)
		if errStore != nil {
			return fmt.Errorf("failed to count pull request comments: %w", errStore)
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

		rpcOutput, errGit := c.gitRPCClient.GetCommitDivergences(groupCtx, options)
		if errGit != nil {
			return fmt.Errorf("failed to count pull request commits: %w", errGit)
		}
		if len(rpcOutput.Divergences) > 0 {
			totalCommits = int(rpcOutput.Divergences[0].Ahead)
		}
		return nil
	})

	errGroup.Go(func() error {
		// read short stat
		stat, errGit := c.gitRPCClient.DiffShortStat(groupCtx, &gitrpc.DiffParams{
			ReadParams: gitrpc.CreateRPCReadParams(repo),
			BaseRef:    afterRef,
			HeadRef:    gitRef,
			MergeBase:  true,
		})
		if errGit != nil {
			return fmt.Errorf("failed to count pull request file changes: %w", errGit)
		}
		totalFiles = stat.Files
		return nil
	})

	err = errGroup.Wait()
	if err != nil {
		return types.PullReqMetaData{}, err
	}

	return types.PullReqMetaData{
		Conversations: totalConvs,
		Commits:       totalCommits,
		FilesChanged:  totalFiles,
	}, nil
}
