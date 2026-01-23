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
)

const PathParamAutolinkID = "autolink_id"

func ParseAutolinkFilter(r *http.Request) (*types.AutoLinkFilter, error) {
	// inherited is used to list autolinks from parent scopes
	inherited, err := ParseInheritedFromQuery(r)
	if err != nil {
		return nil, err
	}

	return &types.AutoLinkFilter{
		ListQueryFilter: ParseListQueryFilterFromRequest(r),
		Inherited:       inherited,
	}, nil
}

func GetAutolinkIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamAutolinkID)
}
