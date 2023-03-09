// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"strings"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Blame(ctx context.Context, session *auth.Session,
	repoRef, gitRef, path string,
	lineFrom, lineTo int,
) (types.Stream[*gitrpc.BlamePart], error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, usererror.BadRequest("File path needs to specified.")
	}

	if lineTo > 0 && lineFrom > lineTo {
		return nil, usererror.BadRequest("Line range must be valid.")
	}

	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, true); err != nil {
		return nil, err
	}

	if gitRef == "" {
		gitRef = repo.DefaultBranch
	}

	reader := gitrpc.NewStreamReader(
		c.gitRPCClient.Blame(ctx, &gitrpc.BlameParams{
			ReadParams: CreateRPCReadParams(repo),
			GitRef:     gitRef,
			Path:       path,
			LineFrom:   lineFrom,
			LineTo:     lineTo,
		}))

	return reader, nil
}
