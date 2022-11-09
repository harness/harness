// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"net/url"
	"path"
	"strings"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Find finds a repo.
func (c *Controller) Find(ctx context.Context, session *auth.Session, repoRef string,
	cfg *types.Config) (*types.Repository, error) {
	repo, err := findRepoFromRef(ctx, c.repoStore, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, true); err != nil {
		return nil, err
	}
	repoPath := path.Clean(repo.Path)
	if !strings.HasSuffix(repoPath, ".git") {
		repoPath += ".git"
	}
	repo.URL, err = url.JoinPath(cfg.Git.BaseURL, repoPath)
	if err != nil {
		return nil, err
	}
	return repo, nil
}
