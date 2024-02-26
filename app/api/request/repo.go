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
	"net/url"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamRepoRef    = "repo_ref"
	QueryParamRepoID    = "repo_id"
	QueryParamRecursive = "recursive"
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
	return enum.ParseRepoAttr(
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

// ParseRecursiveFromQuery extracts the recursive option from the URL query.
func ParseRecursiveFromQuery(r *http.Request) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamRecursive, false)
}
