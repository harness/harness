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

package migrate

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/migrate"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type PullreqsInput struct {
	PullRequestData []*migrate.ExternalPullRequest `json:"pull_request_data"`
}

func (c *Controller) PullRequests(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *PullreqsInput,
) ([]*types.PullReq, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	if repo.State != enum.RepoStateMigrateDataImport {
		return nil, fmt.Errorf("repo state is %s want %s: %w",
			repo.State, enum.RepoStateMigrateDataImport, errInvalidRepoState)
	}

	pullreqs, err := c.pullreqImporter.Import(ctx, session.Principal, repo, in.PullRequestData)
	if err != nil {
		return nil, fmt.Errorf("failed to import pull requests and/or comments: %w", err)
	}

	return pullreqs, nil
}
