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

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
)

// ListBranches lists the branches of a repo.
func (c *Controller) ListBranches(ctx context.Context,
	session *auth.Session,
	repoRef string,
	filter *types.BranchFilter,
) ([]types.BranchExtended, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	rpcOut, err := c.git.ListBranches(ctx, &git.ListBranchesParams{
		ReadParams:    git.CreateReadParams(repo),
		IncludeCommit: filter.IncludeCommit,
		Query:         filter.Query,
		Sort:          mapToRPCBranchSortOption(filter.Sort),
		Order:         mapToRPCSortOrder(filter.Order),
		Page:          int32(filter.Page),
		PageSize:      int32(filter.Size),
	})
	if err != nil {
		return nil, fmt.Errorf("fail to get the list of branches from git: %w", err)
	}

	branches := rpcOut.Branches

	metadata, err := c.collectBranchMetadata(ctx, repo, branches, filter.BranchMetadataOptions)
	if err != nil {
		return nil, fmt.Errorf("fail to collect branch metadata: %w", err)
	}

	response := make([]types.BranchExtended, len(branches))
	for i := range branches {
		response[i].Branch, err = controller.MapBranch(branches[i])
		if err != nil {
			return nil, fmt.Errorf("failed to map branch: %w", err)
		}

		response[i].IsDefault = repo.DefaultBranch == branches[i].Name

		metadata.apply(i, &response[i])
	}

	return response, nil
}

// collectBranchMetadata collects the metadata for the provided list of branches.
// The metadata includes check, rules, pull requests, and branch divergences.
// Each of these would be returned only if the corresponding option is true.
func (c *Controller) collectBranchMetadata(
	ctx context.Context,
	repo *types.RepositoryCore,
	branches []git.Branch,
	options types.BranchMetadataOptions,
) (branchMetadataOutput, error) {
	var (
		checkSummary  map[sha.SHA]types.CheckCountSummary
		branchRuleMap map[string][]types.RuleInfo
		pullReqMap    map[string][]*types.PullReq
		divergences   *git.GetCommitDivergencesOutput
		err           error
	)

	if options.IncludeChecks {
		commitSHAs := make([]string, len(branches))
		for i := range branches {
			commitSHAs[i] = branches[i].SHA.String()
		}

		checkSummary, err = c.checkStore.ResultSummary(ctx, repo.ID, commitSHAs)
		if err != nil {
			return branchMetadataOutput{}, fmt.Errorf("fail to fetch check summary for commits: %w", err)
		}
	}

	if options.IncludeRules {
		rules, err := c.protectionManager.ForRepository(ctx, repo.ID)
		if err != nil {
			return branchMetadataOutput{}, fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
		}

		branchRuleMap = make(map[string][]types.RuleInfo)
		for i := range branches {
			branchName := branches[i].Name

			branchRuleInfos, err := protection.GetRuleInfos(
				rules,
				repo.DefaultBranch,
				branchName,
				protection.RuleInfoFilterStatusActive,
				protection.RuleInfoFilterTypeBranch)
			if err != nil {
				return branchMetadataOutput{}, fmt.Errorf("failed get branch rule infos: %w", err)
			}

			branchRuleMap[branchName] = branchRuleInfos
		}
	}

	if options.IncludePullReqs {
		branchNames := make([]string, len(branches))
		for i := range branches {
			branchNames[i] = branches[i].Name
		}

		pullReqMap, err = c.pullReqStore.ListOpenByBranchName(ctx, repo.ID, branchNames)
		if err != nil {
			return branchMetadataOutput{}, fmt.Errorf("fail to fetch pull requests per branch: %w", err)
		}
	}

	if options.MaxDivergence > 0 {
		readParams := git.CreateReadParams(repo)

		divergenceRequests := make([]git.CommitDivergenceRequest, len(branches))
		for i := range branches {
			divergenceRequests[i].From = branches[i].Name
			divergenceRequests[i].To = repo.DefaultBranch
		}

		divergences, err = c.git.GetCommitDivergences(ctx, &git.GetCommitDivergencesParams{
			ReadParams: readParams,
			MaxCount:   int32(options.MaxDivergence),
			Requests:   divergenceRequests,
		})
		if err != nil {
			return branchMetadataOutput{}, fmt.Errorf("fail to fetch commit divergences: %w", err)
		}
	}

	return branchMetadataOutput{
		checkSummary:  checkSummary,
		branchRuleMap: branchRuleMap,
		pullReqMap:    pullReqMap,
		divergences:   divergences,
	}, nil
}

type branchMetadataOutput struct {
	checkSummary  map[sha.SHA]types.CheckCountSummary
	branchRuleMap map[string][]types.RuleInfo
	pullReqMap    map[string][]*types.PullReq
	divergences   *git.GetCommitDivergencesOutput
}

func (metadata branchMetadataOutput) apply(
	idx int,
	branch *types.BranchExtended,
) {
	if metadata.checkSummary != nil {
		branch.CheckSummary = ptr.Of(metadata.checkSummary[branch.SHA])
	}

	if metadata.branchRuleMap != nil {
		branch.Rules = metadata.branchRuleMap[branch.Name]
	}

	if metadata.pullReqMap != nil {
		branch.PullRequests = metadata.pullReqMap[branch.Name]
	}

	if metadata.divergences != nil {
		branch.CommitDivergence = ptr.Of(types.CommitDivergence(metadata.divergences.Divergences[idx]))
	}
}

func mapToRPCBranchSortOption(o enum.BranchSortOption) git.BranchSortOption {
	switch o {
	case enum.BranchSortOptionDate:
		return git.BranchSortOptionDate
	case enum.BranchSortOptionName:
		return git.BranchSortOptionName
	case enum.BranchSortOptionDefault:
		return git.BranchSortOptionDefault
	default:
		// no need to error out - just use default for sorting
		return git.BranchSortOptionDefault
	}
}

func mapToRPCSortOrder(o enum.Order) git.SortOrder {
	switch o {
	case enum.OrderAsc:
		return git.SortOrderAsc
	case enum.OrderDesc:
		return git.SortOrderDesc
	case enum.OrderDefault:
		return git.SortOrderDefault
	default:
		// no need to error out - just use default for sorting
		return git.SortOrderDefault
	}
}
