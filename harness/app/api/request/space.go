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

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamSpaceRef = "space_ref"

	QueryParamIncludeSubspaces = "include_subspaces"
)

func GetSpaceRefFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamSpaceRef)
}

// ParseSortSpace extracts the space sort parameter from the url.
func ParseSortSpace(r *http.Request) enum.SpaceAttr {
	return enum.ParseSpaceAttr(
		r.URL.Query().Get(QueryParamSort),
	)
}

// ParseSpaceFilter extracts the space filter from the url.
func ParseSpaceFilter(r *http.Request) (*types.SpaceFilter, error) {
	// recursive is optional to get sapce and its subsapces recursively.
	recursive, err := ParseRecursiveFromQuery(r)
	if err != nil {
		return nil, err
	}

	// deletedBeforeOrAt is optional to retrieve spaces deleted before or at the specified timestamp.
	var deletedBeforeOrAt *int64
	deletionVal, ok, err := GetDeletedBeforeOrAtFromQuery(r)
	if err != nil {
		return nil, err
	}
	if ok {
		deletedBeforeOrAt = &deletionVal
	}

	// deletedAt is optional to retrieve spaces deleted at the specified timestamp.
	var deletedAt *int64
	deletedAtVal, ok, err := GetDeletedAtFromQuery(r)
	if err != nil {
		return nil, err
	}
	if ok {
		deletedAt = &deletedAtVal
	}

	return &types.SpaceFilter{
		Query:             ParseQuery(r),
		Order:             ParseOrder(r),
		Page:              ParsePage(r),
		Sort:              ParseSortSpace(r),
		Size:              ParseLimit(r),
		Recursive:         recursive,
		DeletedAt:         deletedAt,
		DeletedBeforeOrAt: deletedBeforeOrAt,
	}, nil
}

func GetIncludeSubspacesFromQuery(r *http.Request) (bool, error) {
	v, err := QueryParamAsBoolOrDefault(r, QueryParamIncludeSubspaces, false)
	if err != nil {
		return false, fmt.Errorf("failed to parse include subspaces parameter: %w", err)
	}

	return v, nil
}
