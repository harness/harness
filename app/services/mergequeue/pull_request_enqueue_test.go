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

func TestVerifyIfMergeQueueable(t *testing.T) {
	s := &Service{}

	nowMilli := int64(1000000)

	tests := []struct {
		name    string
		pr      *types.PullReq
		wantErr bool
	}{
		{
			name: "open-pr-no-substate",
			pr: &types.PullReq{
				State:    enum.PullReqStateOpen,
				SubState: enum.PullReqSubStateNone,
				IsDraft:  false,
			},
			wantErr: false,
		},
		{
			name: "merged-pr",
			pr: &types.PullReq{
				State:    enum.PullReqStateMerged,
				SubState: enum.PullReqSubStateNone,
				Merged:   &nowMilli,
			},
			wantErr: true,
		},
		{
			name: "closed-pr",
			pr: &types.PullReq{
				State:    enum.PullReqStateClosed,
				SubState: enum.PullReqSubStateNone,
			},
			wantErr: true,
		},
		{
			name: "draft-pr",
			pr: &types.PullReq{
				State:    enum.PullReqStateOpen,
				SubState: enum.PullReqSubStateNone,
				IsDraft:  true,
			},
			wantErr: true,
		},
		{
			name: "already-in-merge-queue",
			pr: &types.PullReq{
				State:    enum.PullReqStateOpen,
				SubState: enum.PullReqSubStateMergeQueue,
			},
			wantErr: true,
		},
		{
			name: "auto-merge-substate",
			pr: &types.PullReq{
				State:    enum.PullReqStateOpen,
				SubState: enum.PullReqSubStateAutoMerge,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.VerifyIfMergeQueueable(tt.pr)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}
