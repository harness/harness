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

package rules

import (
	"context"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
)

type Validator interface {
	Validate(context.Context, []int64, map[int64]*types.PrincipalInfo) error
}

type validator struct{}

func (v validator) Validate(
	_ context.Context,
	ruleUserIDs []int64,
	userMap map[int64]*types.PrincipalInfo,
) error {
	return ValidateUsers(ruleUserIDs, userMap)
}

func ValidateUsers(
	ruleUserIDs []int64,
	userMap map[int64]*types.PrincipalInfo,
) error {
	var missing []int64
	idSet := make(map[int64]struct{}, len(ruleUserIDs))
	for _, id := range ruleUserIDs {
		if _, seen := idSet[id]; seen {
			continue // already checked
		}
		idSet[id] = struct{}{}
		if _, exists := userMap[id]; !exists {
			missing = append(missing, id)
		}
	}

	if len(missing) > 0 {
		return usererror.BadRequestf(
			"unknown users in bypass and/or reviewer list: %v", missing,
		)
	}

	return nil
}
