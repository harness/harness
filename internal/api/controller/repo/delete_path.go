// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

/*
* DeletePath deletes a repo path.
 */
func (c *Controller) DeletePath(ctx context.Context, session *auth.Session, repoRef string, pathID int64) error {
	repo, err := c.repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return err
	}

	err = c.repoStore.DeletePath(ctx, repo.ID, pathID)
	if err != nil {
		return err
	}

	return nil
}
