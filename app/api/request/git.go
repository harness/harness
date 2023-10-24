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
	"net/http"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	QueryParamGitRef        = "git_ref"
	QueryParamIncludeCommit = "include_commit"
	PathParamCommitSHA      = "commit_sha"
	QueryParamLineFrom      = "line_from"
	QueryParamLineTo        = "line_to"
	QueryParamPath          = "path"
	QueryParamSince         = "since"
	QueryParamUntil         = "until"
	QueryParamCommitter     = "committer"
	QueryParamInternal      = "internal"
)

func GetGitRefFromQueryOrDefault(r *http.Request, deflt string) string {
	return QueryParamOrDefault(r, QueryParamGitRef, deflt)
}

func GetIncludeCommitFromQueryOrDefault(r *http.Request, deflt bool) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamIncludeCommit, deflt)
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

// ParseBranchFilter extracts the branch filter from the url.
func ParseBranchFilter(r *http.Request) *types.BranchFilter {
	return &types.BranchFilter{
		Query: ParseQuery(r),
		Sort:  ParseSortBranch(r),
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Size:  ParseLimit(r),
	}
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
	return &types.CommitFilter{
		After: QueryParamOrDefault(r, QueryParamAfter, ""),
		PaginationFilter: types.PaginationFilter{
			Page:  ParsePage(r),
			Limit: ParseLimit(r),
		},
		Path:      QueryParamOrDefault(r, QueryParamPath, ""),
		Since:     since,
		Until:     until,
		Committer: QueryParamOrDefault(r, QueryParamCommitter, ""),
	}, nil
}

// GetInternalFromQueryOrDefault returns the internal flag from the request query.
func GetInternalFromQueryOrDefault(r *http.Request, dflt bool) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamInternal, dflt)
}
