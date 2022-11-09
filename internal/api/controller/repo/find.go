// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Find finds a repo.
func (c *Controller) Find(ctx context.Context, session *auth.Session, repoRef string) (*types.Repository, error) {
	repo, err := findRepoFromRef(ctx, c.repoStore, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, true); err != nil {
		return nil, err
	}

	repo.GitURL, err = GenerateRepoGitURL(c.gitBaseURL, repo.Path)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func GenerateRepoGitURL(gitBaseURL string, repoPath string) (string, error) {
	repoPath = path.Clean(repoPath)
	if !strings.HasSuffix(repoPath, ".git") {
		repoPath += ".git"
	}
	gitURL, err := url.JoinPath(gitBaseURL, repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to join base url '%s' with path '%s': %w", gitBaseURL, repoPath, err)
	}

	return gitURL, nil
}
