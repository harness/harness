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

func TestRemoveDeletedComments(t *testing.T) {
	type activity struct {
		id       int64
		kind     enum.PullReqActivityKind
		order    int64
		subOrder int64
		deleted  *int64
	}

	var n int64
	d := &n

	tests := []struct {
		name  string
		input []activity
		want  []int64
	}{
		{
			name: "nothing-deleted",
			input: []activity{
				{id: 1, kind: enum.PullReqActivityKindComment, order: 0, subOrder: 0, deleted: nil},
				{id: 2, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 0, deleted: nil},
				{id: 3, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 1, deleted: nil},
				{id: 4, kind: enum.PullReqActivityKindComment, order: 2, subOrder: 0, deleted: nil},
				{id: 5, kind: enum.PullReqActivityKindComment, order: 2, subOrder: 1, deleted: nil},
			},
			want: []int64{1, 2, 3, 4, 5},
		},
		{
			name: "deleted-thread",
			input: []activity{
				{id: 1, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 0, deleted: d},
				{id: 2, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 1, deleted: d},
				{id: 3, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 2, deleted: d},
			},
			want: []int64{},
		},
		{
			name: "deleted-top-level-replies-not",
			input: []activity{
				{id: 1, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 0, deleted: d},
				{id: 2, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 1, deleted: nil},
				{id: 3, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 2, deleted: d},
			},
			want: []int64{1, 2, 3},
		},
		{
			name: "deleted-all-replies",
			input: []activity{
				{id: 1, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 0, deleted: nil},
				{id: 2, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 1, deleted: d},
				{id: 3, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 2, deleted: d},
			},
			want: []int64{1, 2, 3},
		},
		{
			name: "complex",
			input: []activity{
				// kind=system, deleted, must not be removed
				{id: 1, kind: enum.PullReqActivityKindSystem, order: 0, subOrder: 0, deleted: d},
				// thread size=1, not deleted
				{id: 2, kind: enum.PullReqActivityKindComment, order: 1, subOrder: 0, deleted: nil},
				// thread size=1, deleted
				{id: 3, kind: enum.PullReqActivityKindComment, order: 2, subOrder: 0, deleted: d},
				// kind=system, not deleted, must not be removed
				{id: 4, kind: enum.PullReqActivityKindSystem, order: 3, subOrder: 0, deleted: nil},
				// thread size=2, not deleted
				{id: 5, kind: enum.PullReqActivityKindComment, order: 4, subOrder: 0, deleted: nil},
				{id: 6, kind: enum.PullReqActivityKindComment, order: 4, subOrder: 1, deleted: d},
				// thread size=2, change comment, deleted
				{id: 7, kind: enum.PullReqActivityKindChangeComment, order: 5, subOrder: 0, deleted: d},
				{id: 8, kind: enum.PullReqActivityKindChangeComment, order: 5, subOrder: 1, deleted: d},
			},
			want: []int64{1, 2, 4, 5, 6},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			list := make([]*types.PullReqActivity, len(test.input))
			for i := range test.input {
				list[i] = &types.PullReqActivity{
					ID:       test.input[i].id,
					Kind:     test.input[i].kind,
					Order:    test.input[i].order,
					SubOrder: test.input[i].subOrder,
					Deleted:  test.input[i].deleted,
				}
			}

			var got []int64
			for _, act := range removeDeletedComments(list) {
				got = append(got, act.ID)
			}

			if !slices.Equal(test.want, got) {
				t.Errorf("want=%v got=%v", test.want, got)
			}
		})
	}
}
