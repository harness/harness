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
	"github.com/harness/gitness/types/enum"
)

// RawDiff writes raw git diff to writer w.
func (c *Controller) RawDiff(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	w io.Writer,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return fmt.Errorf("failed to get pull request by number: %w", err)
	}

	headRef := pr.SourceBranch
	if pr.SourceSHA != "" {
		headRef = pr.SourceSHA
	}
	baseRef := pr.TargetBranch
	if pr.MergeBaseSHA != nil {
		baseRef = *pr.MergeBaseSHA
	}

	return c.gitRPCClient.RawDiff(ctx, &gitrpc.DiffParams{
		ReadParams: gitrpc.CreateRPCReadParams(repo),
		BaseRef:    baseRef,
		HeadRef:    headRef,
		MergeBase:  true,
	}, w)
}
