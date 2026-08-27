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
	"maps"
	"slices"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func (c *Controller) Search(
	ctx context.Context,
	session *auth.Session,
	in types.SearchInput,
) (types.SearchResult, error) {
	if err := sanitizeInput(&in); err != nil {
		return types.SearchResult{}, err
	}

	// repoIDToPathMap is a map of repositories to search, but with yet unverified access permission.
	repoIDToPathMap := make(map[int64]string)

	// Go through the list of requested repositories, verify if each exists and add it to the map.
	for _, repoPath := range in.RepoPaths {
		if repoPath == "" {
			continue
		}

		repo, err := c.repoFinder.FindByRef(ctx, repoPath)
		if err != nil {
			return types.SearchResult{}, fmt.Errorf("failed to find repo by path %q: %w", repoPath, err)
		}

		repoIDToPathMap[repo.ID] = repo.Path
	}

	// Go through the list of requested spaces, verify if each exists and add it to the list spaces.
	spaceIDs := make([]int64, 0, len(in.SpacePaths))
	for _, spacePath := range in.SpacePaths {
		if spacePath == "" {
			continue
		}

		space, err := c.spaceFinder.FindByRef(ctx, spacePath)
		if err != nil {
			return types.SearchResult{}, fmt.Errorf("failed to find space by path %q: %w", spacePath, err)
		}

		spaceIDs = append(spaceIDs, space.ID)
	}

	slices.Sort(spaceIDs)
	spaceIDs = slices.Compact(spaceIDs)

	for _, spaceID := range spaceIDs {
		repoPaths, err := c.repoStore.MapOfAllRepos(ctx, spaceID, in.Recursive)
		if err != nil {
			return types.SearchResult{}, fmt.Errorf("failed to find repo uids by space id: %w", err)
		}

		maps.Copy(repoIDToPathMap, repoPaths)
	}

	if len(repoIDToPathMap) == 0 {
		return types.SearchResult{}, usererror.NotFound("No repositories found")
	}

	repoIDs, err := c.repoFilter.ViewAccessFilter(ctx, session, repoIDToPathMap)
	if err != nil {
		return types.SearchResult{}, fmt.Errorf("failed to filter repos with view access: %w", err)
	}

	if len(repoIDs) == 0 {
		return types.SearchResult{}, usererror.NotFound("No repositories found")
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

	for idx, repoMatch := range result.Stats.PerRepoMatches {
		repoPath, ok := repoIDToPathMap[repoMatch.RepoID]
		if !ok {
			log.Ctx(ctx).Warn().Msgf("repo path not found for repo ID %d, repo mapping: %v",
				repoMatch.RepoID, repoIDToPathMap)
			continue
		}
		result.Stats.PerRepoMatches[idx].RepoPath = repoPath
	}

	return result, nil
}

func sanitizeInput(in *types.SearchInput) error {
	if in.Query == "" {
		return usererror.BadRequest("Query cannot be empty.")
	}

	if len(in.RepoPaths) == 0 && len(in.SpacePaths) == 0 {
		return usererror.BadRequest(
			"either repo paths or space paths need to be set.")
	}

	slices.Sort(in.RepoPaths)
	slices.Sort(in.SpacePaths)
	in.RepoPaths = slices.Compact(in.RepoPaths)
	in.SpacePaths = slices.Compact(in.SpacePaths)

	return nil
}
