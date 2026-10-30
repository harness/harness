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

package repo

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
)

type sizeUpdate struct {
	repoID     int64
	sizeKiB    int64
	lfsSizeKiB int64
	measuredAt int64
	inTx       bool
}

type recordingListener struct {
	calls      int
	measuredAt int64

	mu      sync.Mutex
	updates []sizeUpdate
	err     error
}

func (r *recordingListener) AfterSizeCalculation(_ context.Context, measuredAt int64) {
	r.calls++
	r.measuredAt = measuredAt
}

func (r *recordingListener) AfterSizeUpdate(ctx context.Context, repoID, measuredAt int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.updates = append(r.updates, sizeUpdate{
		repoID:     repoID,
		measuredAt: measuredAt,
		inTx:       inTx(ctx),
	})
	return r.err
}

func (r *recordingListener) recorded() []sizeUpdate {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]sizeUpdate(nil), r.updates...)
}

// txMarker is the key directTransactor marks its context with, standing in for the real
// transactor's accessor so a test can tell whether a call happened inside a transaction.
type txMarkerKey struct{}

func inTx(ctx context.Context) bool {
	return ctx.Value(txMarkerKey{}) != nil
}

// directTransactor runs txFn inline, marking the context so what it wraps is observable. There is
// no database here, so a rollback is not: what these tests check is that the size write and the
// notification are inside the same txFn, which is what makes them one transaction in production.
type directTransactor struct{}

func (directTransactor) WithTx(ctx context.Context, txFn func(ctx context.Context) error, _ ...any) error {
	return txFn(context.WithValue(ctx, txMarkerKey{}, struct{}{}))
}

// fakeRepoStoreFailing satisfies store.RepoStore via a nil embedded field, overriding only
// ListSizeInfos. The rest panic if ever called.
type fakeRepoStoreFailing struct {
	store.RepoStore
}

func (*fakeRepoStoreFailing) ListSizeInfos(context.Context) ([]*types.RepositorySizeInfo, error) {
	return nil, errors.New("boom")
}

// fakeRepoStore reports the given repos as due for measurement and records what was written for
// them, noting whether the write saw a transaction.
type fakeRepoStore struct {
	store.RepoStore

	sizeInfos []*types.RepositorySizeInfo

	mu      sync.Mutex
	updates []sizeUpdate
}

func (f *fakeRepoStore) ListSizeInfos(context.Context) ([]*types.RepositorySizeInfo, error) {
	return f.sizeInfos, nil
}

func (f *fakeRepoStore) UpdateSize(ctx context.Context, id, sizeInKiB, sizeLFSInKiB int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.updates = append(f.updates, sizeUpdate{
		repoID:     id,
		sizeKiB:    sizeInKiB,
		lfsSizeKiB: sizeLFSInKiB,
		inTx:       inTx(ctx),
	})
	return nil
}

func (f *fakeRepoStore) recorded() []sizeUpdate {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sizeUpdate(nil), f.updates...)
}

// fakeGit reports a fixed git size for every repo.
type fakeGit struct {
	git.Interface

	sizeKiB int64
}

func (f *fakeGit) GetRepositorySize(
	context.Context,
	*git.GetRepositorySizeParams,
) (*git.GetRepositorySizeOutput, error) {
	return &git.GetRepositorySizeOutput{Size: f.sizeKiB}, nil
}

// fakeLFSStore reports a fixed LFS size for every repo.
type fakeLFSStore struct {
	store.LFSObjectStore

	sizeKiB int64
}

func (f *fakeLFSStore) GetSizeInKBByRepoID(context.Context, int64) (int64, error) {
	return f.sizeKiB, nil
}

// fakeSpaceStoreFailing makes sendMetric's own failure path harmless, so these tests exercise only
// the listener.
type fakeSpaceStoreFailing struct {
	store.SpaceStore
}

func (*fakeSpaceStoreFailing) GetRootSpacesSize(context.Context) ([]types.SpaceStorage, error) {
	return nil, errors.New("boom")
}

// A run that fails before it updates a single size must still notify: the listener reads the sizes
// that are in the database, which this run did not invalidate.
func TestSizeCalculator_NotifiesListenerOnFailedRun(t *testing.T) {
	listener := &recordingListener{}
	calc := &SizeCalculator{
		enabled:    true,
		repoStore:  &fakeRepoStoreFailing{},
		spaceStore: &fakeSpaceStoreFailing{},
		listener:   listener,
	}

	if _, err := calc.Handle(context.Background(), "", nil); err == nil {
		t.Fatal("Handle succeeded, want the ListSizeInfos error to propagate")
	}
	if listener.calls != 1 {
		t.Errorf("listener called %d times, want 1", listener.calls)
	}
	if listener.measuredAt == 0 {
		t.Error("listener called with measuredAt 0, want the run's measurement time")
	}
}

// newTestCalculator builds a calculator over fakes that always measure sizeKiB/lfsKiB.
func newTestCalculator(
	repoStore *fakeRepoStore,
	listener Listener,
	sizeKiB, lfsKiB int64,
) *SizeCalculator {
	return &SizeCalculator{
		enabled:    true,
		numWorkers: 1,
		git:        &fakeGit{sizeKiB: sizeKiB},
		repoStore:  repoStore,
		spaceStore: &fakeSpaceStoreFailing{},
		lfsStore:   &fakeLFSStore{sizeKiB: lfsKiB},
		tx:         directTransactor{},
		listener:   listener,
	}
}

// The size write and the notification have to be one transaction, or a listener that reports sizes
// onward can miss a change that is already committed -- which for billing means revenue the next
// run has no reason to look at again.
func TestSizeCalculator_NotifiesListenerInsideTheSizeTransaction(t *testing.T) {
	repoStore := &fakeRepoStore{
		sizeInfos: []*types.RepositorySizeInfo{{ID: 42, GitUID: "git-uid", Size: 10, LFSSize: 20}},
	}
	listener := &recordingListener{}
	calc := newTestCalculator(repoStore, listener, 100, 200)

	if _, err := calc.Handle(context.Background(), "", nil); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	writes := repoStore.recorded()
	if len(writes) != 1 || !writes[0].inTx {
		t.Fatalf("size writes = %+v, want one write inside a transaction", writes)
	}
	if writes[0].sizeKiB != 100 || writes[0].lfsSizeKiB != 200 {
		t.Errorf("wrote %+v, want the newly measured sizes", writes[0])
	}

	notified := listener.recorded()
	if len(notified) != 1 {
		t.Fatalf("listener notified %d times, want 1", len(notified))
	}
	got := notified[0]
	if got.repoID != 42 {
		t.Errorf("listener notified for repo %d, want 42", got.repoID)
	}
	if !got.inTx {
		t.Error("listener was notified outside the transaction that wrote the size")
	}
	if got.measuredAt != listener.measuredAt {
		t.Errorf("listener got measuredAt %d per repo but %d for the run; a run must report one instant",
			got.measuredAt, listener.measuredAt)
	}
}

// A listener that fails costs its own repo only: the run keeps measuring the rest, and the failed
// one is picked up again next run.
func TestSizeCalculator_ListenerFailureDoesNotStopTheRun(t *testing.T) {
	repoStore := &fakeRepoStore{
		sizeInfos: []*types.RepositorySizeInfo{
			{ID: 1, GitUID: "git-uid-1"},
			{ID: 2, GitUID: "git-uid-2"},
		},
	}
	listener := &recordingListener{err: errors.New("boom")}
	calc := newTestCalculator(repoStore, listener, 100, 200)

	if _, err := calc.Handle(context.Background(), "", nil); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if notified := listener.recorded(); len(notified) != 2 {
		t.Errorf("listener notified %d times, want 2: a failure must not abandon the other repos",
			len(notified))
	}
}

// A repo nothing was pushed to measures the same size as last time, and is skipped without a write.
// Per-repo notification therefore does not cover it -- AfterSizeCalculation is what does.
func TestSizeCalculator_UnchangedSizeNotifiesNothing(t *testing.T) {
	repoStore := &fakeRepoStore{
		sizeInfos: []*types.RepositorySizeInfo{{ID: 7, GitUID: "git-uid", Size: 100, LFSSize: 200}},
	}
	listener := &recordingListener{}
	calc := newTestCalculator(repoStore, listener, 100, 200)

	if _, err := calc.Handle(context.Background(), "", nil); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if writes := repoStore.recorded(); len(writes) != 0 {
		t.Errorf("size writes = %+v, want none for an unchanged repo", writes)
	}
	if notified := listener.recorded(); len(notified) != 0 {
		t.Errorf("listener notified %d times, want 0 for an unchanged repo", len(notified))
	}
}

// A disabled calculator did not measure anything, so there is nothing to notify about.
func TestSizeCalculator_DisabledDoesNotNotify(t *testing.T) {
	listener := &recordingListener{}
	calc := &SizeCalculator{
		enabled:    false,
		spaceStore: &fakeSpaceStoreFailing{},
		listener:   listener,
	}

	if _, err := calc.Handle(context.Background(), "", nil); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if listener.calls != 0 {
		t.Errorf("listener called %d times, want 0", listener.calls)
	}
}
