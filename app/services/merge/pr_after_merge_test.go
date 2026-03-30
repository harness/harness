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

package merge

import (
	"testing"
	"time"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestMutatePullReqAfterMerge(t *testing.T) {
	headSHA := sha.Must("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	baseSHA := sha.Must("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	mergeBaseSHA := sha.Must("cccccccccccccccccccccccccccccccccccccccc")
	mergeSHA := sha.Must("dddddddddddddddddddddddddddddddddddddddd")

	mergedBy := &types.PrincipalInfo{
		ID:          42,
		UID:         "user42",
		DisplayName: "Test User",
		Email:       "test@example.com",
	}

	mergeOutput := git.MergeOutput{
		HeadSHA:          headSHA,
		BaseSHA:          baseSHA,
		MergeBaseSHA:     mergeBaseSHA,
		MergeSHA:         mergeSHA,
		CommitCount:      3,
		ChangedFileCount: 5,
		Additions:        100,
		Deletions:        20,
	}

	tests := []struct {
		name          string
		initialPR     types.PullReq
		mergeMethod   enum.MergeMethod
		mergeOutput   git.MergeOutput
		mergedBy      *types.PrincipalInfo
		rulesBypassed bool
	}{
		{
			name: "merge-method-no-bypass",
			initialPR: types.PullReq{
				ID:          1,
				ActivitySeq: 5,
				State:       enum.PullReqStateOpen,
				SubState:    enum.PullReqSubStateAutoMerge,
			},
			mergeMethod:   enum.MergeMethodMerge,
			mergeOutput:   mergeOutput,
			mergedBy:      mergedBy,
			rulesBypassed: false,
		},
		{
			name: "squash-method-with-bypass",
			initialPR: types.PullReq{
				ID:          2,
				ActivitySeq: 0,
				State:       enum.PullReqStateOpen,
				SubState:    enum.PullReqSubStateNone,
			},
			mergeMethod:   enum.MergeMethodSquash,
			mergeOutput:   mergeOutput,
			mergedBy:      mergedBy,
			rulesBypassed: true,
		},
		{
			name: "rebase-method",
			initialPR: types.PullReq{
				ID:          3,
				ActivitySeq: 10,
				State:       enum.PullReqStateOpen,
			},
			mergeMethod:   enum.MergeMethodRebase,
			mergeOutput:   mergeOutput,
			mergedBy:      mergedBy,
			rulesBypassed: false,
		},
		{
			name: "fast-forward-method",
			initialPR: types.PullReq{
				ID:          4,
				ActivitySeq: 2,
				State:       enum.PullReqStateOpen,
			},
			mergeMethod:   enum.MergeMethodFastForward,
			mergeOutput:   mergeOutput,
			mergedBy:      mergedBy,
			rulesBypassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{}
			pr := tt.initialPR
			initialActivitySeq := pr.ActivitySeq

			before := time.Now().UnixMilli()
			seqMerge, seqBranchDeleted := svc.mutatePullReqAfterMerge(
				&pr,
				tt.mergeMethod,
				tt.mergeOutput,
				tt.mergedBy,
				tt.rulesBypassed,
			)
			after := time.Now().UnixMilli()

			resultPR := &pr

			// state and substate
			if resultPR.State != enum.PullReqStateMerged {
				t.Errorf("State: got %v, want %v", resultPR.State, enum.PullReqStateMerged)
			}
			if resultPR.SubState != enum.PullReqSubStateNone {
				t.Errorf("SubState: got %v, want %v", resultPR.SubState, enum.PullReqSubStateNone)
			}

			// merged timestamp
			if resultPR.Merged == nil {
				t.Fatal("Merged: expected non-nil timestamp")
			}
			if *resultPR.Merged < before || *resultPR.Merged > after {
				t.Errorf("Merged timestamp %d not in expected range [%d, %d]", *resultPR.Merged, before, after)
			}

			// merged by
			if resultPR.MergedBy == nil {
				t.Fatal("MergedBy: expected non-nil")
			}
			if *resultPR.MergedBy != tt.mergedBy.ID {
				t.Errorf("MergedBy: got %d, want %d", *resultPR.MergedBy, tt.mergedBy.ID)
			}

			// merge method
			if resultPR.MergeMethod == nil {
				t.Fatal("MergeMethod: expected non-nil")
			}
			if *resultPR.MergeMethod != tt.mergeMethod {
				t.Errorf("MergeMethod: got %v, want %v", *resultPR.MergeMethod, tt.mergeMethod)
			}

			// SHAs
			if resultPR.SourceSHA != tt.mergeOutput.HeadSHA.String() {
				t.Errorf("SourceSHA: got %q, want %q", resultPR.SourceSHA, tt.mergeOutput.HeadSHA.String())
			}
			if resultPR.MergeTargetSHA == nil {
				t.Fatal("MergeTargetSHA: expected non-nil")
			}
			if *resultPR.MergeTargetSHA != tt.mergeOutput.BaseSHA.String() {
				t.Errorf("MergeTargetSHA: got %q, want %q", *resultPR.MergeTargetSHA, tt.mergeOutput.BaseSHA.String())
			}
			if resultPR.MergeBaseSHA != tt.mergeOutput.MergeBaseSHA.String() {
				t.Errorf("MergeBaseSHA: got %q, want %q", resultPR.MergeBaseSHA, tt.mergeOutput.MergeBaseSHA.String())
			}
			if resultPR.MergeSHA == nil {
				t.Fatal("MergeSHA: expected non-nil")
			}
			if *resultPR.MergeSHA != tt.mergeOutput.MergeSHA.String() {
				t.Errorf("MergeSHA: got %q, want %q", *resultPR.MergeSHA, tt.mergeOutput.MergeSHA.String())
			}

			// merge check statuses (set by MarkAsMerged)
			if resultPR.MergeCheckStatus != enum.MergeCheckStatusMergeable {
				t.Errorf("MergeCheckStatus: got %v, want %v", resultPR.MergeCheckStatus, enum.MergeCheckStatusMergeable)
			}
			if resultPR.RebaseCheckStatus != enum.MergeCheckStatusMergeable {
				t.Errorf("RebaseCheckStatus: got %v, want %v", resultPR.RebaseCheckStatus, enum.MergeCheckStatusMergeable)
			}

			// rules bypassed
			if resultPR.MergeViolationsBypassed == nil {
				t.Fatal("MergeViolationsBypassed: expected non-nil")
			}
			if *resultPR.MergeViolationsBypassed != tt.rulesBypassed {
				t.Errorf("MergeViolationsBypassed: got %v, want %v", *resultPR.MergeViolationsBypassed, tt.rulesBypassed)
			}

			// diff stats
			o := tt.mergeOutput
			if resultPR.Stats.Commits == nil || *resultPR.Stats.Commits != int64(o.CommitCount) {
				t.Errorf("Stats.Commits: got %v, want %d", resultPR.Stats.Commits, o.CommitCount)
			}
			if resultPR.Stats.FilesChanged == nil || *resultPR.Stats.FilesChanged != int64(o.ChangedFileCount) {
				t.Errorf("Stats.FilesChanged: got %v, want %d", resultPR.Stats.FilesChanged, o.ChangedFileCount)
			}
			if resultPR.Stats.Additions == nil || *resultPR.Stats.Additions != int64(o.Additions) {
				t.Errorf("Stats.Additions: got %v, want %d", resultPR.Stats.Additions, o.Additions)
			}
			if resultPR.Stats.Deletions == nil || *resultPR.Stats.Deletions != int64(o.Deletions) {
				t.Errorf("Stats.Deletions: got %v, want %d", resultPR.Stats.Deletions, o.Deletions)
			}

			// activity sequence
			expectedSeqMerge := initialActivitySeq + 1
			expectedSeqBranchDeleted := initialActivitySeq + 2
			if seqMerge != expectedSeqMerge {
				t.Errorf("seqMerge: got %d, want %d", seqMerge, expectedSeqMerge)
			}
			if seqBranchDeleted != expectedSeqBranchDeleted {
				t.Errorf("seqBranchDeleted: got %d, want %d", seqBranchDeleted, expectedSeqBranchDeleted)
			}
			if resultPR.ActivitySeq != expectedSeqBranchDeleted {
				t.Errorf("ActivitySeq: got %d, want %d", resultPR.ActivitySeq, expectedSeqBranchDeleted)
			}
		})
	}
}
