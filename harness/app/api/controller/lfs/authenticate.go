// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lfs

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/token"
)

func (c *Controller) Authenticate(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
) (*AuthenticateResponse, error) {
	repo, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	gitLFSEnabled, err := settings.RepoGet(
		ctx,
		c.settings,
		repo.ID,
		settings.KeyGitLFSEnabled,
		settings.DefaultGitLFSEnabled,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check settings for Git LFS enabled: %w", err)
	}

	if !gitLFSEnabled {
		return nil, usererror.ErrGitLFSDisabled
	}

	jwt, err := c.remoteAuth.GenerateToken(ctx, session.Principal.ID, session.Principal.Type, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	return &AuthenticateResponse{
		Header: map[string]string{
			"Authorization": authn.HeaderTokenPrefixRemoteAuth + jwt,
		},
		HRef:      c.urlProvider.GenerateGITCloneURL(ctx, repoRef) + "/info/lfs",
		ExpiresIn: token.RemoteAuthTokenLifeTime,
	}, nil
}
