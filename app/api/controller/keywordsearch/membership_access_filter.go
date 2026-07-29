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

package keywordsearch

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type membershipViewAccessFilter struct {
	authorizer   authz.Authorizer
	publicAccess publicaccess.Service
}

func (f membershipViewAccessFilter) ViewAccessFilter(
	ctx context.Context,
	session *auth.Session,
	repoMap map[int64]string, // repo ID to path
) ([]int64, error) {
	// mapSpaceRepoIDs contains the map of relevant spaces, and for each a list of repositories in it.
	mapSpaceRepoIDs := make(map[string][]int64)

	// checks contains list of types.PermissionCheck objects for the CheckMany call.
	checks := make([]types.PermissionCheck, 0)

	for repoID, repoPath := range repoMap {
		spacePath := paths.Parent(repoPath)
		if spacePath == "" {
			continue
		}

		if repoIDsInSpace, ok := mapSpaceRepoIDs[spacePath]; ok {
			repoIDsInSpace = append(repoIDsInSpace, repoID)
			mapSpaceRepoIDs[spacePath] = repoIDsInSpace
			continue
		}

		checks = append(checks, types.PermissionCheck{
			Scope:      types.Scope{SpacePath: spacePath},
			Resource:   types.Resource{Type: enum.ResourceTypeRepo},
			Permission: enum.PermissionRepoView,
		})

		repoIDsInSpace := make([]int64, 0, 16)
		repoIDsInSpace = append(repoIDsInSpace, repoID)

		mapSpaceRepoIDs[spacePath] = repoIDsInSpace
	}

	results, err := f.authorizer.CheckMany(ctx, session, checks...)
	if err != nil {
		return nil, fmt.Errorf("failed to check view access for repositories: %w", err)
	}

	if len(checks) != len(results) {
		return nil, fmt.Errorf("mismatch between number of checks=%d and results=%d",
			len(checks), len(results))
	}

	// count repos with access
	var count int
	for i, result := range results {
		if result {
			count += len(mapSpaceRepoIDs[checks[i].Scope.SpacePath])
		}
	}

	// create the final result
	reposWithViewAccess := make([]int64, 0, count)
	for i, result := range results {
		reposInSpace := mapSpaceRepoIDs[checks[i].Scope.SpacePath]
		if result {
			reposWithViewAccess = append(reposWithViewAccess, reposInSpace...)
			continue
		}

		for _, repoID := range reposInSpace {
			isPublic, err := f.publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repoMap[repoID])
			if err != nil {
				return nil, fmt.Errorf("failed to get if repository is public: %w", err)
			}

			if isPublic {
				reposWithViewAccess = append(reposWithViewAccess, repoID)
			}
		}
	}

	return reposWithViewAccess, nil
}
