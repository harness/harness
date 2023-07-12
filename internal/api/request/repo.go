// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"
	"net/url"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamRepoRef = "repo_ref"
	QueryParamRepoID = "repo_id"
)

func GetRepoRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PathParamRepoRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped
	return url.PathUnescape(rawRef)
}

// GetRepoIDFromQuery returns the repo id from the request query.
func GetRepoIDFromQuery(r *http.Request) (int64, error) {
	return QueryParamAsPositiveInt64(r, QueryParamRepoID)
}

// ParseSortRepo extracts the repo sort parameter from the url.
func ParseSortRepo(r *http.Request) enum.RepoAttr {
	return enum.ParseRepoAtrr(
		r.FormValue(QueryParamSort),
	)
}

// ParseRepoFilter extracts the repository filter from the url.
func ParseRepoFilter(r *http.Request) *types.RepoFilter {
	return &types.RepoFilter{
		Query: ParseQuery(r),
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Sort:  ParseSortRepo(r),
		Size:  ParseLimit(r),
	}
}
