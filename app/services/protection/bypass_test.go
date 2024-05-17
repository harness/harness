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
	"testing"

	"github.com/harness/gitness/types"
)

func TestBranch_matches(t *testing.T) {
	user := &types.Principal{ID: 42}
	admin := &types.Principal{ID: 66, Admin: true}

	tests := []struct {
		name   string
		bypass DefBypass
		actor  *types.Principal
		owner  bool
		exp    bool
	}{
		{
			name:   "empty",
			bypass: DefBypass{UserIDs: nil, RepoOwners: false},
			actor:  user,
			exp:    false,
		},
		{
			name:   "admin-no-owner",
			bypass: DefBypass{UserIDs: nil, RepoOwners: true},
			actor:  admin,
			owner:  false,
			exp:    false,
		},
		{
			name:   "repo-owners-false",
			bypass: DefBypass{UserIDs: nil, RepoOwners: false},
			actor:  user,
			owner:  true,
			exp:    false,
		},
		{
			name:   "repo-owners-true",
			bypass: DefBypass{UserIDs: nil, RepoOwners: true},
			actor:  user,
			owner:  true,
			exp:    true,
		},
		{
			name:   "selected-false",
			bypass: DefBypass{UserIDs: []int64{1, 66}, RepoOwners: false},
			actor:  user,
			exp:    false,
		},
		{
			name:   "selected-true",
			bypass: DefBypass{UserIDs: []int64{1, 42, 66}, RepoOwners: false},
			actor:  user,
			exp:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.bypass.Sanitize(); err != nil {
				t.Errorf("invalid: %s", err.Error())
			}

			if want, got := test.exp, test.bypass.matches(test.actor, test.owner); want != got {
				t.Errorf("want=%t got=%t", want, got)
			}
		})
	}
}
