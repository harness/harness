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

package protection

import (
	"fmt"

	"github.com/harness/gitness/types"

	"golang.org/x/exp/slices"
)

type DefBypass struct {
	UserIDs    []int64 `json:"user_ids,omitempty"`
	RepoOwners bool    `json:"repo_owners,omitempty"`
}

func (v DefBypass) matches(actor *types.Principal, isRepoOwner bool) bool {
	return actor != nil &&
		(v.RepoOwners && isRepoOwner ||
			slices.Contains(v.UserIDs, actor.ID))
}

func (v DefBypass) Sanitize() error {
	if err := validateIDSlice(v.UserIDs); err != nil {
		return fmt.Errorf("user IDs error: %w", err)
	}

	return nil
}
