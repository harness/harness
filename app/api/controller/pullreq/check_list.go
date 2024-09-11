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

package pullreq

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListChecks return an array of status check results for a commit in a repository.
func (c *Controller) ListChecks(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
) (types.PullReqChecks, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return types.PullReqChecks{}, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return types.PullReqChecks{}, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	isRepoOwner, err := apiauth.IsRepoOwner(ctx, c.authorizer, session, repo)
	if err != nil {
		return types.PullReqChecks{}, fmt.Errorf("failed to determine if user is repo owner: %w", err)
	}

	protectionRules, err := c.protectionManager.ForRepository(ctx, repo.ID)
	if err != nil {
		return types.PullReqChecks{}, fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
	}

	reqChecks, err := protectionRules.RequiredChecks(ctx, protection.RequiredChecksInput{
		ResolveUserGroupID: c.userGroupService.ListUserIDsByGroupIDs,
		Actor:              &session.Principal,
		IsRepoOwner:        isRepoOwner,
		Repo:               repo,
		PullReq:            pr,
	})
	if err != nil {
		return types.PullReqChecks{}, fmt.Errorf("failed to get identifiers of required checks: %w", err)
	}

	commitSHA := pr.SourceSHA

	checks, err := c.checkStore.List(ctx, repo.ID, commitSHA, types.CheckListOptions{})
	if err != nil {
		return types.PullReqChecks{}, fmt.Errorf("failed to list status check results for repo: %w", err)
	}

	result := types.PullReqChecks{
		CommitSHA: commitSHA,
		Checks:    nil,
	}

	for _, check := range checks {
		_, required := reqChecks.RequiredIdentifiers[check.Identifier]
		if required {
			delete(reqChecks.RequiredIdentifiers, check.Identifier)
		}

		_, bypassable := reqChecks.BypassableIdentifiers[check.Identifier]
		if bypassable {
			delete(reqChecks.BypassableIdentifiers, check.Identifier)
		}

		result.Checks = append(result.Checks, types.PullReqCheck{
			Required:   required || bypassable,
			Bypassable: bypassable,
			Check:      check,
		})
	}

	for requiredID := range reqChecks.RequiredIdentifiers {
		result.Checks = append(result.Checks, types.PullReqCheck{
			Required:   true,
			Bypassable: false,
			Check: types.Check{
				RepoID:     repo.ID,
				CommitSHA:  commitSHA,
				Identifier: requiredID,
				Status:     enum.CheckStatusPending,
				Metadata:   json.RawMessage("{}"),
			},
		})
	}

	for bypassableID := range reqChecks.BypassableIdentifiers {
		result.Checks = append(result.Checks, types.PullReqCheck{
			Required:   true,
			Bypassable: true,
			Check: types.Check{
				RepoID:     repo.ID,
				CommitSHA:  commitSHA,
				Identifier: bypassableID,
				Status:     enum.CheckStatusPending,
				Metadata:   json.RawMessage("{}"),
			},
		})
	}

	// Note: The DB List method sorts by "check_updated desc", but here we sort by Identifier,
	// because we extended the list to include required elements not yet reported (their check_updated timestamp is 0).
	sort.Slice(result.Checks, func(i, j int) bool {
		return result.Checks[i].Check.Identifier < result.Checks[j].Check.Identifier
	})

	return result, nil
}
