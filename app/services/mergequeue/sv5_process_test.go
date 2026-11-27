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
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
	checkevents "github.com/harness/gitness/app/events/check"
	gitevents "github.com/harness/gitness/app/events/git"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/merge"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	gitnessurl "github.com/harness/gitness/app/url"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/lock"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

const sv5SystemPrincipalID = 77

func sv5SHA(n int) sha.SHA { return sha.Must(fmt.Sprintf("%040x", n)) }

type sv5PrincipalStore struct{ store.PrincipalStore }

func (sv5PrincipalStore) FindServiceByUID(context.Context, string) (*types.Service, error) {
	return &types.Service{ID: sv5SystemPrincipalID, UID: "sys", DisplayName: "System", Admin: true}, nil
}

// sv5Booted is set once at package initialization: the system service principal is package-level state.
var sv5Booted = sv5InitBootstrap()

func sv5InitBootstrap() bool {
	cfg := &types.Config{}
	cfg.Principal.System.UID = "sys"
	ctrl := service.NewController(check.PrincipalUIDDefault, nil, sv5PrincipalStore{})
	if err := bootstrap.SystemService(context.Background(), cfg, ctrl); err != nil {
		panic(err)
	}
	return true
}

func sv5Bootstrap(t *testing.T) {
	t.Helper()
	require.True(t, sv5Booted)
}

type sv5Tx struct{}

func (sv5Tx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

type sv5Errs map[string]error

type sv5MQStore struct {
	store.MergeQueueStore

	errs    sv5Errs
	queues  map[int64]*types.MergeQueue
	deleted []int64
}

func (f *sv5MQStore) Find(_ context.Context, id int64) (*types.MergeQueue, error) {
	if err := f.errs["Find"]; err != nil {
		return nil, err
	}
	q, ok := f.queues[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return q, nil
}

func (f *sv5MQStore) FindByRepoAndBranch(_ context.Context, repoID int64, branch string) (*types.MergeQueue, error) {
	if err := f.errs["FindByRepoAndBranch"]; err != nil {
		return nil, err
	}
	for _, q := range f.queues {
		if q.RepoID == repoID && q.Branch == branch {
			return q, nil
		}
	}
	return nil, gitnessstore.ErrResourceNotFound
}

func (f *sv5MQStore) Delete(_ context.Context, id int64) error {
	f.deleted = append(f.deleted, id)
	return f.errs["Delete"]
}

type sv5EntryStore struct {
	store.MergeQueueEntryStore

	mu      sync.Mutex
	errs    sv5Errs
	entries map[int64]*types.MergeQueueEntry
	deleted []int64
}

func (f *sv5EntryStore) Find(_ context.Context, prID int64) (*types.MergeQueueEntry, error) {
	if err := f.errs["Find"]; err != nil {
		return nil, err
	}
	e, ok := f.entries[prID]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return e, nil
}

func (f *sv5EntryStore) Update(_ context.Context, e *types.MergeQueueEntry) error {
	f.entries[e.PullReqID] = e
	return f.errs["Update"]
}

func (f *sv5EntryStore) UpdateOptLock(
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

func (f *sv5EntryStore) Delete(_ context.Context, prID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, prID)
	delete(f.entries, prID)
	return f.errs["Delete"]
}

func (f *sv5EntryStore) DeleteAllForMergeQueue(context.Context, int64) error {
	return f.errs["DeleteAll"]
}

func (f *sv5EntryStore) ListForMergeQueue(_ context.Context, qID int64) ([]*types.MergeQueueEntry, error) {
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

type sv5PullReqStore struct {
	store.PullReqStore

	errs    sv5Errs
	prs     map[int64]*types.PullReq
	updated []int64
}

func (f *sv5PullReqStore) Find(_ context.Context, id int64) (*types.PullReq, error) {
	if err := f.errs["Find"]; err != nil {
		return nil, err
	}
	pr, ok := f.prs[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return pr, nil
}

func (f *sv5PullReqStore) Update(_ context.Context, pr *types.PullReq) error {
	f.updated = append(f.updated, pr.ID)
	return f.errs["Update"]
}

type sv5ActivityStore struct {
	store.PullReqActivityStore

	err      error
	payloads []types.PullReqActivityPayload
}

func (f *sv5ActivityStore) CreateWithPayload(
	_ context.Context, _ *types.PullReq, _ int64, p types.PullReqActivityPayload, _ *types.PullReqActivityMetadata,
) (*types.PullReqActivity, error) {
	f.payloads = append(f.payloads, p)
	return &types.PullReqActivity{}, f.err
}

type sv5RepoStore struct{ store.RepoStore }

func (sv5RepoStore) Find(context.Context, int64) (*types.Repository, error) {
	return &types.Repository{ID: 10, NumOpenPulls: 5}, nil
}

func (sv5RepoStore) UpdateOptLock(
	_ context.Context, r *types.Repository, fn func(*types.Repository) error,
) (*types.Repository, error) {
	cp := *r
	return &cp, fn(&cp)
}

type sv5RuleStore struct {
	store.RuleStore

	rules []types.RuleInfoInternal
}

func (f *sv5RuleStore) ListAllRepoRules(context.Context, int64, ...enum.RuleType) ([]types.RuleInfoInternal, error) {
	return f.rules, nil
}

func (f *sv5RuleStore) ListOnlyRepoRules(
	context.Context, *types.RepositoryCore, ...enum.RuleType,
) ([]types.RuleInfoInternal, error) {
	return f.rules, nil
}

type sv5RepoIDCache struct{ store.RepoIDCache }

func (sv5RepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if id == 404 {
		return nil, errSv5Boom
	}
	return &types.RepositoryCore{
		ID: id, GitUID: "uid", DefaultBranch: "main", Path: "acct/org/proj/repo", ParentID: 3,
	}, nil
}

type sv5Git struct {
	git.Interface

	mu         sync.Mutex
	targetSHA  sha.SHA
	getRefErr  error
	mergeErrBy map[string]error // by head SHA
	conflictBy map[string]bool  // by head SHA
	mergeCalls []git.MergeParams
	nextMerge  int
	updateRefs []git.UpdateRefParams
	refsErr    error
	refsCalls  []git.UpdateRefsParams
	updateErr  error
	deleted    []string
}

func (g *sv5Git) GetRef(context.Context, git.GetRefParams) (git.GetRefResponse, error) {
	return git.GetRefResponse{SHA: g.targetSHA}, g.getRefErr
}

func (g *sv5Git) Merge(_ context.Context, p *git.MergeParams) (git.MergeOutput, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.mergeCalls = append(g.mergeCalls, *p)
	if err := g.mergeErrBy[p.HeadSHA.String()]; err != nil {
		return git.MergeOutput{}, err
	}
	if g.conflictBy[p.HeadSHA.String()] {
		return git.MergeOutput{}, nil
	}
	g.nextMerge++
	return git.MergeOutput{
		BaseSHA: p.BaseSHA, HeadSHA: p.HeadSHA, MergeSHA: sv5SHA(1000 + g.nextMerge),
		MergeBaseSHA: sv5SHA(2000), CommitCount: 2, ChangedFileCount: 3, Additions: 4, Deletions: 5,
	}, nil
}

func (g *sv5Git) UpdateRef(_ context.Context, p git.UpdateRefParams) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.updateRefs = append(g.updateRefs, p)
	return g.updateErr
}

func (g *sv5Git) UpdateRefs(_ context.Context, p git.UpdateRefsParams) error {
	g.refsCalls = append(g.refsCalls, p)
	return g.refsErr
}

func (g *sv5Git) DeleteBranch(_ context.Context, p *git.DeleteBranchParams) error {
	g.deleted = append(g.deleted, p.BranchName)
	return nil
}

func (g *sv5Git) GetTreeNode(context.Context, *git.GetTreeNodeParams) (*git.GetTreeNodeOutput, error) {
	return nil, gitnesserrors.NotFound("none")
}

type sv5Producer struct {
	mu      sync.Mutex
	streams []string
}

func (p *sv5Producer) Send(_ context.Context, streamID string, _ map[string]any) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streams = append(p.streams, streamID)
	return "1", nil
}

func (p *sv5Producer) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.streams)
}

type sv5URLProvider struct{ gitnessurl.Provider }

func (sv5URLProvider) GetInternalAPIURL(context.Context) string { return "http://internal.test" }

type sv5UserGroups struct{ usergroup.Service }

func (sv5UserGroups) ListUserIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return []int64{}, nil
}

func (sv5UserGroups) MapGroupIDsToPrincipals(context.Context, []int64) (map[int64][]*types.Principal, error) {
	return map[int64][]*types.Principal{}, nil
}

type sv5Instrument struct{ instrument.Service }

func (sv5Instrument) Track(context.Context, instrument.Event) error { return nil }

type sv5SSE struct{ sse.Streamer }

func (sv5SSE) Publish(context.Context, int64, enum.SSEType, any) {}

type sv5CheckStore struct {
	store.CheckStore

	checks []types.Check
}

func (*sv5CheckStore) ListResults(context.Context, int64, string) ([]types.CheckResult, error) {
	return []types.CheckResult{}, nil
}

func (c *sv5CheckStore) List(context.Context, int64, string, types.CheckListOptions) ([]types.Check, error) {
	return c.checks, nil
}

type sv5ReviewerStore struct{ store.PullReqReviewerStore }

func (sv5ReviewerStore) List(context.Context, int64) ([]*types.PullReqReviewer, error) {
	return []*types.PullReqReviewer{}, nil
}

type sv5AutoMergeStore struct{ store.AutoMergeStore }

func (sv5AutoMergeStore) Delete(context.Context, int64) (bool, error) { return false, nil }

type sv5Env struct {
	svc      *Service
	mqs      *sv5MQStore
	entries  *sv5EntryStore
	prs      *sv5PullReqStore
	activity *sv5ActivityStore
	git      *sv5Git
	producer *sv5Producer
	rules    *sv5RuleStore
	checks   *sv5CheckStore
	repo     *types.RepositoryCore
	queue    *types.MergeQueue
}

func sv5NewEnv(t *testing.T) *sv5Env {
	t.Helper()
	sv5Bootstrap(t)

	env := &sv5Env{
		mqs: &sv5MQStore{
			errs:   sv5Errs{},
			queues: map[int64]*types.MergeQueue{1: {ID: 1, RepoID: 10, Branch: "main"}},
		},
		entries:  &sv5EntryStore{errs: sv5Errs{}, entries: map[int64]*types.MergeQueueEntry{}},
		prs:      &sv5PullReqStore{errs: sv5Errs{}, prs: map[int64]*types.PullReq{}},
		activity: &sv5ActivityStore{},
		git: &sv5Git{
			targetSHA:  sv5SHA(1),
			mergeErrBy: map[string]error{},
			conflictBy: map[string]bool{},
		},
		producer: &sv5Producer{},
		rules:    &sv5RuleStore{},
		checks:   &sv5CheckStore{},
		repo: &types.RepositoryCore{
			ID: 10, GitUID: "uid", DefaultBranch: "main", Path: "acct/org/proj/repo", ParentID: 3,
		},
	}
	env.queue = env.mqs.queues[1]

	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errSv5Boom },
		env.producer,
		events.NewNoopCollector(),
		false,
	)
	require.NoError(t, err)
	mqReporter, err := mergequeueevents.NewReporter(system)
	require.NoError(t, err)
	gitReporter, err := gitevents.NewReporter(system)
	require.NoError(t, err)
	prReporter, err := pullreqevents.NewReporter(system)
	require.NoError(t, err)

	finder := refcache.NewRepoFinder(nil, nil, sv5RepoIDCache{}, nil, cache.Evictor[*types.RepositoryCore]{})

	mergeSvc := merge.NewService(
		env.git, sv5Tx{}, prReporter, finder, sv5RepoStore{}, env.prs, env.activity, env.checks,
		sv5ReviewerStore{}, sv5AutoMergeStore{},
		codeowners.New(nil, env.git, codeowners.Config{FilePaths: []string{"CODEOWNERS"}}, nil, nil),
		sv5UserGroups{}, sv5URLProvider{}, sv5SSE{}, sv5Instrument{}, nil,
	)

	manager := protection.NewManager(env.rules)
	require.NoError(t, manager.Register(protection.TypeBranch, func() protection.Definition {
		return &protection.Branch{}
	}))

	env.svc = &Service{
		git:                     env.git,
		tx:                      sv5Tx{},
		gitEventReporter:        gitReporter,
		mergeQueueEventReporter: mqReporter,
		repoFinder:              finder,
		repoStore:               sv5RepoStore{},
		pullreqStore:            env.prs,
		activityStore:           env.activity,
		checkStore:              env.checks,
		mergeQueueStore:         env.mqs,
		mergeQueueEntryStore:    env.entries,
		protectionManager:       manager,
		mergeService:            mergeSvc,
		urlProvider:             sv5URLProvider{},
		locker: locker.NewLocker(lock.NewInMemory(lock.Config{
			App: "test", Expiry: 10 * time.Second, Tries: 2, RetryDelay: 10 * time.Millisecond,
			DriftFactor: 0.01, TimeoutFactor: 0.25,
		})),
	}
	return env
}

func sv5Rules(deleteBranch bool) []types.RuleInfoInternal {
	def := `{"pullreq":{"merge_queue":{"status_checks":{"require_identifiers":["ci"]},"group_size":1,` +
		`"checks_concurrency":5,"max_check_duration_seconds":60}`
	if deleteBranch {
		def += `,"merge":{"delete_branch":true}`
	}
	def += `}}`
	return []types.RuleInfoInternal{{
		RuleInfo:   types.RuleInfo{ID: 1, Identifier: "r", Type: protection.TypeBranch, State: enum.RuleStateActive},
		RepoTarget: json.RawMessage(`{}`),
		Pattern:    json.RawMessage(`{"default":true}`),
		Definition: json.RawMessage(def),
	}}
}

// addEntry registers a pull request in the merge queue. The source SHA of the pull request is sv5SHA(10+id).
func (e *sv5Env) addEntry(id int64, state enum.MergeQueueEntryState) *types.MergeQueueEntry {
	repoID := int64(10)
	e.prs.prs[id] = &types.PullReq{
		ID: id, Number: id, State: enum.PullReqStateOpen, SubState: enum.PullReqSubStateMergeQueue,
		TargetRepoID: 10, SourceRepoID: &repoID, TargetBranch: "main", SourceBranch: fmt.Sprintf("feat-%d", id),
		SourceSHA: sv5SHA(int(10 + id)).String(),
	}
	entry := &types.MergeQueueEntry{
		PullReqID: id, MergeQueueID: 1, OrderIndex: id, State: state,
		BaseCommitSHA: sha.None, HeadCommitSHA: sha.None, MergeCommitSHA: sha.None, MergeBaseSHA: sha.None,
		ChecksCommitSHA: sha.None, MergeMethod: enum.MergeMethodMerge,
	}
	e.entries.entries[id] = entry
	return entry
}

func (e *sv5Env) process(
	t *testing.T, setup protection.MergeQueueSetup,
) error {
	t.Helper()
	ctx := context.Background()
	data, err := e.svc.createProcessCommonData(ctx, e.repo, e.queue, e.git.targetSHA)
	require.NoError(t, err)
	entries, err := e.entries.ListForMergeQueue(ctx, 1)
	require.NoError(t, err)
	return e.svc.process(ctx, data, entries, setup)
}

func TestSv5ProcessCreatesMergeCommitsAndStartsChecks(t *testing.T) {
	env := sv5NewEnv(t)
	e1 := env.addEntry(1, enum.MergeQueueEntryStateMergePending)
	e2 := env.addEntry(2, enum.MergeQueueEntryStateMergePending)

	setup := protection.MergeQueueSetup{GroupSize: 1, ChecksConcurrency: 5, MaxCheckDurationSeconds: 60}
	require.NoError(t, env.process(t, setup))

	require.Len(t, env.git.mergeCalls, 2)
	require.Equal(t, sv5SHA(1), env.git.mergeCalls[0].BaseSHA)
	require.Equal(t, sv5SHA(11), env.git.mergeCalls[0].HeadSHA)
	require.Equal(t, e1.MergeCommitSHA, env.git.mergeCalls[1].BaseSHA, "entries are chained")

	require.Equal(t, sv5SHA(1001), e1.MergeCommitSHA)
	require.Equal(t, sv5SHA(1002), e2.MergeCommitSHA)
	require.Equal(t, 2, e1.CommitCount)
	require.Equal(t, enum.MergeQueueEntryStateChecksInProgress, e1.State)
	require.Equal(t, enum.MergeQueueEntryStateChecksInProgress, e2.State)
	require.Equal(t, e2.MergeCommitSHA, e2.ChecksCommitSHA)
	require.NotNil(t, e2.ChecksDeadline)

	require.Len(t, env.git.updateRefs, 1)
	require.Equal(t, sv5SHA(1002), env.git.updateRefs[0].NewValue, "queue ref points at the last merge commit")
	require.Equal(t, 2, env.producer.count(), "checks requested for both entries")
}

func TestSv5ProcessKeepsUpToDateEntriesAndStopsStaleChecks(t *testing.T) {
	env := sv5NewEnv(t)

	// Entry 1 is up to date.
	e1 := env.addEntry(1, enum.MergeQueueEntryStateChecksInProgress)
	e1.BaseCommitSHA, e1.HeadCommitSHA, e1.MergeCommitSHA = sv5SHA(1), sv5SHA(11), sv5SHA(500)
	e1.ChecksCommitSHA = sv5SHA(500)

	// Entry 2 was a checks leader but its pull request got a new commit.
	e2 := env.addEntry(2, enum.MergeQueueEntryStateChecksInProgress)
	e2.BaseCommitSHA, e2.HeadCommitSHA, e2.MergeCommitSHA = sv5SHA(500), sv5SHA(99), sv5SHA(501)
	e2.ChecksCommitSHA = sv5SHA(501)

	// Setup values are out of range and get clamped.
	require.NoError(t, env.process(t, protection.MergeQueueSetup{GroupSize: -1, ChecksConcurrency: 1000}))

	require.Len(t, env.git.mergeCalls, 1, "only the stale entry is merged again")
	require.Equal(t, sv5SHA(500), env.git.mergeCalls[0].BaseSHA)
	require.Equal(t, sv5SHA(500), e1.MergeCommitSHA)
	require.Equal(t, sv5SHA(1001), e2.MergeCommitSHA)
	require.Equal(t, 2, env.producer.count(), "stale checks are canceled and checks of the new commit requested")
}

func TestSv5ProcessRemovesUnprocessableEntries(t *testing.T) {
	tests := []struct {
		name      string
		prepare   func(env *sv5Env)
		wantMerge int
	}{
		{"pull request left the queue", func(env *sv5Env) {
			env.prs.prs[1].SubState = enum.PullReqSubStateNone
		}, 0},
		{"draft pull request", func(env *sv5Env) { env.prs.prs[1].IsDraft = true }, 0},
		{"invalid source sha", func(env *sv5Env) { env.prs.prs[1].SourceSHA = "zz" }, 0},
		{"merge conflict", func(env *sv5Env) { env.git.conflictBy[sv5SHA(11).String()] = true }, 1},
		{"merge failure", func(env *sv5Env) { env.git.mergeErrBy[sv5SHA(11).String()] = errSv5Boom }, 1},
		{"entry update failure", func(env *sv5Env) { env.entries.errs["UpdateOptLock"] = errSv5Boom }, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := sv5NewEnv(t)
			env.addEntry(1, enum.MergeQueueEntryStateMergePending)
			env.addEntry(2, enum.MergeQueueEntryStateMergePending)
			tc.prepare(env)

			// The second entry is left alone by the failing preparation, except for the update failure.
			err := env.process(t, protection.MergeQueueSetup{GroupSize: 1, ChecksConcurrency: 1})
			require.NoError(t, err)
			require.Contains(t, env.entries.deleted, int64(1))
			require.GreaterOrEqual(t, len(env.git.mergeCalls), tc.wantMerge)
		})
	}
}

func TestSv5ProcessLimitsMergeCommitsPerRun(t *testing.T) {
	env := sv5NewEnv(t)
	for id := int64(1); id <= 12; id++ {
		env.addEntry(id, enum.MergeQueueEntryStateMergePending)
	}
	// An entry that gets cut off from the run and is in progress must be reset.
	env.entries.entries[12].State = enum.MergeQueueEntryStateChecksInProgress
	env.entries.entries[12].ChecksCommitSHA = sv5SHA(777)
	env.entries.entries[12].MergeCommitSHA = sv5SHA(778)

	require.NoError(t, env.process(t, protection.MergeQueueSetup{GroupSize: 2, ChecksConcurrency: 1}))

	require.Len(t, env.git.mergeCalls, 10)
	require.Equal(t, enum.MergeQueueEntryStateMergePending, env.entries.entries[11].State)
	require.Equal(t, enum.MergeQueueEntryStateMergePending, env.entries.entries[12].State)
	require.True(t, env.entries.entries[12].MergeCommitSHA.IsEmpty())
}

func TestSv5ProcessErrors(t *testing.T) {
	setup := protection.MergeQueueSetup{GroupSize: 1, ChecksConcurrency: 1}

	t.Run("pull request lookup", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		env.prs.errs["Find"] = errSv5Boom
		require.ErrorIs(t, env.process(t, setup), errSv5Boom)
	})

	t.Run("source repository lookup", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		fork := int64(404)
		env.prs.prs[1].SourceRepoID = &fork
		require.ErrorContains(t, env.process(t, setup), "find source repo")
	})

	t.Run("fork source repository is used for the merge input", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		fork := int64(11)
		env.prs.prs[1].SourceRepoID = &fork
		require.NoError(t, env.process(t, setup))
		require.Len(t, env.git.mergeCalls, 1)
	})

	t.Run("entries cannot be stored", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		env.entries.errs["Update"] = errSv5Boom
		require.ErrorContains(t, env.process(t, setup), "update merge queue entries")
	})

	t.Run("ref update failure is tolerated", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		env.git.updateErr = errSv5Boom
		require.NoError(t, env.process(t, setup))
	})
}

func TestSv5Reprocess(t *testing.T) {
	ctx := context.Background()
	setup := protection.MergeQueueSetup{GroupSize: 1, ChecksConcurrency: 1}

	t.Run("processes the queue", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		require.NoError(t, env.svc.reprocess(ctx, env.repo, env.queue, setup))
		require.Len(t, env.git.mergeCalls, 1)
	})

	t.Run("target branch deleted clears the queue", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateChecksInProgress)
		env.entries.entries[1].ChecksCommitSHA = sv5SHA(5)
		env.git.getRefErr = gitnesserrors.NotFound("branch")

		require.NoError(t, env.svc.reprocess(ctx, env.repo, env.queue, setup))
		require.Equal(t, []int64{1}, env.mqs.deleted)
		require.Equal(t, enum.PullReqSubStateNone, env.prs.prs[1].SubState)
		require.Len(t, env.git.updateRefs, 1, "merge queue reference is deleted")
		require.True(t, env.git.updateRefs[0].NewValue.IsNil())
		require.Len(t, env.activity.payloads, 1)
	})

	t.Run("clearing failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.git.getRefErr = gitnesserrors.NotFound("branch")
		env.mqs.errs["Delete"] = errSv5Boom
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		require.ErrorContains(t, env.svc.reprocess(ctx, env.repo, env.queue, setup), "was deleted")
	})

	t.Run("head lookup failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.git.getRefErr = errSv5Boom
		require.ErrorContains(t, env.svc.reprocess(ctx, env.repo, env.queue, setup), "last commit SHA")
	})

	t.Run("entry listing failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.entries.errs["ListForMergeQueue"] = errSv5Boom
		require.ErrorContains(t, env.svc.reprocess(ctx, env.repo, env.queue, setup), "list merge queue entries")
	})
}

func TestSv5RemoveAll(t *testing.T) {
	ctx := context.Background()
	payload := types.PullRequestActivityPayloadMergeQueueRemove{Reason: enum.MergeQueueRemovalReasonError}

	t.Run("no queue", func(t *testing.T) {
		env := sv5NewEnv(t)
		require.NoError(t, env.svc.RemoveAll(ctx, env.repo, "other", payload))
		require.Empty(t, env.mqs.deleted)
	})

	t.Run("clears the queue and aborts checks", func(t *testing.T) {
		env := sv5NewEnv(t)
		e1 := env.addEntry(1, enum.MergeQueueEntryStateChecksInProgress)
		e1.ChecksCommitSHA = sv5SHA(9)
		env.addEntry(2, enum.MergeQueueEntryStateMergePending)
		env.addEntry(3, enum.MergeQueueEntryStateMergePending)
		env.prs.prs[3].SubState = enum.PullReqSubStateNone // already released, is skipped
		env.activity.err = errSv5Boom

		require.NoError(t, env.svc.RemoveAll(ctx, env.repo, "main", payload))
		require.Equal(t, []int64{1}, env.mqs.deleted)
		require.Equal(t, []int64{1, 2}, env.prs.updated)
		require.Len(t, env.activity.payloads, 2)
		require.Equal(t, 1, env.producer.count(), "one checks-canceled event")
		require.Len(t, env.git.updateRefs, 1)
	})

	failing := map[string]struct {
		errKey string
		msg    string
	}{
		"find queue":   {"FindByRepoAndBranch", "find merge queue"},
		"list entries": {"ListForMergeQueue", "list merge queue entries"},
		"delete all":   {"DeleteAll", "delete all merge queue entries"},
		"delete queue": {"Delete", "delete merge queue"},
		"find pr":      {"Find", "find pull request"},
		"update pr":    {"Update", "update pull request"},
	}
	for name, tc := range failing {
		t.Run("error "+name, func(t *testing.T) {
			env := sv5NewEnv(t)
			env.addEntry(1, enum.MergeQueueEntryStateMergePending)
			switch tc.errKey {
			case "Find", "Update":
				env.prs.errs[tc.errKey] = errSv5Boom
			case "FindByRepoAndBranch", "Delete":
				env.mqs.errs[tc.errKey] = errSv5Boom
			default:
				env.entries.errs[tc.errKey] = errSv5Boom
			}
			require.ErrorContains(t, env.svc.RemoveAll(ctx, env.repo, "main", payload), tc.msg)
		})
	}
}

func TestSv5Remove(t *testing.T) {
	ctx := context.Background()
	payload := types.PullRequestActivityPayloadMergeQueueRemove{Reason: enum.MergeQueueRemovalReasonManual}

	t.Run("not in queue", func(t *testing.T) {
		env := sv5NewEnv(t)
		require.ErrorIs(t, env.svc.Remove(ctx, 1, payload), ErrNotInQueue)
	})

	t.Run("entry lookup failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.entries.errs["Find"] = errSv5Boom
		require.ErrorContains(t, env.svc.Remove(ctx, 1, payload), "find merge queue entry")
	})

	t.Run("queue lookup failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		env.entries.entries[1].MergeQueueID = 42
		require.ErrorContains(t, env.svc.Remove(ctx, 1, payload), "find merge queue")
	})

	t.Run("removes the entry and reports the update", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		require.NoError(t, env.svc.Remove(ctx, 1, payload))
		require.Equal(t, []int64{1}, env.entries.deleted)
		require.Equal(t, 1, env.producer.count())
		require.Len(t, env.activity.payloads, 1)
	})
}

func TestSv5RemoveRebuildsChains(t *testing.T) {
	ctx := context.Background()
	payload := types.PullRequestActivityPayloadMergeQueueRemove{Reason: enum.MergeQueueRemovalReasonManual}
	env := sv5NewEnv(t)

	group := env.addEntry(1, enum.MergeQueueEntryStateMergeGroup)
	group.ChecksCommitSHA = sv5SHA(50)
	leader := env.addEntry(2, enum.MergeQueueEntryStateChecksInProgress)
	leader.ChecksCommitSHA = sv5SHA(50)
	started := int64(5)
	leader.ChecksStarted = &started
	next := env.addEntry(3, enum.MergeQueueEntryStateChecksInProgress)
	next.MergeCommitSHA = sv5SHA(60)
	next.ChecksCommitSHA = sv5SHA(60)
	pending := env.addEntry(4, enum.MergeQueueEntryStateMergePending)

	require.NoError(t, env.svc.remove(ctx, 2, payload))

	require.Equal(t, enum.MergeQueueEntryStateChecksPending, group.State, "orphaned group member is re-queued")
	require.True(t, group.ChecksCommitSHA.IsEmpty())
	require.Equal(t, enum.MergeQueueEntryStateMergePending, next.State, "later entries are reset")
	require.Equal(t, enum.MergeQueueEntryStateMergePending, pending.State)
	require.Equal(t, 2, env.producer.count(), "checks of the removed leader and the reset entry are canceled")
	require.Equal(t, enum.PullReqSubStateNone, env.prs.prs[2].SubState)
}

func TestSv5RemoveErrors(t *testing.T) {
	ctx := context.Background()
	payload := types.PullRequestActivityPayloadMergeQueueRemove{Reason: enum.MergeQueueRemovalReasonManual}

	t.Run("queue missing", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		env.prs.prs[1].TargetBranch = "other"
		require.ErrorIs(t, env.svc.remove(ctx, 1, payload), ErrNotInQueue)
	})

	t.Run("not an entry of the queue", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		env.addEntry(2, enum.MergeQueueEntryStateMergePending)
		delete(env.entries.entries, 2)
		require.ErrorIs(t, env.svc.remove(ctx, 2, payload), ErrNotInQueue)
	})

	t.Run("merged pull request gets no activity", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.addEntry(1, enum.MergeQueueEntryStateMergePending)
		env.prs.prs[1].State = enum.PullReqStateMerged
		require.NoError(t, env.svc.remove(ctx, 1, payload))
		require.Empty(t, env.activity.payloads)
	})

	failing := map[string]string{
		"Find": "find pull request", "FindByRepoAndBranch": "find merge queue",
		"ListForMergeQueue": "list merge queue entries", "Delete": "delete merge queue entry",
		"Update": "update merge queue entry", "PRUpdate": "update pull request",
	}
	for key, msg := range failing {
		t.Run("error "+key, func(t *testing.T) {
			env := sv5NewEnv(t)
			env.addEntry(1, enum.MergeQueueEntryStateMergePending)
			env.addEntry(2, enum.MergeQueueEntryStateChecksPending)
			switch key {
			case "Find":
				env.prs.errs["Find"] = errSv5Boom
			case "PRUpdate":
				env.prs.errs["Update"] = errSv5Boom
			case "FindByRepoAndBranch":
				env.mqs.errs[key] = errSv5Boom
			default:
				env.entries.errs[key] = errSv5Boom
			}
			require.ErrorContains(t, env.svc.remove(ctx, 1, payload), msg)
		})
	}
}

func TestSv5ResetAndFindEntriesToMerge(t *testing.T) {
	env := sv5NewEnv(t)
	e1 := env.addEntry(1, enum.MergeQueueEntryStateChecksInProgress)
	e1.ChecksCommitSHA = sv5SHA(5)
	env.entries.errs["UpdateOptLock"] = errSv5Boom
	env.svc.resetEntries(context.Background(), env.queue, []*types.MergeQueueEntry{e1})
	require.Equal(t, 1, env.producer.count())

	_, err := env.svc.findEntriesToMerge(e1, nil)
	require.ErrorIs(t, err, errIncompleteMergeQueue)
}

func (e *sv5Env) readyChain() (*types.MergeQueueEntry, *types.MergeQueueEntry) {
	e1 := e.addEntry(1, enum.MergeQueueEntryStateMergeGroup)
	e1.BaseCommitSHA, e1.HeadCommitSHA, e1.MergeCommitSHA = sv5SHA(1), sv5SHA(11), sv5SHA(101)
	e1.MergeBaseSHA = sv5SHA(2000)
	e2 := e.addEntry(2, enum.MergeQueueEntryStateChecksInProgress)
	e2.BaseCommitSHA, e2.HeadCommitSHA, e2.MergeCommitSHA = sv5SHA(101), sv5SHA(12), sv5SHA(102)
	e2.ChecksCommitSHA = sv5SHA(102)
	return e1, e2
}

func TestSv5FastForward(t *testing.T) {
	ctx := context.Background()

	t.Run("merges all entries up to the passed one", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.rules.rules = sv5Rules(true)
		_, e2 := env.readyChain()
		e2.DeleteSourceBranch = true
		env.addEntry(3, enum.MergeQueueEntryStateMergePending) // after the passed entry: stays queued

		require.NoError(t, env.svc.fastForward(ctx, env.queue, e2))

		require.Len(t, env.git.refsCalls, 1)
		refs := env.git.refsCalls[0].Refs
		require.Len(t, refs, 5)
		require.Equal(t, "refs/heads/main", refs[0].Name)
		require.Equal(t, sv5SHA(102), refs[0].New)
		require.Equal(t, sv5SHA(1), refs[0].Old)

		require.Equal(t, []int64{1, 2}, env.entries.deleted)
		for _, id := range []int64{1, 2} {
			require.NotNil(t, env.prs.prs[id].Merged)
			require.Equal(t, enum.PullReqStateMerged, env.prs.prs[id].State)
		}
		require.Nil(t, env.prs.prs[3].Merged)
		require.ElementsMatch(t, []string{"feat-1", "feat-2"}, env.git.deleted, "rule requires branch removal")
		require.GreaterOrEqual(t, env.producer.count(), 3, "branch updated and merge events")
	})

	t.Run("keeps the branch unless asked or required", func(t *testing.T) {
		env := sv5NewEnv(t)
		_, e2 := env.readyChain()
		env.entries.entries[1].DeleteSourceBranch = false
		e2.DeleteSourceBranch = true

		require.NoError(t, env.svc.fastForward(ctx, env.queue, e2))
		require.Equal(t, []string{"feat-2"}, env.git.deleted)
	})

	t.Run("source branch of a fork is kept", func(t *testing.T) {
		env := sv5NewEnv(t)
		_, e2 := env.readyChain()
		e2.DeleteSourceBranch = true
		fork := int64(11)
		env.prs.prs[2].SourceRepoID = &fork

		require.NoError(t, env.svc.fastForward(ctx, env.queue, e2))
		require.Empty(t, env.git.deleted)
	})

	t.Run("repo lookup failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		_, e2 := env.readyChain()
		q := *env.queue
		q.RepoID = 404
		require.ErrorContains(t, env.svc.fastForward(ctx, &q, e2), "find repo")
	})

	t.Run("entry listing failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		_, e2 := env.readyChain()
		env.entries.errs["ListForMergeQueue"] = errSv5Boom
		require.ErrorContains(t, env.svc.fastForward(ctx, env.queue, e2), "list merge queue entries")
	})

	t.Run("incomplete queue", func(t *testing.T) {
		env := sv5NewEnv(t)
		e1, e2 := env.readyChain()
		e1.BaseCommitSHA = sha.None
		require.ErrorIs(t, env.svc.fastForward(ctx, env.queue, e2), errIncompleteMergeQueue)
	})

	t.Run("pull request changed since the merge commit was created", func(t *testing.T) {
		env := sv5NewEnv(t)
		_, e2 := env.readyChain()
		env.prs.prs[1].SourceSHA = sv5SHA(99).String()
		err := env.svc.fastForward(ctx, env.queue, e2)
		require.ErrorIs(t, err, errIncompleteMergeQueue)
		require.False(t, isFastForwardError(err))
	})

	t.Run("pull request lookup failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		_, e2 := env.readyChain()
		env.prs.errs["Find"] = errSv5Boom
		require.ErrorContains(t, env.svc.fastForward(ctx, env.queue, e2), "find pull request")
	})

	t.Run("target branch moved", func(t *testing.T) {
		env := sv5NewEnv(t)
		_, e2 := env.readyChain()
		env.git.refsErr = gitnesserrors.PreconditionFailed("moved")
		err := env.svc.fastForward(ctx, env.queue, e2)
		require.Error(t, err)
		require.True(t, isFastForwardError(err))
	})

	t.Run("reference update failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		_, e2 := env.readyChain()
		env.git.refsErr = errSv5Boom
		err := env.svc.fastForward(ctx, env.queue, e2)
		require.ErrorIs(t, err, errSv5Boom)
		require.False(t, isFastForwardError(err))
	})
}

func TestSv5HandlerCheckFinishedMergesQueue(t *testing.T) {
	ctx := context.Background()
	newEvent := func() *events.Event[*checkevents.ReportedPayload] {
		return &events.Event[*checkevents.ReportedPayload]{Payload: &checkevents.ReportedPayload{
			Base:       checkevents.Base{RepoID: 10, SHA: sv5SHA(102).String()},
			Identifier: "ci", Status: enum.CheckStatusSuccess,
		}}
	}
	// The entries store looks up by merge commit; wrap it.
	setup := func(t *testing.T) *sv5Env {
		env := sv5NewEnv(t)
		env.rules.rules = sv5Rules(false)
		env.readyChain()
		env.svc.mergeQueueEntryStore = &sv5MergeCommitStore{sv5EntryStore: env.entries}
		return env
	}

	t.Run("required checks done: fast-forward and reprocess", func(t *testing.T) {
		env := setup(t)
		env.checks.checks = []types.Check{{Identifier: "ci", Status: enum.CheckStatusSuccess}}
		require.NoError(t, env.svc.handlerCheckFinished(ctx, newEvent()))
		require.Len(t, env.git.refsCalls, 1)
		require.Equal(t, []int64{1, 2}, env.entries.deleted)
	})

	t.Run("required checks missing: waits", func(t *testing.T) {
		env := setup(t)
		require.NoError(t, env.svc.handlerCheckFinished(ctx, newEvent()))
		require.Empty(t, env.git.refsCalls)
	})

	t.Run("target moved: reset and reprocess", func(t *testing.T) {
		env := setup(t)
		env.checks.checks = []types.Check{{Identifier: "ci", Status: enum.CheckStatusSuccess}}
		env.git.refsErr = gitnesserrors.PreconditionFailed("moved")
		require.NoError(t, env.svc.handlerCheckFinished(ctx, newEvent()))
		require.NotEmpty(t, env.git.updateRefs, "merge queue reference is reset")
	})

	t.Run("fast-forward failure", func(t *testing.T) {
		env := setup(t)
		env.checks.checks = []types.Check{{Identifier: "ci", Status: enum.CheckStatusSuccess}}
		env.git.refsErr = errSv5Boom
		require.ErrorIs(t, env.svc.handlerCheckFinished(ctx, newEvent()), errSv5Boom)
	})
}

type sv5MergeCommitStore struct {
	*sv5EntryStore
}

func (s *sv5MergeCommitStore) FindByMergeCommit(_ context.Context, commit sha.SHA) (*types.MergeQueueEntry, error) {
	for _, e := range s.entries {
		if e.MergeCommitSHA.Equal(commit) {
			return e, nil
		}
	}
	return nil, gitnessstore.ErrResourceNotFound
}
