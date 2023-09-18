// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// FileViewDelete removes a file from being marked as viewed.
func (c *Controller) FileViewDelete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	filePath string,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return fmt.Errorf("failed to find pull request by number: %w", err)
	}

	if filePath == "" {
		return usererror.BadRequest("file path can't be empty")
	}

	err = c.fileViewStore.DeleteByFileForPrincipal(ctx, pr.ID, session.Principal.ID, filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file view entry in db: %w", err)
	}

	return nil
}
