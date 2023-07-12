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
)

func GetRepoRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PathParamRepoRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped
	return url.PathUnescape(rawRef)
}

// ParseSortRepo extracts the repo sort parameter from the url.
func ParseSortRepo(r *http.Request) enum.RepoAttr {
	return enum.ParseRepoAtrr(
		r.URL.Query().Get(QueryParamSort),
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
