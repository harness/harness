// Copyright 2026 Harness, Inc.
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
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	checkevents "github.com/harness/gitness/app/events/check"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/merge"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/lock"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

const (
	svbShaA = "1111111111111111111111111111111111111111"
	svbShaB = "2222222222222222222222222222222222222222"
	svbShaC = "3333333333333333333333333333333333333333"
)

func svbSHA(v string) sha.SHA { return sha.Must(v) }

type svbTx struct{}

func (svbTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

// svbErrs fails the named store method when present.
type svbErrs map[string]error

type svbMQStore struct {
	store.MergeQueueStore

	errs    svbErrs
	byID    map[int64]*types.MergeQueue
	created []*types.MergeQueue
	updated []*types.MergeQueue
	deleted []int64
	// createConflict makes Create fail with a conflict error once.
	createConflict bool
}

func (f *svbMQStore) Find(_ context.Context, id int64) (*types.MergeQueue, error) {
	if err := f.errs["Find"]; err != nil {
		return nil, err
	}
	q, ok := f.byID[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return q, nil
}

func (f *svbMQStore) FindByRepoAndBranch(_ context.Context, repoID int64, branch string) (*types.MergeQueue, error) {
	if err := f.errs["FindByRepoAndBranch"]; err != nil {
		return nil, err
	}
	for _, q := range f.byID {
		if q.RepoID == repoID && q.Branch == branch {
			return q, nil
		}
	}
	return nil, gitnessstore.ErrResourceNotFound
}

func (f *svbMQStore) Create(_ context.Context, q *types.MergeQueue) error {
	if f.createConflict {
		f.createConflict = false
		f.byID[99] = &types.MergeQueue{ID: 99, RepoID: q.RepoID, Branch: q.Branch}
		return gitnesserrors.Conflict("exists")
	}
	if err := f.errs["Create"]; err != nil {
		return err
	}
	q.ID = int64(len(f.byID) + 1)
	f.byID[q.ID] = q
	f.created = append(f.created, q)
	return nil
}

func (f *svbMQStore) Update(_ context.Context, q *types.MergeQueue) error {
	f.updated = append(f.updated, q)
	return f.errs["Update"]
}

func (f *svbMQStore) Delete(_ context.Context, id int64) error {
	f.deleted = append(f.deleted, id)
	return f.errs["Delete"]
}

func (f *svbMQStore) UpdateOptLock(
	_ context.Context, q *types.MergeQueue, mutate func(*types.MergeQueue) error,
) (*types.MergeQueue, error) {
	if err := f.errs["UpdateOptLock"]; err != nil {
		return nil, err
	}
	if err := mutate(q); err != nil {
		return nil, err
	}
	return q, nil
}

type svbEntryStore struct {
	store.MergeQueueEntryStore

	errs     svbErrs
	entries  map[int64]*types.MergeQueueEntry // by pull request ID
	overdue  []*types.MergeQueueEntry
	branches map[string]struct{}
	count    int64
	created  *types.MergeQueueEntry
	deleted  []int64
}

func (f *svbEntryStore) Find(_ context.Context, prID int64) (*types.MergeQueueEntry, error) {
	if err := f.errs["Find"]; err != nil {
		return nil, err
	}
	e, ok := f.entries[prID]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return e, nil
}

func (f *svbEntryStore) FindByMergeCommit(_ context.Context, commit sha.SHA) (*types.MergeQueueEntry, error) {
	if err := f.errs["FindByMergeCommit"]; err != nil {
		return nil, err
	}
	for _, e := range f.entries {
		if e.MergeCommitSHA.Equal(commit) {
			return e, nil
		}
	}
	return nil, gitnessstore.ErrResourceNotFound
}

func (f *svbEntryStore) Create(_ context.Context, e *types.MergeQueueEntry) error {
	if err := f.errs["Create"]; err != nil {
		return err
	}
	f.created = e
	f.entries[e.PullReqID] = e
	return nil
}

func (f *svbEntryStore) Update(_ context.Context, e *types.MergeQueueEntry) error {
	f.entries[e.PullReqID] = e
	return f.errs["Update"]
}

func (f *svbEntryStore) UpdateOptLock(
	_ context.Context, e *types.MergeQueueEntry, mutate func(*types.MergeQueueEntry) error,
) (*types.MergeQueueEntry, error) {
	if err := f.errs["UpdateOptLock"]; err != nil {
		return nil, err
	}
	if err := mutate(e); err != nil {
		return nil, err
	}
	return e, nil
}

func (f *svbEntryStore) Delete(_ context.Context, prID int64) error {
	f.deleted = append(f.deleted, prID)
	delete(f.entries, prID)
	return f.errs["Delete"]
}

func (f *svbEntryStore) DeleteAllForMergeQueue(context.Context, int64) error {
	return f.errs["DeleteAll"]
}

func (f *svbEntryStore) ListForMergeQueue(_ context.Context, qID int64) ([]*types.MergeQueueEntry, error) {
	if err := f.errs["ListForMergeQueue"]; err != nil {
		return nil, err
	}
	var out []*types.MergeQueueEntry
	for _, e := range f.entries {
		if e.MergeQueueID == qID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OrderIndex < out[j].OrderIndex })
	return out, nil
}

func (f *svbEntryStore) ListOverdueChecks(context.Context, int64) ([]*types.MergeQueueEntry, error) {
	return f.overdue, f.errs["ListOverdueChecks"]
}

func (f *svbEntryStore) CountForRepoAndBranch(context.Context, int64, string) (int64, error) {
	return f.count, f.errs["Count"]
}

func (f *svbEntryStore) BranchesWithPullReqInQueue(
	context.Context, int64, []string,
) (map[string]struct{}, error) {
	return f.branches, f.errs["Branches"]
}

type svbPullReqStore struct {
	store.PullReqStore

	errs    svbErrs
	prs     map[int64]*types.PullReq
	updated []*types.PullReq
}

func (f *svbPullReqStore) Find(_ context.Context, id int64) (*types.PullReq, error) {
	if err := f.errs["Find"]; err != nil {
		return nil, err
	}
	pr, ok := f.prs[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return pr, nil
}

func (f *svbPullReqStore) Update(_ context.Context, pr *types.PullReq) error {
	f.updated = append(f.updated, pr)
	return f.errs["Update"]
}

type svbActivityStore struct {
	store.PullReqActivityStore

	err   error
	count int
}

func (f *svbActivityStore) CreateWithPayload(
	context.Context, *types.PullReq, int64, types.PullReqActivityPayload, *types.PullReqActivityMetadata,
) (*types.PullReqActivity, error) {
	f.count++
	return &types.PullReqActivity{}, f.err
}

type svbCheckStore struct {
	store.CheckStore

	checks      []types.Check
	listErr     error
	byIdentFail bool
}

func (f *svbCheckStore) List(context.Context, int64, string, types.CheckListOptions) ([]types.Check, error) {
	return f.checks, f.listErr
}

func (f *svbCheckStore) FindByIdentifier(_ context.Context, _ int64, _ string, ident string) (types.Check, error) {
	if f.byIdentFail {
		return types.Check{}, errSvbBoom
	}
	for _, c := range f.checks {
		if c.Identifier == ident {
			return c, nil
		}
	}
	return types.Check{}, gitnessstore.ErrResourceNotFound
}

type svbRuleStore struct {
	store.RuleStore

	rules []types.RuleInfoInternal
	err   error
}

func (f *svbRuleStore) ListOnlyRepoRules(
	context.Context, *types.RepositoryCore, ...enum.RuleType,
) ([]types.RuleInfoInternal, error) {
	return f.rules, f.err
}

type svbRepoIDCache struct {
	store.RepoIDCache

	err error
}

func (f *svbRepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.RepositoryCore{ID: id, GitUID: "uid", DefaultBranch: "main", Path: "acct/org/proj/repo"}, nil
}

type svbGit struct {
	git.Interface

	refSHA    sha.SHA
	refErr    error
	mergeOut  git.MergeOutput
	mergeErr  error
	mergeCall git.MergeParams
}

func (g *svbGit) GetRef(context.Context, git.GetRefParams) (git.GetRefResponse, error) {
	return git.GetRefResponse{SHA: g.refSHA}, g.refErr
}

func (g *svbGit) Merge(_ context.Context, p *git.MergeParams) (git.MergeOutput, error) {
	g.mergeCall = *p
	return g.mergeOut, g.mergeErr
}

type svbProducer struct {
	mu      sync.Mutex
	streams []string
}

func (p *svbProducer) Send(_ context.Context, streamID string, _ map[string]any) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streams = append(p.streams, streamID)
	return "1", nil
}

func (p *svbProducer) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.streams)
}

type svbEnv struct {
	svc       *Service
	mqs       *svbMQStore
	entries   *svbEntryStore
	prs       *svbPullReqStore
	activity  *svbActivityStore
	checks    *svbCheckStore
	rules     *svbRuleStore
	repoCache *svbRepoIDCache
	git       *svbGit
	producer  *svbProducer
}

func svbNewLocker() *locker.Locker {
	return locker.NewLocker(lock.NewInMemory(lock.Config{
		App:           "test",
		Expiry:        10 * time.Second,
		Tries:         2,
		RetryDelay:    10 * time.Millisecond,
		DriftFactor:   0.01,
		TimeoutFactor: 0.25,
	}))
}

func svbNewEnv(t *testing.T) *svbEnv {
	t.Helper()

	env := &svbEnv{
		mqs: &svbMQStore{
			errs: svbErrs{},
			byID: map[int64]*types.MergeQueue{1: {ID: 1, RepoID: 10, Branch: "main"}},
		},
		entries:   &svbEntryStore{errs: svbErrs{}, entries: map[int64]*types.MergeQueueEntry{}},
		prs:       &svbPullReqStore{errs: svbErrs{}, prs: map[int64]*types.PullReq{}},
		activity:  &svbActivityStore{},
		checks:    &svbCheckStore{},
		rules:     &svbRuleStore{},
		repoCache: &svbRepoIDCache{},
		git:       &svbGit{refSHA: svbSHA(svbShaA)},
		producer:  &svbProducer{},
	}

	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errSvbBoom },
		env.producer,
		events.NewNoopCollector(),
		false,
	)
	require.NoError(t, err)
	reporter, err := mergequeueevents.NewReporter(system)
	require.NoError(t, err)

	env.svc = &Service{
		git:                     env.git,
		tx:                      svbTx{},
		mergeQueueEventReporter: reporter,
		repoFinder: refcache.NewRepoFinder(
			nil, nil, env.repoCache, nil, cache.Evictor[*types.RepositoryCore]{}),
		pullreqStore:         env.prs,
		activityStore:        env.activity,
		checkStore:           env.checks,
		mergeQueueStore:      env.mqs,
		mergeQueueEntryStore: env.entries,
		protectionManager:    svbNewManager(t, env.rules),
		locker:               svbNewLocker(),
	}
	return env
}

func svbNewManager(t *testing.T, rs store.RuleStore) *protection.Manager {
	t.Helper()
	manager := protection.NewManager(rs)
	require.NoError(t, manager.Register(protection.TypeBranch, func() protection.Definition {
		return &protection.Branch{}
	}))
	return manager
}

// svbActiveRules returns a branch rule that defines a merge queue with required check "ci".
func svbActiveRules() []types.RuleInfoInternal {
	return []types.RuleInfoInternal{{
		RuleInfo:   types.RuleInfo{ID: 1, Identifier: "r", Type: protection.TypeBranch, State: enum.RuleStateActive},
		RepoTarget: json.RawMessage(`{}`),
		Pattern:    json.RawMessage(`{"default":true}`),
		Definition: json.RawMessage(
			`{"pullreq":{"merge_queue":{"status_checks":{"require_identifiers":["ci"]},` +
				`"group_size":2,"checks_concurrency":1,"max_check_duration_seconds":60}}}`),
	}}
}

func svbRepo() *types.RepositoryCore {
	return &types.RepositoryCore{ID: 10, GitUID: "uid", DefaultBranch: "main", Path: "acct/org/proj/repo"}
}

func svbEntry(prID, order int64, state enum.MergeQueueEntryState) *types.MergeQueueEntry {
	return &types.MergeQueueEntry{
		PullReqID: prID, MergeQueueID: 1, OrderIndex: order, State: state,
		BaseCommitSHA: sha.None, HeadCommitSHA: sha.None, MergeCommitSHA: sha.None, MergeBaseSHA: sha.None,
		ChecksCommitSHA: sha.None, MergeMethod: enum.MergeMethodMerge,
	}
}

func svbPR(id int64) *types.PullReq {
	repoID := int64(10)
	return &types.PullReq{
		ID: id, Number: id, State: enum.PullReqStateOpen, TargetRepoID: 10, SourceRepoID: &repoID,
		TargetBranch: "main", SubState: enum.PullReqSubStateMergeQueue,
	}
}

func TestSvbBuildEntryInfo(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)
	entry := svbEntry(1, 1, enum.MergeQueueEntryStateChecksInProgress)
	entry.ChecksCommitSHA = svbSHA(svbShaB)
	env.checks.checks = []types.Check{
		{Identifier: "lint", Status: enum.CheckStatusSuccess},
		{Identifier: "ci", Status: enum.CheckStatusRunning},
	}

	info, err := env.svc.buildEntryInfo(ctx, 10, entry, 2, []string{"ci", "e2e"})
	require.NoError(t, err)
	require.Equal(t, 2, info.PullRequestsAhead)
	require.Equal(t, enum.MergeQueueEntryStateChecksInProgress, info.State)
	require.Len(t, info.Checks, 3)

	// Sorted by identifier: ci (required), e2e (required but not reported yet), lint (optional).
	require.Equal(t, "ci", info.Checks[0].Check.Identifier)
	require.True(t, info.Checks[0].Required)
	require.Equal(t, "e2e", info.Checks[1].Check.Identifier)
	require.True(t, info.Checks[1].Required)
	require.Equal(t, enum.CheckStatusPending, info.Checks[1].Check.Status)
	require.Equal(t, int64(10), info.Checks[1].Check.RepoID)
	require.Equal(t, "lint", info.Checks[2].Check.Identifier)
	require.False(t, info.Checks[2].Required)

	// Without a checks commit nothing is listed and all required checks are pending.
	entry.ChecksCommitSHA = sha.None
	info, err = env.svc.buildEntryInfo(ctx, 10, entry, 0, []string{"ci"})
	require.NoError(t, err)
	require.Len(t, info.Checks, 1)
	require.Equal(t, enum.CheckStatusPending, info.Checks[0].Check.Status)

	entry.ChecksCommitSHA = svbSHA(svbShaB)
	env.checks.listErr = errSvbBoom
	_, err = env.svc.buildEntryInfo(ctx, 10, entry, 0, nil)
	require.ErrorIs(t, err, errSvbBoom)
}

func TestSvbBuildMergeQueueInfo(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)
	env.entries.entries[1] = svbEntry(1, 1, enum.MergeQueueEntryStateMergePending)
	env.entries.entries[2] = svbEntry(2, 2, enum.MergeQueueEntryStateMergePending)
	setup := protection.MergeQueueSetup{RequiredChecks: []string{"ci"}}

	info, err := env.svc.BuildMergeQueueInfo(ctx, svbRepo(), env.entries.entries[2], setup)
	require.NoError(t, err)
	require.Equal(t, 1, info.PullRequestsAhead)

	_, err = env.svc.BuildMergeQueueInfo(ctx, svbRepo(), svbEntry(3, 3, enum.MergeQueueEntryStateMergePending), setup)
	require.Error(t, err, "entries that are not in the queue are rejected")

	env.entries.errs["ListForMergeQueue"] = errSvbBoom
	_, err = env.svc.BuildMergeQueueInfo(ctx, svbRepo(), env.entries.entries[1], setup)
	require.ErrorIs(t, err, errSvbBoom)
}

func TestSvbListMergeQueue(t *testing.T) {
	ctx := context.Background()
	setup := protection.MergeQueueSetup{RequiredChecks: []string{"ci"}}

	t.Run("lists in order", func(t *testing.T) {
		env := svbNewEnv(t)
		env.entries.entries[5] = svbEntry(5, 2, enum.MergeQueueEntryStateMergePending)
		env.entries.entries[4] = svbEntry(4, 1, enum.MergeQueueEntryStateMergePending)
		env.prs.prs[4], env.prs.prs[5] = svbPR(4), svbPR(5)

		items, err := env.svc.ListMergeQueue(ctx, svbRepo(), "main", setup)
		require.NoError(t, err)
		require.Len(t, items, 2)
		require.Equal(t, int64(4), items[0].PullReq.ID)
		require.Equal(t, 0, items[0].MergeQueueInfo.PullRequestsAhead)
		require.Equal(t, int64(5), items[1].PullReq.ID)
		require.Equal(t, 1, items[1].MergeQueueInfo.PullRequestsAhead)
	})

	t.Run("no queue", func(t *testing.T) {
		env := svbNewEnv(t)
		items, err := env.svc.ListMergeQueue(ctx, svbRepo(), "other", setup)
		require.NoError(t, err)
		require.Empty(t, items)
	})

	t.Run("failures", func(t *testing.T) {
		env := svbNewEnv(t)
		env.mqs.errs["FindByRepoAndBranch"] = errSvbBoom
		_, err := env.svc.ListMergeQueue(ctx, svbRepo(), "main", setup)
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(t)
		env.entries.errs["ListForMergeQueue"] = errSvbBoom
		_, err = env.svc.ListMergeQueue(ctx, svbRepo(), "main", setup)
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(t)
		env.entries.entries[4] = svbEntry(4, 1, enum.MergeQueueEntryStateMergePending)
		_, err = env.svc.ListMergeQueue(ctx, svbRepo(), "main", setup)
		require.ErrorContains(t, err, "failed to find pull request 4")

		env = svbNewEnv(t)
		entry := svbEntry(4, 1, enum.MergeQueueEntryStateChecksInProgress)
		entry.ChecksCommitSHA = svbSHA(svbShaB)
		env.entries.entries[4] = entry
		env.checks.listErr = errSvbBoom
		_, err = env.svc.ListMergeQueue(ctx, svbRepo(), "main", setup)
		require.ErrorContains(t, err, "failed to build merge queue info")
	})
}

func TestSvbBranchInQueue(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)

	env.entries.branches = map[string]struct{}{"main": {}}
	in, err := env.svc.IsBranchInQueue(ctx, 10, "main")
	require.NoError(t, err)
	require.True(t, in)

	violations, err := env.svc.BranchInQueueViolations(ctx, 10, "main")
	require.NoError(t, err)
	require.Len(t, violations, 1)
	require.Equal(t, RuleIdentifier, violations[0].Rule.Identifier)

	got, err := env.svc.BranchesWithPullReqInQueue(ctx, 10, []string{"main"})
	require.NoError(t, err)
	require.Contains(t, got, "main")

	env.entries.branches = nil
	in, err = env.svc.IsBranchInQueue(ctx, 10, "main")
	require.NoError(t, err)
	require.False(t, in)
	violations, err = env.svc.BranchInQueueViolations(ctx, 10, "main")
	require.NoError(t, err)
	require.Empty(t, violations)

	env.entries.errs["Branches"] = errSvbBoom
	_, err = env.svc.IsBranchInQueue(ctx, 10, "main")
	require.ErrorIs(t, err, errSvbBoom)
	_, err = env.svc.BranchInQueueViolations(ctx, 10, "main")
	require.ErrorIs(t, err, errSvbBoom)
	_, err = env.svc.BranchesWithPullReqInQueue(ctx, 10, []string{"main"})
	require.ErrorIs(t, err, errSvbBoom)
}

func TestSvbFindOrCreateMergeQueue(t *testing.T) {
	ctx := context.Background()

	t.Run("existing", func(t *testing.T) {
		env := svbNewEnv(t)
		q, err := env.svc.FindOrCreateMergeQueue(ctx, 10, "main")
		require.NoError(t, err)
		require.Equal(t, int64(1), q.ID)
		require.Empty(t, env.mqs.created)
	})

	t.Run("creates", func(t *testing.T) {
		env := svbNewEnv(t)
		q, err := env.svc.FindOrCreateMergeQueue(ctx, 10, "dev")
		require.NoError(t, err)
		require.Equal(t, "dev", q.Branch)
		require.NotZero(t, q.Created)
		require.Len(t, env.mqs.created, 1)
	})

	t.Run("conflict re-reads the queue", func(t *testing.T) {
		env := svbNewEnv(t)
		env.mqs.createConflict = true
		q, err := env.svc.FindOrCreateMergeQueue(ctx, 10, "dev")
		require.NoError(t, err)
		require.Equal(t, int64(99), q.ID)
	})

	t.Run("failures", func(t *testing.T) {
		env := svbNewEnv(t)
		env.mqs.errs["FindByRepoAndBranch"] = errSvbBoom
		_, err := env.svc.FindOrCreateMergeQueue(ctx, 10, "dev")
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(t)
		env.mqs.errs["Create"] = errSvbBoom
		_, err = env.svc.FindOrCreateMergeQueue(ctx, 10, "dev")
		require.ErrorIs(t, err, errSvbBoom)
	})
}

func TestSvbGetLastCommitSHA(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)

	got, err := env.svc.getLastCommitSHA(ctx, svbRepo(), "main")
	require.NoError(t, err)
	require.Equal(t, svbShaA, got.String())

	env.git.refErr = errSvbBoom
	_, err = env.svc.getLastCommitSHA(ctx, svbRepo(), "main")
	require.ErrorIs(t, err, errSvbBoom)
}

func TestSvbTriggerEvents(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)

	env.svc.TriggerBranchDeleted(ctx, 10, "main")
	env.svc.TriggerBranchUpdated(ctx, 10, "main")
	require.Equal(t, 2, env.producer.count())
	require.NotEqual(t, env.producer.streams[0], env.producer.streams[1])
}

func TestSvbEnqueue(t *testing.T) {
	ctx := context.Background()
	repo := svbRepo()

	newPR := func() *types.PullReq {
		pr := svbPR(7)
		pr.SubState = enum.PullReqSubStateNone
		return pr
	}

	t.Run("enqueues", func(t *testing.T) {
		env := svbNewEnv(t)
		env.prs.prs[7] = newPR()

		pr, entry, err := env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethodSquash, "title", "msg", true)
		require.NoError(t, err)
		require.Equal(t, enum.PullReqSubStateMergeQueue, pr.SubState)
		require.Equal(t, enum.MergeQueueEntryStateMergePending, entry.State)
		require.Equal(t, enum.MergeMethodSquash, entry.MergeMethod)
		require.Equal(t, int64(1), entry.OrderIndex)
		require.Equal(t, int64(3), entry.CreatedBy)
		require.Equal(t, "title", entry.CommitTitle)
		require.True(t, entry.DeleteSourceBranch)
		require.Equal(t, 1, env.activity.count)
		require.Equal(t, 1, env.producer.count())
	})

	t.Run("activity failure is tolerated", func(t *testing.T) {
		env := svbNewEnv(t)
		env.prs.prs[7] = newPR()
		env.activity.err = errSvbBoom
		_, _, err := env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethodMerge, "", "", false)
		require.NoError(t, err)
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		env := svbNewEnv(t)
		_, _, err := env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethod("bogus"), "", "", false)
		require.Error(t, err)

		_, _, err = env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethodFastForward, "", "", false)
		require.Error(t, err)

		draft := newPR()
		draft.IsDraft = true
		_, _, err = env.svc.Enqueue(ctx, draft, repo, 3, enum.MergeMethodMerge, "", "", false)
		require.Error(t, err)

		env.entries.count = MaximumQueueSize
		_, _, err = env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethodMerge, "", "", false)
		require.ErrorContains(t, err, "full")
	})

	t.Run("store failures", func(t *testing.T) {
		env := svbNewEnv(t)
		env.entries.errs["Count"] = errSvbBoom
		_, _, err := env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethodMerge, "", "", false)
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(t)
		env.mqs.errs["FindByRepoAndBranch"] = errSvbBoom
		_, _, err = env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethodMerge, "", "", false)
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(t)
		env.mqs.errs["UpdateOptLock"] = errSvbBoom
		_, _, err = env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethodMerge, "", "", false)
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(t)
		_, _, err = env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethodMerge, "", "", false)
		require.ErrorContains(t, err, "failed to find pull request")

		env = svbNewEnv(t)
		env.prs.prs[7] = newPR()
		env.entries.errs["Create"] = errSvbBoom
		_, _, err = env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethodMerge, "", "", false)
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(t)
		env.prs.prs[7] = newPR()
		env.prs.errs["Update"] = errSvbBoom
		_, _, err = env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethodMerge, "", "", false)
		require.ErrorIs(t, err, errSvbBoom)

		// The pull request is re-read inside the transaction and must still be queueable.
		env = svbNewEnv(t)
		stale := newPR()
		stale.State = enum.PullReqStateClosed
		env.prs.prs[7] = stale
		_, _, err = env.svc.Enqueue(ctx, newPR(), repo, 3, enum.MergeMethodMerge, "", "", false)
		require.Error(t, err)
	})
}

func TestSvbReserveSequenceNumber(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)
	q := &types.MergeQueue{ID: 1, OrderSequence: 4}

	got, seq, err := env.svc.reserveSequenceNumber(ctx, q)
	require.NoError(t, err)
	require.Equal(t, int64(5), seq)
	require.Equal(t, int64(5), got.OrderSequence)

	_, _, err = env.svc.reserveSequenceNumber(ctx, &types.MergeQueue{OrderSequence: -5})
	require.ErrorContains(t, err, "invalid sequence number")
}

func TestSvbPrioritize(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T) *svbEnv {
		env := svbNewEnv(t)
		for i := int64(1); i <= 3; i++ {
			env.entries.entries[i] = svbEntry(i, i, enum.MergeQueueEntryStateChecksPending)
			env.entries.entries[i].MergeCommitSHA = svbSHA(svbShaA)
		}
		env.mqs.byID[1].OrderSequence = 3
		return env
	}

	t.Run("moves the entry to the front and resets all entries", func(t *testing.T) {
		env := setup(t)
		require.NoError(t, env.svc.Prioritize(ctx, 3))

		entries, err := env.entries.ListForMergeQueue(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, []int64{3, 1, 2}, []int64{entries[0].PullReqID, entries[1].PullReqID, entries[2].PullReqID})
		for _, e := range entries {
			require.Equal(t, enum.MergeQueueEntryStateMergePending, e.State)
			require.True(t, e.MergeCommitSHA.IsNil() || e.MergeCommitSHA.IsEmpty())
		}
		require.Equal(t, int64(6), env.mqs.byID[1].OrderSequence)
		require.Equal(t, 1, env.producer.count())
	})

	t.Run("already head", func(t *testing.T) {
		env := setup(t)
		require.ErrorIs(t, env.svc.Prioritize(ctx, 1), ErrAlreadyHead)
	})

	t.Run("not in queue", func(t *testing.T) {
		env := setup(t)
		require.ErrorIs(t, env.svc.Prioritize(ctx, 99), ErrNotInQueue)
	})

	t.Run("failures", func(t *testing.T) {
		env := setup(t)
		env.entries.errs["Find"] = errSvbBoom
		require.ErrorIs(t, env.svc.Prioritize(ctx, 3), errSvbBoom)

		env = setup(t)
		env.mqs.errs["Find"] = errSvbBoom
		require.ErrorIs(t, env.svc.Prioritize(ctx, 3), errSvbBoom)

		env = setup(t)
		env.entries.errs["ListForMergeQueue"] = errSvbBoom
		require.ErrorIs(t, env.svc.Prioritize(ctx, 3), errSvbBoom)

		env = setup(t)
		env.mqs.errs["Update"] = errSvbBoom
		require.ErrorIs(t, env.svc.Prioritize(ctx, 3), errSvbBoom)

		env = setup(t)
		env.entries.errs["Update"] = errSvbBoom
		require.ErrorIs(t, env.svc.Prioritize(ctx, 3), errSvbBoom)
	})
}

func TestSvbRemove(t *testing.T) {
	ctx := context.Background()

	// A merged pull request is not touched by remove (and no activity is logged),
	// which keeps the queue bookkeeping testable in isolation.
	setup := func(t *testing.T) *svbEnv {
		env := svbNewEnv(t)
		a := svbEntry(1, 1, enum.MergeQueueEntryStateMergeGroup)
		a.ChecksCommitSHA = svbSHA(svbShaB)
		a.MergeCommitSHA = svbSHA(svbShaA)
		b := svbEntry(2, 2, enum.MergeQueueEntryStateChecksInProgress)
		b.ChecksCommitSHA = svbSHA(svbShaB)
		b.MergeCommitSHA = svbSHA(svbShaB)
		c := svbEntry(3, 3, enum.MergeQueueEntryStateChecksPending)
		c.MergeCommitSHA = svbSHA(svbShaC)
		env.entries.entries[1], env.entries.entries[2], env.entries.entries[3] = a, b, c
		for id := int64(1); id <= 3; id++ {
			pr := svbPR(id)
			pr.State = enum.PullReqStateMerged
			env.prs.prs[id] = pr
		}
		return env
	}

	remove := func(env *svbEnv, id int64) error {
		return env.svc.remove(ctx, id, types.PullRequestActivityPayloadMergeQueueRemove{
			Reason: enum.MergeQueueRemovalReasonCheckFail,
		})
	}

	t.Run("removing the chain leader re-queues the group and cancels the rest", func(t *testing.T) {
		env := setup(t)
		require.NoError(t, remove(env, 2))
		require.Equal(t, []int64{2}, env.entries.deleted)

		a := env.entries.entries[1]
		require.Equal(t, enum.MergeQueueEntryStateChecksPending, a.State)
		require.True(t, a.ChecksCommitSHA.IsNil() || a.ChecksCommitSHA.IsEmpty())

		c := env.entries.entries[3]
		require.Equal(t, enum.MergeQueueEntryStateMergePending, c.State)
		require.True(t, c.MergeCommitSHA.IsNil() || c.MergeCommitSHA.IsEmpty())
	})

	t.Run("not in queue", func(t *testing.T) {
		env := setup(t)
		require.Error(t, remove(env, 99), "unknown pull request")

		env = setup(t)
		env.mqs.byID = map[int64]*types.MergeQueue{}
		require.ErrorIs(t, remove(env, 2), ErrNotInQueue)

		env = setup(t)
		env.prs.prs[5] = svbPR(5)
		require.ErrorIs(t, remove(env, 5), ErrNotInQueue, "entry missing from the queue")
	})

	t.Run("failures", func(t *testing.T) {
		for _, key := range []string{"Find", "FindByRepoAndBranch"} {
			env := setup(t)
			env.prs.errs[key] = errSvbBoom
			env.mqs.errs[key] = errSvbBoom
			require.ErrorIs(t, remove(env, 2), errSvbBoom)
		}

		env := setup(t)
		env.entries.errs["ListForMergeQueue"] = errSvbBoom
		require.ErrorIs(t, remove(env, 2), errSvbBoom)

		env = setup(t)
		env.entries.errs["Delete"] = errSvbBoom
		require.ErrorIs(t, remove(env, 2), errSvbBoom)

		env = setup(t)
		env.entries.errs["Update"] = errSvbBoom
		require.ErrorIs(t, remove(env, 2), errSvbBoom)
	})

	t.Run("public Remove reports an event", func(t *testing.T) {
		env := setup(t)
		payload := types.PullRequestActivityPayloadMergeQueueRemove{Reason: enum.MergeQueueRemovalReasonCheckFail}
		require.NoError(t, env.svc.Remove(ctx, 2, payload))
		require.Equal(t, 1, env.producer.count())

		require.ErrorIs(t, env.svc.Remove(ctx, 99, payload), ErrNotInQueue)

		env = setup(t)
		env.entries.errs["Find"] = errSvbBoom
		require.ErrorIs(t, env.svc.Remove(ctx, 2, payload), errSvbBoom)

		env = setup(t)
		env.mqs.errs["Find"] = errSvbBoom
		require.ErrorIs(t, env.svc.Remove(ctx, 2, payload), errSvbBoom)

		env = setup(t)
		env.prs.errs["Find"] = errSvbBoom
		require.ErrorIs(t, env.svc.Remove(ctx, 2, payload), errSvbBoom)
	})

	t.Run("RemoveAll with no queue is a no-op", func(t *testing.T) {
		env := setup(t)
		payload := types.PullRequestActivityPayloadMergeQueueRemove{Reason: enum.MergeQueueRemovalReasonNoQueue}
		require.NoError(t, env.svc.RemoveAll(ctx, svbRepo(), "other", payload))

		env.mqs.errs["FindByRepoAndBranch"] = errSvbBoom
		require.ErrorIs(t, env.svc.RemoveAll(ctx, svbRepo(), "main", payload), errSvbBoom)

		env = setup(t)
		env.entries.errs["ListForMergeQueue"] = errSvbBoom
		require.ErrorIs(t, env.svc.RemoveAll(ctx, svbRepo(), "main", payload), errSvbBoom)

		env = setup(t)
		env.prs.errs["Find"] = errSvbBoom
		require.ErrorIs(t, env.svc.RemoveAll(ctx, svbRepo(), "main", payload), errSvbBoom)

		env = setup(t)
		env.entries.errs["DeleteAll"] = errSvbBoom
		require.ErrorIs(t, env.svc.RemoveAll(ctx, svbRepo(), "main", payload), errSvbBoom)

		env = setup(t)
		env.mqs.errs["Delete"] = errSvbBoom
		require.ErrorIs(t, env.svc.RemoveAll(ctx, svbRepo(), "main", payload), errSvbBoom)

		env = setup(t)
		for _, pr := range env.prs.prs {
			pr.SubState = enum.PullReqSubStateMergeQueue
		}
		env.prs.errs["Update"] = errSvbBoom
		require.ErrorIs(t, env.svc.RemoveAll(ctx, svbRepo(), "main", payload), errSvbBoom)
	})
}

func TestSvbHandlerUpdated(t *testing.T) {
	ctx := context.Background()
	ev := &events.Event[*mergequeueevents.UpdatedPayload]{
		Payload: &mergequeueevents.UpdatedPayload{Base: mergequeueevents.Base{RepoID: 10, Branch: "main"}},
	}

	t.Run("inactive merge queue setup is ignored", func(t *testing.T) {
		env := svbNewEnv(t)
		require.NoError(t, env.svc.handlerUpdated(ctx, ev))
	})

	t.Run("active setup without a queue", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.rules = svbActiveRules()
		env.mqs.byID = map[int64]*types.MergeQueue{}
		require.NoError(t, env.svc.handlerUpdated(ctx, ev))
	})

	t.Run("deleted target branch clears the queue", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.rules = svbActiveRules()
		env.git.refErr = gitnesserrors.NotFound("no branch")
		// The queue exists but RemoveAll can't find it again.
		env.mqs.errs = svbErrs{}
		env.mqs.byID = map[int64]*types.MergeQueue{}
		require.NoError(t, env.svc.handlerUpdated(ctx, ev))
	})

	t.Run("failures", func(t *testing.T) {
		env := svbNewEnv(t)
		env.repoCache.err = errSvbBoom
		require.ErrorIs(t, env.svc.handlerUpdated(ctx, ev), errSvbBoom)

		env = svbNewEnv(t)
		env.rules.err = errSvbBoom
		require.ErrorIs(t, env.svc.handlerUpdated(ctx, ev), errSvbBoom)

		env = svbNewEnv(t)
		env.rules.rules = svbActiveRules()
		env.mqs.errs["FindByRepoAndBranch"] = errSvbBoom
		require.ErrorIs(t, env.svc.handlerUpdated(ctx, ev), errSvbBoom)

		env = svbNewEnv(t)
		env.rules.rules = svbActiveRules()
		env.git.refErr = errSvbBoom
		require.ErrorIs(t, env.svc.handlerUpdated(ctx, ev), errSvbBoom)

		env = svbNewEnv(t)
		env.rules.rules = []types.RuleInfoInternal{{
			RuleInfo:   types.RuleInfo{Type: protection.TypeBranch},
			RepoTarget: json.RawMessage(`bad`),
			Pattern:    json.RawMessage(`{}`),
			Definition: json.RawMessage(`{}`),
		}}
		require.Error(t, env.svc.handlerUpdated(ctx, ev))
	})
}

func TestSvbHandlerBranchDeleted(t *testing.T) {
	ctx := context.Background()
	ev := &events.Event[*mergequeueevents.BranchDeletedPayload]{
		Payload: &mergequeueevents.BranchDeletedPayload{Base: mergequeueevents.Base{RepoID: 10, Branch: "gone"}},
	}

	env := svbNewEnv(t)
	require.NoError(t, env.svc.handlerBranchDeleted(ctx, ev), "no queue for the branch")

	env.repoCache.err = errSvbBoom
	require.ErrorIs(t, env.svc.handlerBranchDeleted(ctx, ev), errSvbBoom)

	env = svbNewEnv(t)
	env.mqs.errs["FindByRepoAndBranch"] = errSvbBoom
	require.ErrorContains(t, env.svc.handlerBranchDeleted(ctx, ev), "failed to clear merge queue")
}

func TestSvbHandlerCheckFinished(t *testing.T) {
	ctx := context.Background()
	mk := func(status enum.CheckStatus, bypassed bool, commit string) *events.Event[*checkevents.ReportedPayload] {
		return &events.Event[*checkevents.ReportedPayload]{Payload: &checkevents.ReportedPayload{
			Base: checkevents.Base{RepoID: 10, SHA: commit}, Identifier: "ci", Status: status, Bypassed: bypassed,
		}}
	}

	setup := func(t *testing.T) *svbEnv {
		env := svbNewEnv(t)
		env.rules.rules = svbActiveRules()
		entry := svbEntry(1, 1, enum.MergeQueueEntryStateChecksInProgress)
		entry.MergeCommitSHA = svbSHA(svbShaB)
		entry.ChecksCommitSHA = svbSHA(svbShaB)
		env.entries.entries[1] = entry
		pr := svbPR(1)
		pr.State = enum.PullReqStateMerged
		env.prs.prs[1] = pr
		return env
	}

	t.Run("ignores checks that are not complete", func(t *testing.T) {
		env := setup(t)
		require.NoError(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusRunning, false, svbShaB)))
	})

	t.Run("invalid sha is discarded", func(t *testing.T) {
		env := setup(t)
		require.Error(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusSuccess, false, "zz")))
	})

	t.Run("not a merge queue commit", func(t *testing.T) {
		env := setup(t)
		require.NoError(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusSuccess, false, svbShaC)))
	})

	t.Run("inactive setup with no queue", func(t *testing.T) {
		env := setup(t)
		env.rules.rules = nil
		env.mqs.byID[1].Branch = "main"
		env.mqs.errs["FindByRepoAndBranch"] = gitnessstore.ErrResourceNotFound
		require.NoError(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusSuccess, false, svbShaB)))
	})

	t.Run("failed check removes the entry and reprocesses", func(t *testing.T) {
		env := setup(t)
		env.checks.checks = []types.Check{{Identifier: "ci", Status: enum.CheckStatusFailure, Link: "http://x"}}
		// Reprocessing fails on the target branch lookup, after the entry was removed.
		env.git.refErr = errSvbBoom
		err := env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusFailure, false, svbShaB))
		require.ErrorIs(t, err, errSvbBoom)
		require.ErrorContains(t, err, "after removing entry because failed check")
		require.Equal(t, []int64{1}, env.entries.deleted)

		// Failure to find the check does not block the removal.
		env = setup(t)
		env.checks.byIdentFail = true
		env.git.refErr = errSvbBoom
		err = env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusFailure, false, svbShaB))
		require.ErrorContains(t, err, "after removing entry because failed check")
		require.Equal(t, []int64{1}, env.entries.deleted)
	})

	t.Run("failed check for entry no longer queued", func(t *testing.T) {
		env := setup(t)
		delete(env.prs.prs, 1)
		env.prs.prs[1] = svbPR(1)
		env.mqs.byID[1].Branch = "main"
		env.prs.prs[1].TargetBranch = "other"
		require.NoError(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusFailure, false, svbShaB)))
	})

	t.Run("failed check removal error", func(t *testing.T) {
		env := setup(t)
		env.entries.errs["Delete"] = errSvbBoom
		require.ErrorIs(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusFailure, false, svbShaB)), errSvbBoom)
	})

	t.Run("waits for the other required checks", func(t *testing.T) {
		env := setup(t)
		env.checks.checks = []types.Check{{Identifier: "lint", Status: enum.CheckStatusSuccess}}
		require.NoError(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusSuccess, false, svbShaB)))
	})

	t.Run("only the checks leader acts on success", func(t *testing.T) {
		env := setup(t)
		env.entries.entries[1].State = enum.MergeQueueEntryStateMergeGroup
		require.NoError(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusSuccess, false, svbShaB)))
	})

	t.Run("failures", func(t *testing.T) {
		env := setup(t)
		env.repoCache.err = errSvbBoom
		require.ErrorIs(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusSuccess, false, svbShaB)), errSvbBoom)

		env = setup(t)
		env.entries.errs["FindByMergeCommit"] = errSvbBoom
		require.ErrorIs(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusSuccess, false, svbShaB)), errSvbBoom)

		env = setup(t)
		env.mqs.errs["Find"] = errSvbBoom
		require.ErrorIs(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusSuccess, false, svbShaB)), errSvbBoom)

		env = setup(t)
		env.rules.err = errSvbBoom
		require.ErrorIs(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusSuccess, false, svbShaB)), errSvbBoom)

		env = setup(t)
		env.checks.listErr = errSvbBoom
		require.ErrorIs(t, env.svc.handlerCheckFinished(ctx, mk(enum.CheckStatusSuccess, false, svbShaB)), errSvbBoom)
	})
}

func TestSvbListChecks(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)
	env.checks.checks = []types.Check{{Identifier: "a"}}

	got, err := env.svc.ListChecks(ctx, 10, svbSHA(svbShaA))
	require.NoError(t, err)
	require.Len(t, got, 1)

	env.checks.listErr = errSvbBoom
	_, err = env.svc.ListChecks(ctx, 10, svbSHA(svbShaA))
	require.ErrorIs(t, err, errSvbBoom)
}

func TestSvbJobOverdueChecksHandle(t *testing.T) {
	ctx := context.Background()
	deadline := time.Now().Add(-time.Minute).UnixMilli()

	setup := func(t *testing.T) (*svbEnv, *jobOverdueChecks) {
		env := svbNewEnv(t)
		entry := svbEntry(1, 1, enum.MergeQueueEntryStateChecksInProgress)
		entry.ChecksCommitSHA = svbSHA(svbShaB)
		entry.ChecksStarted = &deadline
		entry.ChecksDeadline = &deadline
		env.entries.overdue = []*types.MergeQueueEntry{entry}
		env.entries.entries[1] = entry
		pr := svbPR(1)
		pr.State = enum.PullReqStateMerged
		env.prs.prs[1] = pr
		env.checks.checks = []types.Check{
			{Identifier: "done", Status: enum.CheckStatusSuccess},
			{Identifier: "slow", Status: enum.CheckStatusRunning, Link: "http://slow"},
		}
		return env, &jobOverdueChecks{mergeQueueEntryStore: env.entries, service: env.svc}
	}

	t.Run("removes overdue entries", func(t *testing.T) {
		env, job := setup(t)
		out, err := job.Handle(ctx, "", nil)
		require.NoError(t, err)
		require.Empty(t, out)
		require.Equal(t, []int64{1}, env.entries.deleted)
	})

	t.Run("nothing overdue", func(t *testing.T) {
		env, job := setup(t)
		env.entries.overdue = nil
		_, err := job.Handle(ctx, "", nil)
		require.NoError(t, err)
		require.Empty(t, env.entries.deleted)
	})

	t.Run("removal errors are logged and skipped", func(t *testing.T) {
		env, job := setup(t)
		env.entries.errs["Delete"] = errSvbBoom
		_, err := job.Handle(ctx, "", nil)
		require.NoError(t, err)
	})

	t.Run("list failure", func(t *testing.T) {
		env, job := setup(t)
		env.entries.errs["ListOverdueChecks"] = errSvbBoom
		_, err := job.Handle(ctx, "", nil)
		require.ErrorIs(t, err, errSvbBoom)
	})
}

func TestSvbGetCheckInfo(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)
	entry := svbEntry(1, 1, enum.MergeQueueEntryStateChecksInProgress)
	entry.ChecksCommitSHA = svbSHA(svbShaB)
	bypassedBy := int64(5)
	env.checks.checks = []types.Check{
		{Identifier: "done", Status: enum.CheckStatusSuccess},
		{Identifier: "bypassed", Status: enum.CheckStatusRunning, BypassedByID: &bypassedBy},
		{Identifier: "slow", Status: enum.CheckStatusRunning, Link: "http://slow"},
	}
	job := &jobOverdueChecks{mergeQueueEntryStore: env.entries, service: env.svc}

	var cache repoCache
	ident, link := job.getCheckInfo(ctx, entry, &cache, zerolog.Nop())
	require.Equal(t, "slow", ident)
	require.Equal(t, "http://slow", link)

	env.checks.listErr = errSvbBoom
	ident, link = job.getCheckInfo(ctx, entry, &cache, zerolog.Nop())
	require.Empty(t, ident)
	require.Empty(t, link)

	env.mqs.byID = map[int64]*types.MergeQueue{}
	ident, link = job.getCheckInfo(ctx, entry, &repoCache{}, zerolog.Nop())
	require.Empty(t, ident)
	require.Empty(t, link)
}

func TestSvbReprocessTargetBranchHandling(t *testing.T) {
	ctx := context.Background()
	q := &types.MergeQueue{ID: 1, RepoID: 10, Branch: "main"}

	// A missing target branch clears the queue; with no queue row left nothing else happens.
	env := svbNewEnv(t)
	env.mqs.byID = map[int64]*types.MergeQueue{}
	env.git.refErr = gitnesserrors.NotFound("gone")
	require.NoError(t, env.svc.reprocess(ctx, svbRepo(), q, protection.MergeQueueSetup{}))

	env = svbNewEnv(t)
	env.mqs.byID = map[int64]*types.MergeQueue{}
	env.git.refErr = gitnesserrors.NotFound("gone")
	env.mqs.errs["FindByRepoAndBranch"] = errSvbBoom
	require.ErrorContains(t, env.svc.reprocess(ctx, svbRepo(), q, protection.MergeQueueSetup{}), "target branch")

	env = svbNewEnv(t)
	env.git.refErr = errSvbBoom
	require.ErrorIs(t, env.svc.reprocess(ctx, svbRepo(), q, protection.MergeQueueSetup{}), errSvbBoom)
}

func TestSvbResetEntries(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)
	q := env.mqs.byID[1]

	pending := svbEntry(1, 1, enum.MergeQueueEntryStateMergePending)
	group := svbEntry(2, 2, enum.MergeQueueEntryStateMergeGroup)
	group.MergeCommitSHA = svbSHA(svbShaA)
	group.HeadCommitSHA = svbSHA(svbShaB)
	// A zero checks SHA makes stopChecks a no-op for the in-progress entry.
	leader := svbEntry(3, 3, enum.MergeQueueEntryStateChecksInProgress)
	leader.MergeCommitSHA = svbSHA(svbShaC)

	env.svc.resetEntries(ctx, q, []*types.MergeQueueEntry{pending, group, leader})
	for _, e := range []*types.MergeQueueEntry{pending, group, leader} {
		require.Equal(t, enum.MergeQueueEntryStateMergePending, e.State)
	}
	require.True(t, group.MergeCommitSHA.IsNil() || group.MergeCommitSHA.IsEmpty())
	require.True(t, group.HeadCommitSHA.IsNil() || group.HeadCommitSHA.IsEmpty())

	// Failures to update an entry are logged, not returned.
	env.entries.errs["UpdateOptLock"] = errSvbBoom
	env.svc.resetEntries(ctx, q, []*types.MergeQueueEntry{svbEntry(4, 4, enum.MergeQueueEntryStateMergeGroup)})
}

func TestSvbStartStopChecksIgnoreInvalidSHA(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)
	q := env.mqs.byID[1]

	env.svc.startChecks(ctx, q, sha.None)
	env.svc.startChecks(ctx, q, sha.Nil)
	env.svc.stopChecks(ctx, q, sha.None)
	env.svc.stopChecks(ctx, q, sha.Nil)
	require.Zero(t, env.producer.count())
}

func TestSvbResetAndFastForwardGuards(t *testing.T) {
	ctx := context.Background()
	q := &types.MergeQueue{ID: 1, RepoID: 10, Branch: "main"}

	t.Run("reset returns when the branch is locked", func(t *testing.T) {
		env := svbNewEnv(t)
		unlock, err := env.svc.locker.LockBranch(ctx, 10, "main", 10*time.Second)
		require.NoError(t, err)
		defer unlock()
		env.svc.reset(ctx, svbRepo(), q)
		require.Zero(t, env.producer.count())
	})

	t.Run("reset returns when listing entries fails", func(t *testing.T) {
		env := svbNewEnv(t)
		env.entries.errs["ListForMergeQueue"] = errSvbBoom
		env.svc.reset(ctx, svbRepo(), q)
	})

	t.Run("fastForward guards", func(t *testing.T) {
		entry := svbEntry(1, 1, enum.MergeQueueEntryStateChecksInProgress)

		env := svbNewEnv(t)
		env.repoCache.err = errSvbBoom
		require.ErrorIs(t, env.svc.fastForward(ctx, q, entry), errSvbBoom)

		env = svbNewEnv(t)
		unlock, err := env.svc.locker.LockBranch(ctx, 10, "main", 10*time.Second)
		require.NoError(t, err)
		defer unlock()
		require.ErrorContains(t, env.svc.fastForward(ctx, q, entry), "failed to lock repository")

		env = svbNewEnv(t)
		env.entries.errs["ListForMergeQueue"] = errSvbBoom
		require.ErrorIs(t, env.svc.fastForward(ctx, q, entry), errSvbBoom)

		env = svbNewEnv(t)
		require.ErrorContains(t, env.svc.fastForward(ctx, q, entry), "failed to find merge entries")
	})
}

func TestSvbCreateMergeCommit(t *testing.T) {
	ctx := context.Background()
	entry := svbEntry(1, 1, enum.MergeQueueEntryStateMergePending)
	entry.MergeMethod = enum.MergeMethodSquash
	input := merge.PullReqGitInput{CommitMessage: "msg"}

	env := svbNewEnv(t)
	env.git.mergeOut = git.MergeOutput{MergeSHA: svbSHA(svbShaC)}
	out, err := env.svc.createMergeCommit(
		ctx, git.WriteParams{}, svbSHA(svbShaA), svbSHA(svbShaB), entry, input)
	require.NoError(t, err)
	require.Equal(t, svbShaC, out.MergeSHA.String())
	require.Equal(t, "msg", env.git.mergeCall.Message)
	require.Equal(t, svbShaA, env.git.mergeCall.BaseSHA.String())
	require.Equal(t, svbShaB, env.git.mergeCall.HeadSHA.String())
	require.Nil(t, env.git.mergeCall.Refs, "merge queue commits must not update any refs")

	env.git.mergeOut = git.MergeOutput{}
	_, err = env.svc.createMergeCommit(ctx, git.WriteParams{}, svbSHA(svbShaA), svbSHA(svbShaB), entry, input)
	require.ErrorIs(t, err, errMergeConflict)

	env.git.mergeOut = git.MergeOutput{MergeSHA: svbSHA(svbShaC), ConflictFiles: []string{"a.txt"}}
	_, err = env.svc.createMergeCommit(ctx, git.WriteParams{}, svbSHA(svbShaA), svbSHA(svbShaB), entry, input)
	require.ErrorIs(t, err, errMergeConflict)

	env.git.mergeErr = errSvbBoom
	_, err = env.svc.createMergeCommit(ctx, git.WriteParams{}, svbSHA(svbShaA), svbSHA(svbShaB), entry, input)
	require.ErrorIs(t, err, errSvbBoom)
}
