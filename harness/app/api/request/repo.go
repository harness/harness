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
	"slices"
	"strings"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamRepoRef        = "repo_ref"
	QueryParamOnlyFavorites = "only_favorites"
	QueryParamTag           = "tag"
)

func GetRepoRefFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamRepoRef)
}

// ParseSortRepo extracts the repo sort parameter from the url.
func ParseSortRepo(r *http.Request) enum.RepoAttr {
	return enum.ParseRepoAttr(
		r.URL.Query().Get(QueryParamSort),
	)
}

// ParseOnlyFavoritesFromQuery extracts the only_favorites option from the URL.
func ParseOnlyFavoritesFromQuery(r *http.Request) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamOnlyFavorites, false)
}

func ParseTagsFromQuery(r *http.Request) map[string][]string {
	tags, ok := QueryParamList(r, QueryParamTag)
	if !ok {
		return nil
	}

	result := make(map[string][]string)
	for _, t := range tags {
		before, after, found := strings.Cut(t, ":")
		key := strings.TrimSpace(before)

		// key without value
		if !found {
			// dominates everything else → just set to nil
			result[key] = nil
			continue
		}

		// key with value
		if _, ok := result[key]; !ok || result[key] != nil {
			val := strings.TrimSpace(after)
			result[key] = append(result[key], val)
		}
	}

	for key, values := range result {
		slices.Sort(values)
		result[key] = slices.Compact(values)
	}

	return result
}

// ParseRepoFilter extracts the repository filter from the url.
func ParseRepoFilter(r *http.Request, session *auth.Session) (*types.RepoFilter, error) {
	// recursive is optional to get all repos in a space and its subspaces recursively.
	recursive, err := ParseRecursiveFromQuery(r)
	if err != nil {
		return nil, err
	}

	// deletedBeforeOrAt is optional to retrieve repos deleted before or at the specified timestamp.
	var deletedBeforeOrAt *int64
	deletionVal, ok, err := GetDeletedBeforeOrAtFromQuery(r)
	if err != nil {
		return nil, err
	}
	if ok {
		deletedBeforeOrAt = &deletionVal
	}

	// deletedAt is optional to retrieve repos deleted at the specified timestamp.
	var deletedAt *int64
	deletedAtVal, ok, err := GetDeletedAtFromQuery(r)
	if err != nil {
		return nil, err
	}
	if ok {
		deletedAt = &deletedAtVal
	}

	order := ParseOrder(r)
	if order == enum.OrderDefault {
		order = enum.OrderAsc
	}

	onlyFavorites, err := ParseOnlyFavoritesFromQuery(r)
	if err != nil {
		return nil, err
	}
	var onlyFavoritesFor *int64
	if onlyFavorites {
		onlyFavoritesFor = &session.Principal.ID
	}

	return &types.RepoFilter{
		Query:             ParseQuery(r),
		Order:             order,
		Page:              ParsePage(r),
		Sort:              ParseSortRepo(r),
		Size:              ParseLimit(r),
		Recursive:         recursive,
		DeletedAt:         deletedAt,
		DeletedBeforeOrAt: deletedBeforeOrAt,
		OnlyFavoritesFor:  onlyFavoritesFor,
		Tags:              ParseTagsFromQuery(r),
	}, nil
}
