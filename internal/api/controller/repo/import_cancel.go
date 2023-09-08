// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// ImportCancel cancels a repository import.
func (c *Controller) ImportCancel(ctx context.Context,
	session *auth.Session,
	repoRef string,
) error {
	// note: can't use c.getRepoCheckAccess because this needs to fetch a repo being imported.
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoDelete, false); err != nil {
		return err
	}

	if !repo.Importing {
		return usererror.BadRequest("repository is not being imported")
	}

	if err = c.importer.Cancel(ctx, repo); err != nil {
		return fmt.Errorf("failed to cancel repository import")
	}

	return c.DeleteNoAuth(ctx, session, repo)
}
