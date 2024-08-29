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
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type ListService struct {
	tx               dbtx.Transactor
	git              git.Interface
	authorizer       authz.Authorizer
	spaceStore       store.SpaceStore
	repoStore        store.RepoStore
	repoGitInfoCache store.RepoGitInfoCache
	pullreqStore     store.PullReqStore
	labelSvc         *label.Service
}

func NewListService(
	tx dbtx.Transactor,
	git git.Interface,
	authorizer authz.Authorizer,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	repoGitInfoCache store.RepoGitInfoCache,
	pullreqStore store.PullReqStore,
	labelSvc *label.Service,
) *ListService {
	return &ListService{
		tx:               tx,
		git:              git,
		authorizer:       authorizer,
		spaceStore:       spaceStore,
		repoStore:        repoStore,
		repoGitInfoCache: repoGitInfoCache,
		pullreqStore:     pullreqStore,
		labelSvc:         labelSvc,
	}
}

// ListForSpace returns a list of pull requests and their respective repositories for a specific space.
//
//nolint:gocognit
func (c *ListService) ListForSpace(
	ctx context.Context,
	session *auth.Session,
	space *types.Space,
	includeSubspaces bool,
	filter *types.PullReqFilter,
) ([]types.PullReqRepo, error) {
	// list of unsupported filter options
	filter.Sort = enum.PullReqSortEdited // the only supported option, hardcoded in the SQL query
	filter.Order = enum.OrderDesc        // the only supported option, hardcoded in the SQL query
	filter.Page = 0                      // unsupported, pagination should be done with the EditedLt parameter
	filter.EditedGt = 0                  // unsupported

	if includeSubspaces {
		subspaces, err := c.spaceStore.GetDescendantsData(ctx, space.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get space descendant data: %w", err)
		}

		filter.SpaceIDs = make([]int64, 0, len(subspaces))
		for i := range subspaces {
			filter.SpaceIDs = append(filter.SpaceIDs, subspaces[i].ID)
		}
	} else {
		filter.SpaceIDs = []int64{space.ID}
	}

	repoWhitelist := make(map[int64]struct{})

	list := make([]*types.PullReq, 0, 16)
	repoMap := make(map[int64]*types.Repository)

	for loadMore := true; loadMore; {
		const prLimit = 100
		const repoLimit = 10

		pullReqs, repoUnchecked, err := c.streamPullReqs(ctx, filter, prLimit, repoLimit, repoWhitelist)
		if err != nil {
			return nil, fmt.Errorf("failed to load pull requests: %w", err)
		}

		loadMore = len(pullReqs) == prLimit || len(repoUnchecked) == repoLimit
		if loadMore && len(pullReqs) > 0 {
			filter.EditedLt = pullReqs[len(pullReqs)-1].Edited
		}

		for repoID := range repoUnchecked {
			repo, err := c.repoStore.Find(ctx, repoID)
			if errors.Is(err, gitness_store.ErrResourceNotFound) {
				filter.RepoIDBlacklist = append(filter.RepoIDBlacklist, repoID)
				continue
			} else if err != nil {
				return nil, fmt.Errorf("failed to find repo: %w", err)
			}

			err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView)
			switch {
			case err == nil:
				repoWhitelist[repoID] = struct{}{}
				repoMap[repoID] = repo
			case errors.Is(err, apiauth.ErrNotAuthorized):
				filter.RepoIDBlacklist = append(filter.RepoIDBlacklist, repoID)
			default:
				return nil, fmt.Errorf("failed to check access check: %w", err)
			}
		}

		for _, pullReq := range pullReqs {
			if _, ok := repoWhitelist[pullReq.TargetRepoID]; ok {
				list = append(list, pullReq)
			}
		}

		if len(list) >= filter.Size {
			list = list[:filter.Size]
			loadMore = false
		}
	}

	if err := c.labelSvc.BackfillMany(ctx, list); err != nil {
		return nil, fmt.Errorf("failed to backfill labels assigned to pull requests: %w", err)
	}

	for _, pr := range list {
		if err := c.BackfillStats(ctx, pr); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to backfill PR stats")
		}
	}

	response := make([]types.PullReqRepo, len(list))
	for i := range list {
		response[i] = types.PullReqRepo{
			PullRequest: list[i],
			Repository:  repoMap[list[i].TargetRepoID],
		}
	}

	return response, nil
}

// streamPullReqs loads pull requests until it gets either pullReqLimit pull requests
// or newRepoLimit distinct repositories.
func (c *ListService) streamPullReqs(
	ctx context.Context,
	opts *types.PullReqFilter,
	pullReqLimit, newRepoLimit int,
	repoWhitelist map[int64]struct{},
) ([]*types.PullReq, map[int64]struct{}, error) {
	ctx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	repoUnchecked := map[int64]struct{}{}

	pullReqs := make([]*types.PullReq, 0, opts.Size)
	ch, chErr := c.pullreqStore.Stream(ctx, opts)
	for pr := range ch {
		if pr == nil {
			return pullReqs, repoUnchecked, nil
		}

		if _, ok := repoWhitelist[pr.TargetRepoID]; !ok {
			repoUnchecked[pr.TargetRepoID] = struct{}{}
		}

		pullReqs = append(pullReqs, pr)

		if len(pullReqs) >= pullReqLimit || len(repoUnchecked) >= newRepoLimit {
			break
		}
	}

	if err := <-chErr; err != nil {
		return nil, nil, fmt.Errorf("failed to stream pull requests: %w", err)
	}

	return pullReqs, repoUnchecked, nil
}

func (c *ListService) BackfillStats(ctx context.Context, pr *types.PullReq) error {
	s := pr.Stats.DiffStats
	if s.Commits != nil && s.FilesChanged != nil && s.Additions != nil && s.Deletions != nil {
		return nil
	}

	repoGitInfo, err := c.repoGitInfoCache.Get(ctx, pr.TargetRepoID)
	if err != nil {
		return fmt.Errorf("failed get repo git info to fetch diff stats: %w", err)
	}

	output, err := c.git.DiffStats(ctx, &git.DiffParams{
		ReadParams: git.CreateReadParams(repoGitInfo),
		BaseRef:    pr.MergeBaseSHA,
		HeadRef:    pr.SourceSHA,
	})
	if err != nil {
		return fmt.Errorf("failed get diff stats: %w", err)
	}

	pr.Stats.DiffStats = types.NewDiffStats(output.Commits, output.FilesChanged, output.Additions, output.Deletions)

	return nil
}
