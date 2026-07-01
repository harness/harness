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
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ─── stubs ────────────────────────────────────────────────────────────────

// noopTx executes txFn directly without any transaction semantics.
type noopTx struct{}

func (noopTx) WithTx(ctx context.Context, txFn func(context.Context) error, _ ...any) error {
	return txFn(ctx)
}

// stubRepoStore records UpdateOptLock invocations so tests can assert deltas.
type stubRepoStore struct {
	store.RepoStore
	repo            *types.Repository
	findErr         error
	updateOptLockFn func(*types.Repository) error
	captured        *types.Repository
}

func (s *stubRepoStore) Find(_ context.Context, _ int64) (*types.Repository, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	cp := *s.repo
	return &cp, nil
}

func (s *stubRepoStore) UpdateOptLock(
	_ context.Context, repo *types.Repository, mutate func(*types.Repository) error,
) (*types.Repository, error) {
	cp := *repo
	if err := mutate(&cp); err != nil {
		return nil, err
	}
	s.captured = &cp
	if s.updateOptLockFn != nil {
		if err := s.updateOptLockFn(&cp); err != nil {
			return nil, err
		}
	}
	return &cp, nil
}

// stubPullReqStore records Update calls and serves Find responses.
type stubPullReqStore struct {
	store.PullReqStore
	findFn   func(int64) (*types.PullReq, error)
	updateFn func(*types.PullReq) error
	updated  []*types.PullReq
}

func (s *stubPullReqStore) Find(_ context.Context, id int64) (*types.PullReq, error) {
	if s.findFn != nil {
		return s.findFn(id)
	}
	return nil, errors.New("Find not stubbed")
}

func (s *stubPullReqStore) Update(_ context.Context, pr *types.PullReq) error {
	cp := *pr
	s.updated = append(s.updated, &cp)
	if s.updateFn != nil {
		return s.updateFn(pr)
	}
	return nil
}

// stubActivityStore swallows activity inserts.
type stubActivityStore struct {
	store.PullReqActivityStore
}

func (stubActivityStore) CreateWithPayload(
	_ context.Context, _ *types.PullReq, _ int64,
	_ types.PullReqActivityPayload, _ *types.PullReqActivityMetadata,
) (*types.PullReqActivity, error) {
	return &types.PullReqActivity{}, nil
}

// stubAutoMergeStore returns false (no rows deleted) by default.
type stubAutoMergeStore struct {
	store.AutoMergeStore
}

func (stubAutoMergeStore) Delete(_ context.Context, _ int64) (bool, error) {
	return false, nil
}

// stubGit records UpdateRefs calls.
type stubGit struct {
	git.Interface
	updateRefsErr  error
	updateRefsArgs []git.UpdateRefsParams
}

func (s *stubGit) UpdateRefs(_ context.Context, p git.UpdateRefsParams) error {
	s.updateRefsArgs = append(s.updateRefsArgs, p)
	return s.updateRefsErr
}

// ─── helpers ──────────────────────────────────────────────────────────────

func buildTestService(
	repoStore *stubRepoStore, prStore *stubPullReqStore, gitClient *stubGit,
) *Service {
	return &Service{
		git:            gitClient,
		tx:             noopTx{},
		repoStore:      repoStore,
		pullreqStore:   prStore,
		activityStore:  stubActivityStore{},
		autoMergeStore: stubAutoMergeStore{},
	}
}

const testSourceSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func basePR() *types.PullReq {
	return &types.PullReq{
		ID:           7,
		Number:       42,
		State:        enum.PullReqStateOpen,
		SubState:     enum.PullReqSubStateNone,
		TargetRepoID: 100,
		SourceSHA:    testSourceSHA,
		ActivitySeq:  5,
	}
}

func baseMergeOutput() git.MergeOutput {
	return git.MergeOutput{
		HeadSHA:      sha.Must("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		BaseSHA:      sha.Must("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		MergeBaseSHA: sha.Must("cccccccccccccccccccccccccccccccccccccccc"),
		MergeSHA:     sha.Must("dddddddddddddddddddddddddddddddddddddddd"),
	}
}

// ─── tests ────────────────────────────────────────────────────────────────

// TestTxRefAndDatabaseUpdate_HappyPath: on success the repo counters move
// open-- / merged++ and git refs are updated.
func TestTxRefAndDatabaseUpdate_HappyPath(t *testing.T) {
	repoStore := &stubRepoStore{repo: &types.Repository{
		ID: 100, NumOpenPulls: 3, NumMergedPulls: 1,
	}}
	prStore := &stubPullReqStore{}
	gitClient := &stubGit{}
	svc := buildTestService(repoStore, prStore, gitClient)

	pr := basePR()
	mergeOutput := baseMergeOutput()
	refUpdates := []git.RefUpdate{
		{Name: "refs/pull/42/merge", Old: sha.None, New: sha.None},
	}

	_, _, err := svc.TxRefAndDatabaseUpdate(
		context.Background(),
		git.WriteParams{RepoUID: "test-repo"},
		sha.Must("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		refUpdates,
		pr,
		enum.MergeMethodMerge,
		mergeOutput,
		&types.PrincipalInfo{ID: 42},
		false,
		"",
	)
	if err != nil {
		t.Fatalf("TxRefAndDatabaseUpdate returned err: %v", err)
	}
	if repoStore.captured == nil {
		t.Fatal("expected repo counter update, got none")
	}
	if got, want := repoStore.captured.NumOpenPulls, 2; got != want {
		t.Errorf("NumOpenPulls: got %d, want %d", got, want)
	}
	if got, want := repoStore.captured.NumMergedPulls, 2; got != want {
		t.Errorf("NumMergedPulls: got %d, want %d", got, want)
	}
	if n := len(gitClient.updateRefsArgs); n != 1 {
		t.Fatalf("expected 1 git.UpdateRefs call, got %d", n)
	}
	// refUpdates[0].New had sha.None — should be substituted with MergeSHA.
	gotRefs := gitClient.updateRefsArgs[0].Refs
	if len(gotRefs) != 1 || gotRefs[0].New != mergeOutput.MergeSHA {
		t.Errorf("expected New=mergeSHA after substitution, got %+v", gotRefs)
	}
}

// TestTxRefAndDatabaseUpdate_SourceSHAConflict: a concurrent push changed the
// PR's source SHA → return Conflict, do NOT bump counters, do NOT update refs.
func TestTxRefAndDatabaseUpdate_SourceSHAConflict(t *testing.T) {
	repoStore := &stubRepoStore{repo: &types.Repository{
		ID: 100, NumOpenPulls: 3,
	}}
	prStore := &stubPullReqStore{}
	gitClient := &stubGit{}
	svc := buildTestService(repoStore, prStore, gitClient)

	pr := basePR()
	pr.SourceSHA = "different-from-expected"

	_, _, err := svc.TxRefAndDatabaseUpdate(
		context.Background(),
		git.WriteParams{RepoUID: "test-repo"},
		sha.Must("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), // expected sourceSHA
		nil,
		pr,
		enum.MergeMethodMerge,
		baseMergeOutput(),
		&types.PrincipalInfo{ID: 42},
		false,
		"",
	)
	if err == nil {
		t.Fatal("expected error on source SHA mismatch, got nil")
	}
	var ue *usererror.Error
	if !errors.As(err, &ue) || ue.Status != 409 {
		t.Errorf("expected usererror with 409 status, got: %v", err)
	}
	if repoStore.captured != nil {
		t.Errorf("counters must not be updated when source SHA conflicts, got: %+v", repoStore.captured)
	}
	if n := len(gitClient.updateRefsArgs); n != 0 {
		t.Errorf("git refs must not be updated when source SHA conflicts, got %d calls", n)
	}
}

// TestTxRefAndDatabaseUpdate_CounterUpdateFails: counter UpdateOptLock failing
// must not fail the outer call — the PR is already merged.
func TestTxRefAndDatabaseUpdate_CounterUpdateFails(t *testing.T) {
	repoStore := &stubRepoStore{
		repo:            &types.Repository{ID: 100, NumOpenPulls: 3},
		updateOptLockFn: func(*types.Repository) error { return errors.New("db gone") },
	}
	prStore := &stubPullReqStore{}
	gitClient := &stubGit{}
	svc := buildTestService(repoStore, prStore, gitClient)

	pr := basePR()

	_, _, err := svc.TxRefAndDatabaseUpdate(
		context.Background(),
		git.WriteParams{RepoUID: "test-repo"},
		sha.Must("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		nil,
		pr,
		enum.MergeMethodMerge,
		baseMergeOutput(),
		&types.PrincipalInfo{ID: 42},
		false,
		"",
	)
	if err != nil {
		t.Fatalf("counter failure must not surface as error, got: %v", err)
	}
	if n := len(gitClient.updateRefsArgs); n != 1 {
		t.Errorf("expected 1 git.UpdateRefs call before counter update, got %d", n)
	}
}

// TestTxRefAndDatabaseUpdate_RepoFindFails: failure to load the repo upfront
// is a hard error and must skip everything else.
func TestTxRefAndDatabaseUpdate_RepoFindFails(t *testing.T) {
	repoStore := &stubRepoStore{findErr: gitness_store.ErrResourceNotFound}
	prStore := &stubPullReqStore{}
	gitClient := &stubGit{}
	svc := buildTestService(repoStore, prStore, gitClient)

	_, _, err := svc.TxRefAndDatabaseUpdate(
		context.Background(),
		git.WriteParams{RepoUID: "test-repo"},
		sha.Must("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		nil,
		basePR(),
		enum.MergeMethodMerge,
		baseMergeOutput(),
		&types.PrincipalInfo{ID: 42},
		false,
		"",
	)
	if err == nil {
		t.Fatal("expected error when repo find fails, got nil")
	}
	if n := len(gitClient.updateRefsArgs); n != 0 {
		t.Errorf("git refs must not be updated when repo find fails, got %d calls", n)
	}
}
