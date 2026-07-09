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

	"golang.org/x/exp/slices"
)

func TestReviewDecisionRank(t *testing.T) {
	tests := []struct {
		name     string
		decision enum.PullReqReviewDecision
		want     int
	}{
		{name: "changereq", decision: enum.PullReqReviewDecisionChangeReq, want: 0},
		{name: "approved", decision: enum.PullReqReviewDecisionApproved, want: 1},
		{name: "reviewed", decision: enum.PullReqReviewDecisionReviewed, want: 2},
		{name: "pending", decision: enum.PullReqReviewDecisionPending, want: 3},
		{name: "unknown", decision: enum.PullReqReviewDecision("bogus"), want: 4},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := reviewDecisionRank(test.decision); got != test.want {
				t.Errorf("want=%d got=%d", test.want, got)
			}
		})
	}
}

func TestSortReviewersByDecision(t *testing.T) {
	type reviewer struct {
		id       int64
		decision enum.PullReqReviewDecision
	}

	tests := []struct {
		name  string
		input []reviewer
		want  []int64
	}{
		{
			name: "mixed decisions ordered by priority",
			input: []reviewer{
				{id: 1, decision: enum.PullReqReviewDecisionPending},
				{id: 2, decision: enum.PullReqReviewDecisionApproved},
				{id: 3, decision: enum.PullReqReviewDecisionChangeReq},
				{id: 4, decision: enum.PullReqReviewDecisionReviewed},
			},
			want: []int64{3, 2, 4, 1},
		},
		{
			name: "equal decisions preserve input order (stable)",
			input: []reviewer{
				{id: 1, decision: enum.PullReqReviewDecisionApproved},
				{id: 2, decision: enum.PullReqReviewDecisionApproved},
				{id: 3, decision: enum.PullReqReviewDecisionChangeReq},
				{id: 4, decision: enum.PullReqReviewDecisionApproved},
			},
			want: []int64{3, 1, 2, 4},
		},
		{
			name:  "empty list",
			input: []reviewer{},
			want:  nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			list := make([]*types.PullReqReviewer, len(test.input))
			for i := range test.input {
				list[i] = &types.PullReqReviewer{
					PrincipalID:    test.input[i].id,
					ReviewDecision: test.input[i].decision,
				}
			}

			sortReviewersByDecision(list)

			var got []int64
			for _, r := range list {
				got = append(got, r.PrincipalID)
			}

			if !slices.Equal(test.want, got) {
				t.Errorf("want=%v got=%v", test.want, got)
			}
		})
	}
}

func TestSortUserGroupReviewersByDecision(t *testing.T) {
	type reviewer struct {
		id       int64
		decision enum.PullReqReviewDecision
	}

	tests := []struct {
		name  string
		input []reviewer
		want  []int64
	}{
		{
			name: "mixed decisions ordered by priority",
			input: []reviewer{
				{id: 1, decision: enum.PullReqReviewDecisionPending},
				{id: 2, decision: enum.PullReqReviewDecisionChangeReq},
				{id: 3, decision: enum.PullReqReviewDecisionApproved},
			},
			want: []int64{2, 3, 1},
		},
		{
			name: "equal decisions preserve input order (stable)",
			input: []reviewer{
				{id: 1, decision: enum.PullReqReviewDecisionPending},
				{id: 2, decision: enum.PullReqReviewDecisionPending},
				{id: 3, decision: enum.PullReqReviewDecisionApproved},
			},
			want: []int64{3, 1, 2},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			list := make([]*types.UserGroupReviewer, len(test.input))
			for i := range test.input {
				list[i] = &types.UserGroupReviewer{
					UserGroupID: test.input[i].id,
					Decision:    test.input[i].decision,
				}
			}

			sortUserGroupReviewersByDecision(list)

			var got []int64
			for _, r := range list {
				got = append(got, r.UserGroupID)
			}

			if !slices.Equal(test.want, got) {
				t.Errorf("want=%v got=%v", test.want, got)
			}
		})
	}
}
