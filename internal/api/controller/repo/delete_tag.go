// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// DeleteTag deletes a tag from the repo.
func (c *Controller) DeleteTag(ctx context.Context, session *auth.Session,
	repoRef, tagName string) error {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoPush, false); err != nil {
		return err
	}

	writeParams, err := CreateRPCWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return fmt.Errorf("failed to create RPC write params: %w", err)
	}

	err = c.gitRPCClient.DeleteTag(ctx, &gitrpc.DeleteTagParams{
		Name:        tagName,
		WriteParams: writeParams,
	})
	if err != nil {
		return err
	}
	return nil
}
