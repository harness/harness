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
	PathParamSpaceRef = "space_ref"
)

func GetSpaceRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PathParamSpaceRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped and lower
	return url.PathUnescape(rawRef)
}

// ParseSortSpace extracts the space sort parameter from the url.
func ParseSortSpace(r *http.Request) enum.SpaceAttr {
	return enum.ParseSpaceAttr(
		r.URL.Query().Get(QueryParamSort),
	)
}

// ParseSpaceFilter extracts the space filter from the url.
func ParseSpaceFilter(r *http.Request) *types.SpaceFilter {
	return &types.SpaceFilter{
		Query: ParseQuery(r),
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Sort:  ParseSortSpace(r),
		Size:  ParseLimit(r),
	}
}
