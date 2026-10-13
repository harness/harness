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

package handlers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/linkedsync"
	"github.com/harness/gitness/app/services/linkedsync/handlers"
	gitpkg "github.com/harness/gitness/git"
	mockgit "github.com/harness/gitness/mocks/git"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/mock"
)

// testDefaultBranchRef is the fully-qualified ref matching the default branch
// configured on the stub repos below.
const testDefaultBranchRef = "refs/heads/main"

// indexerStub is a minimal keywordsearch.Indexer recording which repos were
// (re-)indexed, so tests can assert indexing happened only for default-branch
// pushes.
type indexerStub struct {
	indexed []int64
	err     error
}

func (s *indexerStub) Index(_ context.Context, repo *types.Repository) error {
	s.indexed = append(s.indexed, repo.ID)
	return s.err
}

// configurePushGitMock stubs only what RunSyncRefs needs for verbatim (no
// rename) entries: a single SyncRefs call.
func configurePushGitMock(m *mockgit.Interface) {
	m.On("SyncRefs", mock.Anything, mock.Anything).Return(&gitpkg.SyncRefsOutput{}, nil)
}

func newPushHandler(
	t *testing.T,
	repoStore *repoStoreStub,
	indexer *indexerStub,
	gitMock *mockgit.Interface,
) *handlers.PushHandler {
	t.Helper()
	repo := &types.RepositoryCore{ID: testRepoID, GitUID: "git-uid-test"}
	return handlers.NewPushHandler(
		repoStore, indexer, gitMock, testRepoFinder(repo), testURLProvider{}, testConnectorService{},
	)
}

func callPushHandle(t *testing.T, h *handlers.PushHandler, ev *linkedsync.Event, repo *types.LinkedRepo) error {
	t.Helper()
	pl, ok := ev.Payload.(linkedsync.PushPayload)
	if !ok {
		t.Fatalf("test event payload is not PushPayload: %T", ev.Payload)
	}
	return h.Handle(context.Background(), ev, pl, repo)
}

func pushEvent(mutators ...func(*linkedsync.PushPayload)) *linkedsync.Event {
	p := linkedsync.PushPayload{
		Ref:   testDefaultBranchRef,
		After: stubMergeBaseSHA,
		Repository: linkedsync.Repository{
			ProviderID: testRepoProviderID,
		},
	}
	for _, m := range mutators {
		m(&p)
	}
	return &linkedsync.Event{
		Provider:   linkedsync.ProviderGitHub,
		AccountID:  "acct-1",
		DeliveryID: "deliv-push-1",
		Payload:    p,
	}
}

// TestPushHandler_DefaultBranchPush_SyncsAndIndexes: a push to the repo's
// configured default branch mirrors the ref and re-indexes for search, same
// as linked_sync.go's controller does for on-demand syncs.
func TestPushHandler_DefaultBranchPush_SyncsAndIndexes(t *testing.T) {
	repoStore := &repoStoreStub{repo: &types.Repository{ID: testRepoID, DefaultBranch: "main"}}
	indexer := &indexerStub{}
	gitMock := &mockgit.Interface{}
	configurePushGitMock(gitMock)
	h := newPushHandler(t, repoStore, indexer, gitMock)

	ev := pushEvent(func(p *linkedsync.PushPayload) { p.Ref = testDefaultBranchRef })
	if err := callPushHandle(t, h, ev, testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	gitMock.AssertCalled(t, "SyncRefs", mock.Anything, mock.MatchedBy(func(p *gitpkg.SyncRefsParams) bool {
		return len(p.Refs) == 1 && p.Refs[0] == "refs/heads/main"
	}))
	if len(indexer.indexed) != 1 || indexer.indexed[0] != testRepoID {
		t.Errorf("expected repo %d to be indexed once, got %v", testRepoID, indexer.indexed)
	}
}

// TestPushHandler_NonDefaultBranchPush_SyncsWithoutIndexing: pushes to any
// other branch mirror the ref but must not trigger a re-index.
func TestPushHandler_NonDefaultBranchPush_SyncsWithoutIndexing(t *testing.T) {
	repoStore := &repoStoreStub{repo: &types.Repository{ID: testRepoID, DefaultBranch: "main"}}
	indexer := &indexerStub{}
	gitMock := &mockgit.Interface{}
	configurePushGitMock(gitMock)
	h := newPushHandler(t, repoStore, indexer, gitMock)

	ev := pushEvent(func(p *linkedsync.PushPayload) { p.Ref = "refs/heads/feature-x" })
	if err := callPushHandle(t, h, ev, testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	gitMock.AssertCalled(t, "SyncRefs", mock.Anything, mock.MatchedBy(func(p *gitpkg.SyncRefsParams) bool {
		return len(p.Refs) == 1 && p.Refs[0] == "refs/heads/feature-x"
	}))
	if len(indexer.indexed) != 0 {
		t.Errorf("expected no indexing for non-default branch push, got %v", indexer.indexed)
	}
}

// TestPushHandler_TagPush_SyncsRef: tags mirror the same as branches, per the
// ticket's scope ("Branches and tags"), and never trigger re-indexing.
func TestPushHandler_TagPush_SyncsRef(t *testing.T) {
	repoStore := &repoStoreStub{repo: &types.Repository{ID: testRepoID, DefaultBranch: "main"}}
	indexer := &indexerStub{}
	gitMock := &mockgit.Interface{}
	configurePushGitMock(gitMock)
	h := newPushHandler(t, repoStore, indexer, gitMock)

	ev := pushEvent(func(p *linkedsync.PushPayload) { p.Ref = "refs/tags/v1.0.0" })
	if err := callPushHandle(t, h, ev, testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	gitMock.AssertCalled(t, "SyncRefs", mock.Anything, mock.MatchedBy(func(p *gitpkg.SyncRefsParams) bool {
		return len(p.Refs) == 1 && p.Refs[0] == "refs/tags/v1.0.0"
	}))
	if len(indexer.indexed) != 0 {
		t.Errorf("expected no indexing for tag push, got %v", indexer.indexed)
	}
}

// TestPushHandler_UnsupportedRef_AckSkips: refs outside heads/tags (e.g. a
// provider echoing a PR ref) are dropped without ever calling SyncRefs, so a
// malformed/unexpected ref can't turn into a hard git error. The bare-prefix
// cases matter just as much: "refs/heads/" names nothing, so SyncRefs would
// hard-fail and poison-pill the consumer into endless retry — the same reason
// PullRequestHandler rejects empty head/base refs up front.
func TestPushHandler_UnsupportedRef_AckSkips(t *testing.T) {
	cases := []struct {
		name string
		ref  string
	}{
		{"pr ref", "refs/pull/7/head"},
		{"bare branch prefix", "refs/heads/"},
		{"bare tag prefix", "refs/tags/"},
		{"empty ref", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoStore := &repoStoreStub{repo: &types.Repository{ID: testRepoID, DefaultBranch: "main"}}
			indexer := &indexerStub{}
			gitMock := &mockgit.Interface{}
			configurePushGitMock(gitMock)
			h := newPushHandler(t, repoStore, indexer, gitMock)

			ev := pushEvent(func(p *linkedsync.PushPayload) { p.Ref = tc.ref })
			if err := callPushHandle(t, h, ev, testLinkedRepo()); err != nil {
				t.Fatalf("Handle: %v", err)
			}

			gitMock.AssertNotCalled(t, "SyncRefs", mock.Anything, mock.Anything)
			if len(indexer.indexed) != 0 {
				t.Errorf("expected no indexing for unsupported ref, got %v", indexer.indexed)
			}
		})
	}
}

// TestPushHandler_DeletedRef_AckSkips: a branch/tag deletion arrives as a
// push with an empty or all-zero After SHA. SyncRefs would hard-fail trying
// to fetch a ref the remote no longer has, so this must ack-skip instead of
// retrying forever. Mirroring the deletion itself is out of scope.
func TestPushHandler_DeletedRef_AckSkips(t *testing.T) {
	cases := []struct {
		name  string
		after string
	}{
		{"empty after", ""},
		{"all-zero after", "0000000000000000000000000000000000000000"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoStore := &repoStoreStub{repo: &types.Repository{ID: testRepoID, DefaultBranch: "main"}}
			indexer := &indexerStub{}
			gitMock := &mockgit.Interface{}
			configurePushGitMock(gitMock)
			h := newPushHandler(t, repoStore, indexer, gitMock)

			ev := pushEvent(func(p *linkedsync.PushPayload) {
				p.Ref = "refs/heads/deleted-branch"
				p.After = tc.after
			})
			if err := callPushHandle(t, h, ev, testLinkedRepo()); err != nil {
				t.Fatalf("Handle: %v", err)
			}

			gitMock.AssertNotCalled(t, "SyncRefs", mock.Anything, mock.Anything)
			if len(indexer.indexed) != 0 {
				t.Errorf("expected no indexing for deleted ref, got %v", indexer.indexed)
			}
		})
	}
}

// TestPushHandler_MalformedAfterSHA_AckSkips: a producer bug or unexpected
// payload shape must not poison-pill retry.
func TestPushHandler_MalformedAfterSHA_AckSkips(t *testing.T) {
	repoStore := &repoStoreStub{repo: &types.Repository{ID: testRepoID, DefaultBranch: "main"}}
	indexer := &indexerStub{}
	gitMock := &mockgit.Interface{}
	configurePushGitMock(gitMock)
	h := newPushHandler(t, repoStore, indexer, gitMock)

	ev := pushEvent(func(p *linkedsync.PushPayload) { p.After = "not-a-valid-sha!!" })
	if err := callPushHandle(t, h, ev, testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	gitMock.AssertNotCalled(t, "SyncRefs", mock.Anything, mock.Anything)
	if len(indexer.indexed) != 0 {
		t.Errorf("expected no indexing for malformed after-sha, got %v", indexer.indexed)
	}
}

// TestPushHandler_SyncRefsError_Propagates: a real git-sync failure must
// surface as an error so the broker retries, unlike the ack-skip cases above.
func TestPushHandler_SyncRefsError_Propagates(t *testing.T) {
	repoStore := &repoStoreStub{repo: &types.Repository{ID: testRepoID, DefaultBranch: "main"}}
	indexer := &indexerStub{}
	gitMock := &mockgit.Interface{}
	gitMock.On("SyncRefs", mock.Anything, mock.Anything).Return(&gitpkg.SyncRefsOutput{}, errors.New("boom"))
	h := newPushHandler(t, repoStore, indexer, gitMock)

	ev := pushEvent()
	err := callPushHandle(t, h, ev, testLinkedRepo())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestPushHandler_IndexerError_DoesNotFailHandle: re-indexing is best-effort,
// mirroring linked_sync.go's controller — a failed index must not undo or
// fail the already-successful ref sync.
func TestPushHandler_IndexerError_DoesNotFailHandle(t *testing.T) {
	repoStore := &repoStoreStub{repo: &types.Repository{ID: testRepoID, DefaultBranch: "main"}}
	indexer := &indexerStub{err: errors.New("index boom")}
	gitMock := &mockgit.Interface{}
	configurePushGitMock(gitMock)
	h := newPushHandler(t, repoStore, indexer, gitMock)

	ev := pushEvent(func(p *linkedsync.PushPayload) { p.Ref = testDefaultBranchRef })
	if err := callPushHandle(t, h, ev, testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
}
