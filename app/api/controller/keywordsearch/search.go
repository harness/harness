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
	"math"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *Controller) Search(
	ctx context.Context,
	session *auth.Session,
	in types.SearchInput,
) (types.SearchResult, error) {
	if in.Query == "" {
		return types.SearchResult{}, usererror.BadRequest("query cannot be empty.")
	}

	if len(in.RepoPaths) == 0 && len(in.SpacePaths) == 0 {
		return types.SearchResult{}, usererror.BadRequest(
			"either repo paths or space paths need to be set.")
	}

	repoIDToPathMap, err := c.getReposByPath(ctx, session, in.RepoPaths)
	if err != nil {
		return types.SearchResult{}, fmt.Errorf("failed to search repos by path: %w", err)
	}

	spaceRepoIDToPathMap, err := c.getReposBySpacePaths(ctx, session, in.SpacePaths, in.Recursive)
	if err != nil {
		return types.SearchResult{}, fmt.Errorf("failed to search repos by space path: %w", err)
	}

	for repoID, repoPath := range spaceRepoIDToPathMap {
		repoIDToPathMap[repoID] = repoPath
	}

	if len(repoIDToPathMap) == 0 {
		return types.SearchResult{}, usererror.NotFound("no repositories found")
	}

	repoIDs := make([]int64, 0, len(repoIDToPathMap))
	for repoID := range repoIDToPathMap {
		repoIDs = append(repoIDs, repoID)
	}

	result, err := c.searcher.Search(ctx, repoIDs, in.Query, in.EnableRegex, in.MaxResultCount)
	if err != nil {
		return types.SearchResult{}, fmt.Errorf("failed to search: %w", err)
	}

	for idx, fileMatch := range result.FileMatches {
		repoPath, ok := repoIDToPathMap[fileMatch.RepoID]
		if !ok {
			log.Ctx(ctx).Warn().Msgf("repo path not found for repo ID %d, repo mapping: %v",
				fileMatch.RepoID, repoIDToPathMap)
			continue
		}
		result.FileMatches[idx].RepoPath = repoPath
	}
	return result, nil
}

// getReposByPath returns a list of repo IDs that the user has access to for input repo paths.
func (c *Controller) getReposByPath(
	ctx context.Context,
	session *auth.Session,
	repoPaths []string,
) (map[int64]string, error) {
	repoIDToPathMap := make(map[int64]string)
	if len(repoPaths) == 0 {
		return repoIDToPathMap, nil
	}

	for _, repoPath := range repoPaths {
		if repoPath == "" {
			continue
		}

		repo, err := c.repoCtrl.Find(ctx, session, repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to find repository: %w", err)
		}
		repoIDToPathMap[repo.ID] = repoPath
	}
	return repoIDToPathMap, nil
}

func (c *Controller) getReposBySpacePaths(
	ctx context.Context,
	session *auth.Session,
	spacePaths []string,
	recursive bool,
) (map[int64]string, error) {
	repoIDToPathMap := make(map[int64]string)
	for _, spacePath := range spacePaths {
		m, err := c.getReposBySpacePath(ctx, session, spacePath, recursive)
		if err != nil {
			return nil, fmt.Errorf("failed to search repos by space path: %w", err)
		}

		for repoID, repoPath := range m {
			repoIDToPathMap[repoID] = repoPath
		}
	}
	return repoIDToPathMap, nil
}

func (c *Controller) getReposBySpacePath(
	ctx context.Context,
	session *auth.Session,
	spacePath string,
	recursive bool,
) (map[int64]string, error) {
	repoIDToPathMap := make(map[int64]string)
	if spacePath == "" {
		return repoIDToPathMap, nil
	}

	filter := &types.RepoFilter{
		Page:      1,
		Size:      int(math.MaxInt),
		Query:     "",
		Order:     enum.OrderAsc,
		Sort:      enum.RepoAttrNone,
		Recursive: recursive,
	}
	repos, _, err := c.spaceCtrl.ListRepositories(ctx, session, spacePath, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list space repositories: %w", err)
	}

	for _, repo := range repos {
		repoIDToPathMap[repo.ID] = repo.Path
	}
	return repoIDToPathMap, nil
}
