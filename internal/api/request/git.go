// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
		r.FormValue(QueryParamSort),
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
		r.FormValue(QueryParamSort),
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
