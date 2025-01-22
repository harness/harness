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
	PathParamRepoRef = "repo_ref"
	QueryParamRepoID = "repo_id"
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

// ParseRepoFilter extracts the repository filter from the url.
func ParseRepoFilter(r *http.Request) (*types.RepoFilter, error) {
	// recursive is optional to get all repos in a sapce and its subsapces recursively.
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

	return &types.RepoFilter{
		Query:             ParseQuery(r),
		Order:             ParseOrder(r),
		Page:              ParsePage(r),
		Sort:              ParseSortRepo(r),
		Size:              ParseLimit1000(r),
		Recursive:         recursive,
		DeletedAt:         deletedAt,
		DeletedBeforeOrAt: deletedBeforeOrAt,
	}, nil
}
