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

package mergequeue

import (
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestIsEnqueued(t *testing.T) {
	s := &Service{}

	tests := []struct {
		name string
		pr   *types.PullReq
		want bool
	}{
		{
			name: "open-with-merge-queue-substate",
			pr: &types.PullReq{
				State:    enum.PullReqStateOpen,
				SubState: enum.PullReqSubStateMergeQueue,
			},
			want: true,
		},
		{
			name: "open-no-substate",
			pr: &types.PullReq{
				State:    enum.PullReqStateOpen,
				SubState: enum.PullReqSubStateNone,
			},
			want: false,
		},
		{
			name: "open-auto-merge-substate",
			pr: &types.PullReq{
				State:    enum.PullReqStateOpen,
				SubState: enum.PullReqSubStateAutoMerge,
			},
			want: false,
		},
		{
			name: "closed-with-merge-queue-substate",
			pr: &types.PullReq{
				State:    enum.PullReqStateClosed,
				SubState: enum.PullReqSubStateMergeQueue,
			},
			want: false,
		},
		{
			name: "merged-with-merge-queue-substate",
			pr: &types.PullReq{
				State:    enum.PullReqStateMerged,
				SubState: enum.PullReqSubStateMergeQueue,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.IsEnqueued(tt.pr)
			if got != tt.want {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}
