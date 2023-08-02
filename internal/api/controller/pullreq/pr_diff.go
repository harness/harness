// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/config"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// RawDiff writes raw git diff to writer w.
func (c *Controller) RawDiff(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	setSHAs func(sourceSHA, mergeBaseSHA string),
	w io.Writer,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return fmt.Errorf("failed to get pull request by number: %w", err)
	}

	headRef := pr.SourceSHA
	baseRef := pr.MergeBaseSHA

	setSHAs(headRef, baseRef)

	return c.gitRPCClient.RawDiff(ctx, &gitrpc.DiffParams{
		ReadParams: gitrpc.CreateRPCReadParams(repo),
		BaseRef:    baseRef,
		HeadRef:    headRef,
		MergeBase:  true,
	}, w)
}

func (c *Controller) Diff(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
) (types.Stream[*gitrpc.FileDiff], error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	headRef := pr.SourceBranch
	if pr.SourceSHA != "" {
		headRef = pr.SourceSHA
	}
	baseRef := pr.TargetBranch
	if pr.State == enum.PullReqStateMerged {
		baseRef = pr.MergeBaseSHA
	}

	reader := gitrpc.NewStreamReader(c.gitRPCClient.Diff(ctx, &gitrpc.DiffParams{
		ReadParams:   gitrpc.CreateRPCReadParams(repo),
		BaseRef:      baseRef,
		HeadRef:      headRef,
		MergeBase:    true,
		IncludePatch: true,
	}, config.ApiURL))

	return reader, nil
}
