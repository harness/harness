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

package pullreq

import (
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestBuildReviewerTypeByID(t *testing.T) {
	t.Parallel()

	principal := func(id int64) *types.PrincipalInfo { return &types.PrincipalInfo{ID: id} }

	tests := []struct {
		name      string
		requested []int64
		owner     map[int64]*types.PrincipalInfo
		dflt      map[int64]*types.PrincipalInfo
		want      map[int64]enum.PullReqReviewerType
	}{
		{
			name:      "disjoint sources keep their own type",
			requested: []int64{1},
			owner:     map[int64]*types.PrincipalInfo{2: principal(2)},
			dflt:      map[int64]*types.PrincipalInfo{3: principal(3)},
			want: map[int64]enum.PullReqReviewerType{
				1: enum.PullReqReviewerTypeRequested,
				2: enum.PullReqReviewerTypeCodeOwners,
				3: enum.PullReqReviewerTypeDefault,
			},
		},
		{
			// Guards against a regression that reverses the loop order: the same
			// ID in all three sources must resolve to the highest precedence.
			name:      "requested wins over code_owners and default",
			requested: []int64{1},
			owner:     map[int64]*types.PrincipalInfo{1: principal(1)},
			dflt:      map[int64]*types.PrincipalInfo{1: principal(1)},
			want: map[int64]enum.PullReqReviewerType{
				1: enum.PullReqReviewerTypeRequested,
			},
		},
		{
			name:      "code_owners wins over default",
			requested: nil,
			owner:     map[int64]*types.PrincipalInfo{1: principal(1)},
			dflt:      map[int64]*types.PrincipalInfo{1: principal(1)},
			want: map[int64]enum.PullReqReviewerType{
				1: enum.PullReqReviewerTypeCodeOwners,
			},
		},
		{
			name:      "no reviewers yields empty map",
			requested: nil,
			owner:     nil,
			dflt:      nil,
			want:      map[int64]enum.PullReqReviewerType{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildReviewerTypeByID(tc.requested, tc.owner, tc.dflt)
			require.Equal(t, tc.want, got)
		})
	}
}
