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

package request

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/api/usererror"
	gittypes "github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	HeaderParamGitProtocol = "Git-Protocol"

	PathParamCommitSHA = "commit_sha"

	QueryParamGitRef             = "git_ref"
	QueryParamIncludeCommit      = "include_commit"
	QueryParamIncludeDirectories = "include_directories"
	QueryParamLineFrom           = "line_from"
	QueryParamLineTo             = "line_to"
	QueryParamPath               = "path"
	QueryParamSince              = "since"
	QueryParamUntil              = "until"
	QueryParamCommitter          = "committer"
	QueryParamIncludeStats       = "include_stats"
	QueryParamInternal           = "internal"
	QueryParamService            = "service"
	QueryParamCommitSHA          = "commit_sha"

	QueryParamIncludeChecks   = "include_checks"
	QueryParamIncludeRules    = "include_rules"
	QueryParamIncludePullReqs = "include_pullreqs"
	QueryParamMaxDivergence   = "max_divergence"
)

func GetGitRefFromQueryOrDefault(r *http.Request, deflt string) string {
	return QueryParamOrDefault(r, QueryParamGitRef, deflt)
}

func GetIncludeCommitFromQueryOrDefault(r *http.Request, deflt bool) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamIncludeCommit, deflt)
}

func GetIncludeChecksFromQueryOrDefault(r *http.Request, deflt bool) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamIncludeChecks, deflt)
}

func GetIncludeRulesFromQueryOrDefault(r *http.Request, deflt bool) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamIncludeRules, deflt)
}

func GetIncludePullReqsFromQueryOrDefault(r *http.Request, deflt bool) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamIncludePullReqs, deflt)
}

func GetMaxDivergenceFromQueryOrDefault(r *http.Request, deflt int64) (int64, error) {
	return QueryParamAsPositiveInt64OrDefault(r, QueryParamMaxDivergence, deflt)
}

func GetIncludeDirectoriesFromQueryOrDefault(r *http.Request, deflt bool) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamIncludeDirectories, deflt)
}

func GetCommitSHAFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamCommitSHA)
}

// ParseSortBranch extracts the branch sort parameter from the url.
func ParseSortBranch(r *http.Request) enum.BranchSortOption {
	return enum.ParseBranchSortOption(
		r.URL.Query().Get(QueryParamSort),
	)
}

func ParseBranchMetadataOptions(r *http.Request) (types.BranchMetadataOptions, error) {
	includeChecks, err := GetIncludeChecksFromQueryOrDefault(r, false)
	if err != nil {
		return types.BranchMetadataOptions{}, err
	}

	includeRules, err := GetIncludeRulesFromQueryOrDefault(r, false)
	if err != nil {
		return types.BranchMetadataOptions{}, err
	}

	includePullReqs, err := GetIncludePullReqsFromQueryOrDefault(r, false)
	if err != nil {
		return types.BranchMetadataOptions{}, err
	}

	maxDivergence, err := GetMaxDivergenceFromQueryOrDefault(r, 0)
	if err != nil {
		return types.BranchMetadataOptions{}, err
	}

	return types.BranchMetadataOptions{
		IncludeChecks:   includeChecks,
		IncludeRules:    includeRules,
		IncludePullReqs: includePullReqs,
		MaxDivergence:   int(maxDivergence),
	}, nil
}

// ParseBranchFilter extracts the branch filter from the url.
func ParseBranchFilter(r *http.Request) (*types.BranchFilter, error) {
	includeCommit, err := GetIncludeCommitFromQueryOrDefault(r, false)
	if err != nil {
		return nil, err
	}

	metadataOptions, err := ParseBranchMetadataOptions(r)
	if err != nil {
		return nil, err
	}

	return &types.BranchFilter{
		Query:                 ParseQuery(r),
		Sort:                  ParseSortBranch(r),
		Order:                 ParseOrder(r),
		Page:                  ParsePage(r),
		Size:                  ParseLimit(r),
		IncludeCommit:         includeCommit,
		BranchMetadataOptions: metadataOptions,
	}, nil
}

// ParseSortTag extracts the tag sort parameter from the url.
func ParseSortTag(r *http.Request) enum.TagSortOption {
	return enum.ParseTagSortOption(
		r.URL.Query().Get(QueryParamSort),
	)
}

// ParseTagFilter extracts the tag filter from the url.
func ParseTagFilter(r *http.Request) *types.TagFilter {
	return &types.TagFilter{
		Query: ParseQuery(r),
		Sort:  ParseSortTag(r),
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Size:  ParseLimit(r),
	}
}

// ParseCommitFilter extracts the commit filter from the url.
func ParseCommitFilter(r *http.Request) (*types.CommitFilter, error) {
	// since is optional, skipped if set to 0
	since, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamSince, 0)
	if err != nil {
		return nil, err
	}
	// until is optional, skipped if set to 0
	until, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamUntil, 0)
	if err != nil {
		return nil, err
	}
	includeStats, err := QueryParamAsBoolOrDefault(r, QueryParamIncludeStats, false)
	if err != nil {
		return nil, err
	}

	return &types.CommitFilter{
		After: QueryParamOrDefault(r, QueryParamAfter, ""),
		PaginationFilter: types.PaginationFilter{
			Page:  ParsePage(r),
			Limit: ParseLimit(r),
		},
		Path:         QueryParamOrDefault(r, QueryParamPath, ""),
		Since:        since,
		Until:        until,
		Committer:    QueryParamOrDefault(r, QueryParamCommitter, ""),
		IncludeStats: includeStats,
	}, nil
}

// GetGitProtocolFromHeadersOrDefault returns the git protocol from the request headers.
func GetGitProtocolFromHeadersOrDefault(r *http.Request, deflt string) string {
	return GetHeaderOrDefault(r, HeaderParamGitProtocol, deflt)
}

// GetGitServiceTypeFromQuery returns the git service type from the request query.
func GetGitServiceTypeFromQuery(r *http.Request) (enum.GitServiceType, error) {
	// git prefixes the service names with "git-" in the query
	const gitPrefix = "git-"

	val, err := QueryParamOrError(r, QueryParamService)
	if err != nil {
		return "", fmt.Errorf("failed to get param from query: %w", err)
	}
	if !strings.HasPrefix(val, gitPrefix) {
		return "", usererror.BadRequestf("not a git service type: %q", val)
	}

	return enum.ParseGitServiceType(val[len(gitPrefix):])
}

func GetFileDiffFromQuery(r *http.Request) (files gittypes.FileDiffRequests) {
	paths, _ := QueryParamList(r, "path")
	ranges, _ := QueryParamList(r, "range")

	for i, filepath := range paths {
		start := 0
		end := 0
		if i < len(ranges) {
			linesRange := ranges[i]
			parts := strings.Split(linesRange, ":")
			if len(parts) > 1 {
				end, _ = strconv.Atoi(parts[1])
			}
			if len(parts) > 0 {
				start, _ = strconv.Atoi(parts[0])
			}
		}
		files = append(files, gittypes.FileDiffRequest{
			Path:      filepath,
			StartLine: start,
			EndLine:   end,
		})
	}
	return
}

func GetCommitSHAFromQueryOrDefault(r *http.Request) string {
	return QueryParamOrDefault(r, QueryParamCommitSHA, "")
}
