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
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

type ListService struct {
	tx                dbtx.Transactor
	git               git.Interface
	authorizer        authz.Authorizer
	spaceStore        store.SpaceStore
	repoStore         store.RepoStore
	repoGitInfoCache  store.RepoGitInfoCache
	pullreqStore      store.PullReqStore
	checkStore        store.CheckStore
	labelSvc          *label.Service
	protectionManager *protection.Manager
}

func NewListService(
	tx dbtx.Transactor,
	git git.Interface,
	authorizer authz.Authorizer,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	repoGitInfoCache store.RepoGitInfoCache,
	pullreqStore store.PullReqStore,
	checkStore store.CheckStore,
	labelSvc *label.Service,
	protectionManager *protection.Manager,
) *ListService {
	return &ListService{
		tx:                tx,
		git:               git,
		authorizer:        authorizer,
		spaceStore:        spaceStore,
		repoStore:         repoStore,
		repoGitInfoCache:  repoGitInfoCache,
		pullreqStore:      pullreqStore,
		checkStore:        checkStore,
		labelSvc:          labelSvc,
		protectionManager: protectionManager,
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
	filter.Sort = enum.PullReqSortUpdated // the only supported option, hardcoded in the SQL query
	filter.Order = enum.OrderDesc         // the only supported option, hardcoded in the SQL query
	filter.Page = 0                       // unsupported, pagination should be done with the UpdatedLt parameter
	filter.UpdatedGt = 0                  // unsupported

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
			filter.UpdatedLt = pullReqs[len(pullReqs)-1].Updated
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

	if err := c.BackfillMetadata(ctx, response, filter.PullReqMetadataOptions); err != nil {
		return nil, fmt.Errorf("failed to backfill metadata: %w", err)
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
		if len(pullReqs) >= pullReqLimit || len(repoUnchecked) >= newRepoLimit {
			cancelFn() // the loop must be exited by canceling the context
			continue
		}

		if _, ok := repoWhitelist[pr.TargetRepoID]; !ok {
			repoUnchecked[pr.TargetRepoID] = struct{}{}
		}

		pullReqs = append(pullReqs, pr)
	}

	if err := <-chErr; err != nil && !errors.Is(err, context.Canceled) {
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

// BackfillChecks collects the check metadata for the provided list of pull requests.
func (c *ListService) BackfillChecks(
	ctx context.Context,
	list []types.PullReqRepo,
) error {
	// prepare list of commit SHAs per repository

	repoCommitSHAs := make(map[int64][]string)
	for _, entry := range list {
		repoID := entry.Repository.ID
		commitSHAs := repoCommitSHAs[repoID]
		repoCommitSHAs[repoID] = append(commitSHAs, entry.PullRequest.SourceSHA)
	}

	// fetch checks for every repository

	type repoSHA struct {
		repoID int64
		sha    string
	}

	repoCheckSummaryMap := make(map[repoSHA]types.CheckCountSummary)
	for repoID, commitSHAs := range repoCommitSHAs {
		commitCheckSummaryMap, err := c.checkStore.ResultSummary(ctx, repoID, commitSHAs)
		if err != nil {
			return fmt.Errorf("fail to fetch check summary for commits: %w", err)
		}

		for commitSHA, checkSummary := range commitCheckSummaryMap {
			repoCheckSummaryMap[repoSHA{repoID: repoID, sha: commitSHA.String()}] = checkSummary
		}
	}

	// backfill the list with check count summary

	for _, entry := range list {
		entry.PullRequest.CheckSummary =
			ptr.Of(repoCheckSummaryMap[repoSHA{repoID: entry.Repository.ID, sha: entry.PullRequest.SourceSHA}])
	}

	return nil
}

// BackfillRules collects the rule metadata for the provided list of pull requests.
func (c *ListService) BackfillRules(
	ctx context.Context,
	list []types.PullReqRepo,
) error {
	// prepare list of branch names per repository

	repoBranchNames := make(map[int64][]string)
	repoDefaultBranch := make(map[int64]string)
	for _, entry := range list {
		repoID := entry.Repository.ID
		branchNames := repoBranchNames[repoID]
		repoBranchNames[repoID] = append(branchNames, entry.PullRequest.TargetBranch)
		repoDefaultBranch[repoID] = entry.Repository.DefaultBranch
	}

	// fetch checks for every repository

	type repoBranchName struct {
		repoID     int64
		branchName string
	}

	repoBranchNameMap := make(map[repoBranchName][]types.RuleInfo)
	for repoID, branchNames := range repoBranchNames {
		repoProtection, err := c.protectionManager.ForRepository(ctx, repoID)
		if err != nil {
			return fmt.Errorf("fail to fetch protection rules for repository: %w", err)
		}

		for _, branchName := range branchNames {
			branchRuleInfos, err := protection.GetRuleInfos(
				repoProtection,
				repoDefaultBranch[repoID],
				branchName,
				protection.RuleInfoFilterStatusActive,
				protection.RuleInfoFilterTypeBranch)
			if err != nil {
				return fmt.Errorf("fail to get rule infos for branch %s: %w", branchName, err)
			}

			repoBranchNameMap[repoBranchName{repoID: repoID, branchName: branchName}] = branchRuleInfos
		}
	}

	// backfill the list with check count summary

	for _, entry := range list {
		key := repoBranchName{repoID: entry.Repository.ID, branchName: entry.PullRequest.TargetBranch}
		entry.PullRequest.Rules = repoBranchNameMap[key]
	}

	return nil
}

func (c *ListService) BackfillMetadata(
	ctx context.Context,
	list []types.PullReqRepo,
	options types.PullReqMetadataOptions,
) error {
	if options.IsAllFalse() {
		return nil
	}

	if options.IncludeChecks {
		if err := c.BackfillChecks(ctx, list); err != nil {
			return fmt.Errorf("failed to backfill checks")
		}
	}

	if options.IncludeRules {
		if err := c.BackfillRules(ctx, list); err != nil {
			return fmt.Errorf("failed to backfill rules")
		}
	}

	return nil
}

func (c *ListService) BackfillMetadataForRepo(
	ctx context.Context,
	repo *types.Repository,
	list []*types.PullReq,
	options types.PullReqMetadataOptions,
) error {
	if options.IsAllFalse() {
		return nil
	}

	listPullReqRepo := make([]types.PullReqRepo, len(list))
	for i, pr := range list {
		listPullReqRepo[i] = types.PullReqRepo{
			PullRequest: pr,
			Repository:  repo,
		}
	}

	if options.IncludeChecks {
		if err := c.BackfillChecks(ctx, listPullReqRepo); err != nil {
			return fmt.Errorf("failed to backfill checks")
		}
	}

	if options.IncludeRules {
		if err := c.BackfillRules(ctx, listPullReqRepo); err != nil {
			return fmt.Errorf("failed to backfill rules")
		}
	}

	return nil
}

func (c *ListService) BackfillMetadataForPullReq(
	ctx context.Context,
	repo *types.Repository,
	pr *types.PullReq,
	options types.PullReqMetadataOptions,
) error {
	if options.IsAllFalse() {
		return nil
	}

	list := []types.PullReqRepo{{
		PullRequest: pr,
		Repository:  repo,
	}}

	if options.IncludeChecks {
		if err := c.BackfillChecks(ctx, list); err != nil {
			return fmt.Errorf("failed to backfill checks")
		}
	}

	if options.IncludeRules {
		if err := c.BackfillRules(ctx, list); err != nil {
			return fmt.Errorf("failed to backfill rules")
		}
	}

	return nil
}
