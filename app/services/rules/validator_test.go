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
	"testing"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestValidateUsers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		ruleUserIDs []int64
		userMap     map[int64]*types.PrincipalInfo
		wantErr     string
	}{
		{
			name:        "all users exist",
			ruleUserIDs: []int64{1, 2},
			userMap: map[int64]*types.PrincipalInfo{
				1: {ID: 1},
				2: {ID: 2},
			},
		},
		{
			name:        "missing users",
			ruleUserIDs: []int64{1, 2, 3},
			userMap: map[int64]*types.PrincipalInfo{
				1: {ID: 1},
			},
			wantErr: "The following users could not be found: [2 3]. " +
				"Remove or replace them in the bypass or default reviewer list.",
		},
		{
			name:        "duplicate missing user",
			ruleUserIDs: []int64{2, 2},
			userMap:     map[int64]*types.PrincipalInfo{},
			wantErr: "The following users could not be found: [2]. " +
				"Remove or replace them in the bypass or default reviewer list.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateUsers(test.ruleUserIDs, test.userMap)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, test.wantErr)
		})
	}
}
