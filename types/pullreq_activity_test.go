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

package types

import (
	"testing"

	"github.com/harness/gitness/types/enum"
)

func TestPullReqActivity_ReviewComment(t *testing.T) {
	resolved := int64(1)
	activity := func(
		actType enum.PullReqActivityType,
		kind enum.PullReqActivityKind,
		f func(act *PullReqActivity),
	) PullReqActivity {
		act := PullReqActivity{Type: actType, Kind: kind}
		if actType == enum.PullReqActivityTypeCodeComment {
			act.CodeComment = &CodeCommentFields{}
		}
		if f != nil {
			f(&act)
		}
		return act
	}

	tests := []struct {
		name          string
		act           PullReqActivity
		wantBlocking  bool
		wantReplyable bool
		wantCode      bool
	}{
		{
			name:          "comment",
			act:           activity(enum.PullReqActivityTypeComment, enum.PullReqActivityKindComment, nil),
			wantBlocking:  true,
			wantReplyable: true,
		},
		{
			name:          "code comment",
			act:           activity(enum.PullReqActivityTypeCodeComment, enum.PullReqActivityKindChangeComment, nil),
			wantBlocking:  true,
			wantReplyable: true,
			wantCode:      true,
		},
		{
			name: "review comment",
			act:  activity(enum.PullReqActivityTypeComment, enum.PullReqActivityKindReviewComment, nil),
		},
		{
			name:     "review comment on code",
			act:      activity(enum.PullReqActivityTypeCodeComment, enum.PullReqActivityKindReviewComment, nil),
			wantCode: true,
		},
		{
			name: "resolved comment",
			act: activity(enum.PullReqActivityTypeComment, enum.PullReqActivityKindComment,
				func(act *PullReqActivity) { act.Resolved = &resolved }),
			wantReplyable: true,
		},
		{
			name: "reply",
			act: activity(enum.PullReqActivityTypeComment, enum.PullReqActivityKindComment,
				func(act *PullReqActivity) { act.SubOrder = 1 }),
		},
		{
			name: "system",
			act:  activity(enum.PullReqActivityTypeTitleChange, enum.PullReqActivityKindSystem, nil),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.act.IsBlocking(); got != test.wantBlocking {
				t.Errorf("IsBlocking() = %v, want %v", got, test.wantBlocking)
			}
			if got := test.act.IsReplyable(); got != test.wantReplyable {
				t.Errorf("IsReplyable() = %v, want %v", got, test.wantReplyable)
			}
			if got := test.act.IsValidCodeComment(); got != test.wantCode {
				t.Errorf("IsValidCodeComment() = %v, want %v", got, test.wantCode)
			}
		})
	}
}
