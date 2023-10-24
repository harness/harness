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

// ParseCheckListOptions extracts the status check list API options from the url.
func ParseCheckListOptions(r *http.Request) types.CheckListOptions {
	return types.CheckListOptions{
		ListQueryFilter: ParseListQueryFilterFromRequest(r),
	}
}

// ParseCheckRecentOptions extracts the list recent status checks API options from the url.
func ParseCheckRecentOptions(r *http.Request) (types.CheckRecentOptions, error) {
	since, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamSince, 0)
	if err != nil {
		return types.CheckRecentOptions{}, err
	}

	return types.CheckRecentOptions{
		Query: ParseQuery(r),
		Since: since,
	}, nil
}
