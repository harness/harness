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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
)

// ParseRepoActivityFilter parses repository push activity filters from query parameters.
func ParseRepoActivityFilter(r *http.Request) (*types.RepoActivityFilter, error) {
	after, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamAfter, 0)
	if err != nil {
		return nil, err
	}

	before, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamBefore, 0)
	if err != nil {
		return nil, err
	}

	// Only validate after <= before if both are provided (non-zero)
	if after > 0 && before > 0 && before < after {
		return nil, usererror.BadRequestf(
			"Parameter '%s' must be greater than or equal to '%s'.",
			QueryParamBefore, QueryParamAfter,
		)
	}

	return &types.RepoActivityFilter{
		After:  after,
		Before: before,
		Page:   ParsePage(r),
		Size:   ParseLimit(r),
	}, nil
}
