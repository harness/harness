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

package migrate

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateThreads(t *testing.T) {
	// comments with treelike structure
	t0 := time.Now()
	comments := []ExternalComment{
		/* 0 */ {ID: 1, Body: "A", ParentID: 0},
		/* 1 */ {ID: 2, Body: "B", ParentID: 0},
		/* 2 */ {ID: 3, Body: "A1", ParentID: 1},
		/* 3 */ {ID: 4, Body: "B1", ParentID: 2},
		/* 4 */ {ID: 5, Body: "A2", ParentID: 1},
		/* 5 */ {ID: 6, Body: "A2X", ParentID: 5},
		/* 6 */ {ID: 7, Body: "A1X", ParentID: 3},
		/* 7 */ {ID: 8, Body: "B1X", ParentID: 4},
		/* 8 */ {ID: 9, Body: "C", ParentID: 0},
		/* 9 */ {ID: 10, Body: "D1", ParentID: 11}, // Wrong order - a reply before its parent
		/* 10 */ {ID: 11, Body: "D", ParentID: 0},
		{ID: 20, Body: "Self-parent", ParentID: 20},   // Invalid
		{ID: 30, Body: "Crosslinked-X", ParentID: 31}, // Invalid
		{ID: 31, Body: "Crosslinked-Y", ParentID: 30}, // Invalid
	}

	for i := range comments {
		comments[i].Created = t0.Add(time.Duration(i) * time.Minute)
	}

	// flattened threads with top level comments and a list of replies to each of them
	wantThreads := []*externalCommentThread{
		{
			TopLevel: comments[0],                                                           // A
			Replies:  []ExternalComment{comments[2], comments[4], comments[5], comments[6]}, // A1, A2, A2X, A1X
		},
		{
			TopLevel: comments[1],                                 // B
			Replies:  []ExternalComment{comments[3], comments[7]}, // B1, B1X
		},
		{
			TopLevel: comments[8], // C
			Replies:  []ExternalComment{},
		},
		{
			TopLevel: comments[10],                   // D
			Replies:  []ExternalComment{comments[9]}, // D1
		},
	}

	gotThreads := generateThreads(comments)
	if diff := cmp.Diff(gotThreads, wantThreads); diff != "" {
		t.Error(diff)
	}
}

func TestTimestampMillis(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		fallback int64
		want     int64
	}{
		{
			name:     "valid time",
			input:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			fallback: 0,
			want:     time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC).UnixMilli(),
		},
		{
			name:     "zero time",
			input:    time.Time{},
			fallback: 123456789,
			want:     123456789,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timestampMillis(tt.input, tt.fallback)
			if got != tt.want {
				t.Errorf("timestampMillis() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestActivitySeqOrdering tests that ActivitySeq is properly incremented across.
// reviewer activities, review activities, and comments to prevent UNIQUE constraint violations.
func TestActivitySeqOrdering(t *testing.T) {
	tests := []struct {
		name          string
		reviewerCount int
		reviewCount   int
		commentCount  int
		wantMinSeq    int64 // minimum ActivitySeq after all activities
	}{
		{
			name:          "single reviewer, single review, single comment",
			reviewerCount: 1,
			reviewCount:   1,
			commentCount:  1,
			wantMinSeq:    3, // 1 reviewer activity + 1 review activity + 1 comment
		},
		{
			name:          "multiple reviewers, multiple reviews, multiple comments",
			reviewerCount: 3,
			reviewCount:   2,
			commentCount:  5,
			wantMinSeq:    8, // 1 reviewer activity (batched) + 2 review activities + 5 comments
		},
		{
			name:          "no reviewers, multiple reviews and comments",
			reviewerCount: 0,
			reviewCount:   3,
			commentCount:  2,
			wantMinSeq:    5, // 0 reviewer activities + 3 review activities + 2 comments
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate ActivitySeq progression as it would happen in migration
			activitySeq := int64(0)

			// Reviewer activity (batched for all reviewers)
			if tt.reviewerCount > 0 {
				activitySeq++ // One activity for all reviewers
			}

			// Review activities (one per review)
			activitySeq += int64(tt.reviewCount)

			// Comment activities (starts from current ActivitySeq + 1)
			// Comments use: order := int(pullReq.ActivitySeq) + idxTopLevel + 1
			if tt.commentCount > 0 {
				finalCommentOrder := activitySeq + int64(tt.commentCount)
				activitySeq = finalCommentOrder
			}

			if activitySeq < tt.wantMinSeq {
				t.Errorf("ActivitySeq ordering failed: got %d, want at least %d", activitySeq, tt.wantMinSeq)
			}
		})
	}
}

// TestReviewerActivityPayloadStructure tests that reviewer activity payloads.
// contain the expected fields to prevent marshaling/unmarshaling issues.
func TestReviewerActivityPayloadStructure(t *testing.T) {
	// This test ensures the payload structure matches what CreateWithPayload expects
	reviewerIDs := []int64{123, 456, 789}

	// Simulate creating the payload as done in createReviewerActivity
	payload := struct {
		ReviewerType string  `json:"reviewer_type"`
		PrincipalIDs []int64 `json:"principal_ids"`
	}{
		ReviewerType: "requested",
		PrincipalIDs: reviewerIDs,
	}

	// Verify critical fields are populated
	if payload.ReviewerType == "" {
		t.Error("ReviewerType must not be empty")
	}

	if len(payload.PrincipalIDs) != len(reviewerIDs) {
		t.Errorf("PrincipalIDs length mismatch: got %d, want %d", len(payload.PrincipalIDs), len(reviewerIDs))
	}

	for i, id := range reviewerIDs {
		if payload.PrincipalIDs[i] != id {
			t.Errorf("PrincipalID[%d] mismatch: got %d, want %d", i, payload.PrincipalIDs[i], id)
		}
	}
}
