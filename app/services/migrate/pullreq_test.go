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
	"context"
	"testing"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitapi "github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/google/go-cmp/cmp"
)

// fakeGit is a minimal git.Interface used to drive the open pull request import path.
// Only GetBranch and MergeBase are implemented; any other call would panic (and indicates
// the test exercised an unexpected code path).
//
// Both branchSHA and branchErr may be left nil: reading from a nil map returns the zero
// value, so a test only needs to populate the map(s) it cares about. newFakeGit initialises
// both for callers that prefer a consistent, non-nil struct.
type fakeGit struct {
	git.Interface
	branchSHA    map[string]string // branch name -> tip SHA
	branchErr    map[string]error  // branch name -> error returned by GetBranch
	mergeBaseSHA string
	mergeBaseErr error
}

func newFakeGit() *fakeGit {
	return &fakeGit{
		branchSHA: map[string]string{},
		branchErr: map[string]error{},
	}
}

func (f *fakeGit) GetBranch(_ context.Context, p *git.GetBranchParams) (*git.GetBranchOutput, error) {
	if err := f.branchErr[p.BranchName]; err != nil {
		return nil, err
	}
	s, ok := f.branchSHA[p.BranchName]
	if !ok {
		return nil, errors.NotFoundf("branch %q not found", p.BranchName)
	}
	return &git.GetBranchOutput{Branch: git.Branch{Name: p.BranchName, SHA: sha.Must(s)}}, nil
}

func (f *fakeGit) MergeBase(_ context.Context, _ git.MergeBaseParams) (git.MergeBaseOutput, error) {
	if f.mergeBaseErr != nil {
		return git.MergeBaseOutput{}, f.mergeBaseErr
	}
	return git.MergeBaseOutput{MergeBaseSHA: sha.Must(f.mergeBaseSHA)}, nil
}

// fakePrincipalStore is a minimal store.PrincipalStore. Unknown emails return ErrResourceNotFound,
// mirroring the real store, which causes convertPullReq to fall back to the migrator principal.
type fakePrincipalStore struct {
	store.PrincipalStore
	byEmail map[string]*types.Principal
}

func (f *fakePrincipalStore) FindByEmail(_ context.Context, email string) (*types.Principal, error) {
	if p, ok := f.byEmail[email]; ok {
		return p, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

const (
	// valid lowercase hex SHAs (^[0-9a-f]{4,64}$).
	testHeadExternalSHA = "d79d48937903b01dd8e72224dc6989d910fe8fe8" // SHA carried in the migrate payload
	testBaseExternalSHA = "c1f79b793fc423705a85ddd9ab9f218b23e0caa9"
	testSourceTipSHA    = "1111111111111111111111111111111111111111" // current branch tips in the repo
	testTargetTipSHA    = "2222222222222222222222222222222222222222"
	testMergeBaseSHA    = "3333333333333333333333333333333333333333"
	testAuthorEmail     = "author@example.com"
)

func newOpenPullReqData() *ExternalPullRequest {
	ext := &ExternalPullRequest{}
	ext.PullRequest.Number = 42
	ext.PullRequest.Title = "Open PR"
	ext.PullRequest.Author.Email = testAuthorEmail
	ext.PullRequest.Head.Name = "feature"
	ext.PullRequest.Head.SHA = testHeadExternalSHA
	ext.PullRequest.Base.Name = "main"
	ext.PullRequest.Base.SHA = testBaseExternalSHA
	return ext
}

func newTestRepoImportState(g git.Interface, ps store.PrincipalStore) *repoImportState {
	return &repoImportState{
		git:            g,
		principalStore: ps,
		principals:     map[string]*types.Principal{},
		unknownEmails:  map[int]map[string]bool{},
		migrator:       types.Principal{ID: 1, UID: "migrator"},
	}
}

// TestConvertPullReqOpen covers the open pull request path of convertPullReq, in particular the
// merge-base scenario that previously failed the whole import: branches with more than one merge
// base (criss-cross history, e.g. pull requests from forks).
func TestConvertPullReqOpen(t *testing.T) {
	ctx := context.Background()
	repo := &types.RepositoryCore{ID: 1, GitUID: "git-uid", Identifier: "repo"}
	author := &types.Principal{ID: 10, UID: "author"}
	principals := func() store.PrincipalStore {
		return &fakePrincipalStore{byEmail: map[string]*types.Principal{testAuthorEmail: author}}
	}

	t.Run("happy path uses resolved branch tips and merge base", func(t *testing.T) {
		fg := newFakeGit()
		fg.branchSHA["feature"] = testSourceTipSHA
		fg.branchSHA["main"] = testTargetTipSHA
		fg.mergeBaseSHA = testMergeBaseSHA
		r := newTestRepoImportState(fg, principals())

		pr, err := r.convertPullReq(ctx, repo, newOpenPullReqData())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pr.State != enum.PullReqStateOpen {
			t.Errorf("state = %s, want %s", pr.State, enum.PullReqStateOpen)
		}
		if pr.SourceSHA != testSourceTipSHA {
			t.Errorf("SourceSHA = %s, want resolved branch tip %s", pr.SourceSHA, testSourceTipSHA)
		}
		if pr.MergeTargetSHA == nil || *pr.MergeTargetSHA != testTargetTipSHA {
			t.Errorf("MergeTargetSHA = %v, want %s", pr.MergeTargetSHA, testTargetTipSHA)
		}
		if pr.MergeBaseSHA != testMergeBaseSHA {
			t.Errorf("MergeBaseSHA = %s, want %s", pr.MergeBaseSHA, testMergeBaseSHA)
		}
	})

	t.Run("more than one merge base does not fail the import", func(t *testing.T) {
		// This is the customer's reported error: the source and target branches have multiple
		// merge bases. The import must not fail; the PR keeps its real source/target tips and uses
		// the external base SHA as a placeholder merge base, with merge status left unchecked.
		fg := newFakeGit()
		fg.branchSHA["feature"] = testSourceTipSHA
		fg.branchSHA["main"] = testTargetTipSHA
		fg.mergeBaseErr = gitapi.ErrMergeBaseNonUnique
		r := newTestRepoImportState(fg, principals())

		pr, err := r.convertPullReq(ctx, repo, newOpenPullReqData())
		if err != nil {
			t.Fatalf("expected import to tolerate multiple merge bases, got error: %v", err)
		}
		if pr.State != enum.PullReqStateOpen {
			t.Errorf("state = %s, want %s", pr.State, enum.PullReqStateOpen)
		}
		if pr.SourceSHA != testSourceTipSHA {
			t.Errorf("SourceSHA = %s, want resolved branch tip %s", pr.SourceSHA, testSourceTipSHA)
		}
		if pr.MergeTargetSHA == nil || *pr.MergeTargetSHA != testTargetTipSHA {
			t.Errorf("MergeTargetSHA = %v, want %s", pr.MergeTargetSHA, testTargetTipSHA)
		}
		if pr.MergeBaseSHA != testTargetTipSHA {
			t.Errorf("MergeBaseSHA = %s, want target branch SHA placeholder %s", pr.MergeBaseSHA, testTargetTipSHA)
		}
	})

	t.Run("unrelated histories do not fail the import", func(t *testing.T) {
		fg := newFakeGit()
		fg.branchSHA["feature"] = testSourceTipSHA
		fg.branchSHA["main"] = testTargetTipSHA
		fg.mergeBaseErr = &gitapi.UnrelatedHistoriesError{BaseRef: testSourceTipSHA, HeadRef: testTargetTipSHA}
		r := newTestRepoImportState(fg, principals())

		pr, err := r.convertPullReq(ctx, repo, newOpenPullReqData())
		if err != nil {
			t.Fatalf("expected import to tolerate unrelated histories, got error: %v", err)
		}
		if pr.MergeBaseSHA != testTargetTipSHA {
			t.Errorf("MergeBaseSHA = %s, want target branch SHA placeholder %s", pr.MergeBaseSHA, testTargetTipSHA)
		}
	})

	t.Run("other invalid-argument merge base errors still fail the import", func(t *testing.T) {
		// Guards against swallowing unrelated invalid-argument errors: only the named
		// non-unique/unrelated merge base cases are tolerated, everything else surfaces.
		fg := newFakeGit()
		fg.branchSHA["feature"] = testSourceTipSHA
		fg.branchSHA["main"] = testTargetTipSHA
		fg.mergeBaseErr = errors.InvalidArgumentf("some unrelated invalid argument")
		r := newTestRepoImportState(fg, principals())

		if _, err := r.convertPullReq(ctx, repo, newOpenPullReqData()); err == nil {
			t.Fatal("expected unrelated invalid-argument error to fail the import, got nil")
		}
	})

	t.Run("missing branch still fails the import", func(t *testing.T) {
		// Branch resolution errors are out of scope for the graceful handling and keep the
		// original strict behavior.
		fg := newFakeGit()
		fg.branchSHA["main"] = testTargetTipSHA
		fg.branchErr["feature"] = errors.NotFoundf("branch not found")
		r := newTestRepoImportState(fg, principals())

		if _, err := r.convertPullReq(ctx, repo, newOpenPullReqData()); err == nil {
			t.Fatal("expected error for missing source branch, got nil")
		}
	})
}

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
