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

	checkevents "github.com/harness/gitness/app/events/check"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	svcSHAOne   = "0000000000000000000000000000000000000001"
	svcSHATwo   = "0000000000000000000000000000000000000002"
	svcSHAThree = "0000000000000000000000000000000000000003"
)

type svcTx struct{}

func (svcTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

// svcProducer records the events sent through the reporters.
type svcProducer struct {
	mu   sync.Mutex
	sent []string
}

func (p *svcProducer) Send(_ context.Context, streamID string, payload map[string]any) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = append(p.sent, streamID+":"+payloadType(payload))
	return "1-0", nil
}

func payloadType(payload map[string]any) string {
	t, _ := payload["type"].(string)
	return t
}

func (p *svcProducer) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.sent)
}

type svcMQStore struct {
	store.MergeQueueStore
	queues    map[int64]*types.MergeQueue
	nextID    int64
	findErr   error
	createErr error
	updateErr error
}

func (s *svcMQStore) add(q *types.MergeQueue) *types.MergeQueue {
	s.nextID++
	q.ID = s.nextID
	s.queues[q.ID] = q
	return q
}

func (s *svcMQStore) Find(_ context.Context, id int64) (*types.MergeQueue, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	q, ok := s.queues[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return q, nil
}

func (s *svcMQStore) FindByRepoAndBranch(_ context.Context, repoID int64, branch string) (*types.MergeQueue, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	for _, q := range s.queues {
		if q.RepoID == repoID && q.Branch == branch {
			return q, nil
		}
	}
	return nil, gitnessstore.ErrResourceNotFound
}

func (s *svcMQStore) Create(_ context.Context, q *types.MergeQueue) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.add(q)
	return nil
}

func (s *svcMQStore) Update(context.Context, *types.MergeQueue) error { return s.updateErr }

func (s *svcMQStore) UpdateOptLock(
	_ context.Context, q *types.MergeQueue, fn func(*types.MergeQueue) error,
) (*types.MergeQueue, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	if err := fn(q); err != nil {
		return nil, err
	}
	return q, nil
}

type svcEntryStore struct {
	store.MergeQueueEntryStore
	entries     map[int64]*types.MergeQueueEntry // by PR ID
	branches    map[string]struct{}
	branchesErr error
	count       int64
	listErr     error
	updated     []int64
	deleted     []int64
	byMergeSHA  map[sha.SHA]*types.MergeQueueEntry
	findMergeEr error
	overdue     []*types.MergeQueueEntry
	overdueErr  error
}

func newSvcEntryStore() *svcEntryStore {
	return &svcEntryStore{entries: map[int64]*types.MergeQueueEntry{}}
}

func (s *svcEntryStore) Find(_ context.Context, pullReqID int64) (*types.MergeQueueEntry, error) {
	e, ok := s.entries[pullReqID]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return e, nil
}

func (s *svcEntryStore) FindByMergeCommit(_ context.Context, commit sha.SHA) (*types.MergeQueueEntry, error) {
	if s.findMergeEr != nil {
		return nil, s.findMergeEr
	}
	e, ok := s.byMergeSHA[commit]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return e, nil
}

func (s *svcEntryStore) ListForMergeQueue(_ context.Context, queueID int64) ([]*types.MergeQueueEntry, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	var res []*types.MergeQueueEntry
	for _, e := range s.entries {
		if e.MergeQueueID == queueID {
			res = append(res, e)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].OrderIndex < res[j].OrderIndex })
	return res, nil
}

func (s *svcEntryStore) CountForRepoAndBranch(context.Context, int64, string) (int64, error) {
	return s.count, nil
}

func (s *svcEntryStore) Create(_ context.Context, e *types.MergeQueueEntry) error {
	s.entries[e.PullReqID] = e
	return nil
}

func (s *svcEntryStore) Update(_ context.Context, e *types.MergeQueueEntry) error {
	s.updated = append(s.updated, e.PullReqID)
	return nil
}

func (s *svcEntryStore) Delete(_ context.Context, pullReqID int64) error {
	s.deleted = append(s.deleted, pullReqID)
	delete(s.entries, pullReqID)
	return nil
}

func (s *svcEntryStore) BranchesWithPullReqInQueue(context.Context, int64, []string) (map[string]struct{}, error) {
	return s.branches, s.branchesErr
}

func (s *svcEntryStore) ListOverdueChecks(context.Context, int64) ([]*types.MergeQueueEntry, error) {
	return s.overdue, s.overdueErr
}

type svcPRStore struct {
	store.PullReqStore
	prs     map[int64]*types.PullReq
	updated []int64
}

func (s *svcPRStore) Find(_ context.Context, id int64) (*types.PullReq, error) {
	pr, ok := s.prs[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return pr, nil
}

func (s *svcPRStore) Update(_ context.Context, pr *types.PullReq) error {
	s.updated = append(s.updated, pr.ID)
	return nil
}

type svcActivityStore struct {
	store.PullReqActivityStore
	payloads []types.PullReqActivityPayload
	err      error
}

func (s *svcActivityStore) CreateWithPayload(
	_ context.Context, _ *types.PullReq, _ int64, payload types.PullReqActivityPayload, _ *types.PullReqActivityMetadata,
) (*types.PullReqActivity, error) {
	s.payloads = append(s.payloads, payload)
	return &types.PullReqActivity{}, s.err
}

type svcCheckStore struct {
	store.CheckStore
	checks  []types.Check
	listErr error
	byIdent types.Check
	identEr error
}

func (s *svcCheckStore) List(context.Context, int64, string, types.CheckListOptions) ([]types.Check, error) {
	return s.checks, s.listErr
}

func (s *svcCheckStore) FindByIdentifier(context.Context, int64, string, string) (types.Check, error) {
	return s.byIdent, s.identEr
}

type svcRuleStore struct {
	store.RuleStore
	rules []types.RuleInfoInternal
	err   error
}

func (s *svcRuleStore) ListOnlyRepoRules(
	context.Context, *types.RepositoryCore, ...enum.RuleType,
) ([]types.RuleInfoInternal, error) {
	return s.rules, s.err
}

type svcRepoCache struct {
	repos map[int64]*types.RepositoryCore
}

func (c svcRepoCache) Stats() (int64, int64) { return 0, 0 }
func (c svcRepoCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	r, ok := c.repos[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return r, nil
}
func (c svcRepoCache) Evict(context.Context, int64) {}

type svcGit struct {
	git.Interface
	ref    sha.SHA
	refErr error
}

func (g *svcGit) GetRef(context.Context, git.GetRefParams) (git.GetRefResponse, error) {
	return git.GetRefResponse{SHA: g.ref}, g.refErr
}

type svcEnv struct {
	svc       *Service
	queues    *svcMQStore
	entries   *svcEntryStore
	prs       *svcPRStore
	activity  *svcActivityStore
	checks    *svcCheckStore
	rules     *svcRuleStore
	git       *svcGit
	producer  *svcProducer
	repo      *types.RepositoryCore
	repoCache svcRepoCache
}

func svcNewEnv(t *testing.T) *svcEnv {
	t.Helper()

	producer := &svcProducer{}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errors.New("unsupported") },
		producer,
		events.NewNoopCollector(),
	)
	require.NoError(t, err)
	reporter, err := mergequeueevents.NewReporter(system)
	require.NoError(t, err)

	repo := &types.RepositoryCore{ID: 1, Path: "root/repo", GitUID: "uid1", DefaultBranch: "main"}
	env := &svcEnv{
		queues:    &svcMQStore{queues: map[int64]*types.MergeQueue{}},
		entries:   newSvcEntryStore(),
		prs:       &svcPRStore{prs: map[int64]*types.PullReq{}},
		activity:  &svcActivityStore{},
		checks:    &svcCheckStore{},
		rules:     &svcRuleStore{},
		git:       &svcGit{},
		producer:  producer,
		repo:      repo,
		repoCache: svcRepoCache{repos: map[int64]*types.RepositoryCore{1: repo}},
	}

	manager := protection.NewManager(env.rules)
	require.NoError(t, manager.Register(protection.TypeBranch,
		func() protection.Definition { return &protection.Branch{} }))

	env.svc = &Service{
		git:                     env.git,
		tx:                      svcTx{},
		mergeQueueEventReporter: reporter,
		repoFinder:              refcache.NewRepoFinder(nil, nil, env.repoCache, nil, cache.Evictor[*types.RepositoryCore]{}),
		pullreqStore:            env.prs,
		activityStore:           env.activity,
		checkStore:              env.checks,
		mergeQueueStore:         env.queues,
		mergeQueueEntryStore:    env.entries,
		protectionManager:       manager,
	}
	return env
}

// activeRules returns a branch rule that defines an active merge queue.
func svcActiveRules() []types.RuleInfoInternal {
	return []types.RuleInfoInternal{{
		RuleInfo:   types.RuleInfo{ID: 1, Type: protection.TypeBranch, State: enum.RuleStateActive},
		RepoTarget: json.RawMessage(`{"include":{},"exclude":{}}`),
		Pattern:    json.RawMessage(`{}`),
		Definition: json.RawMessage(`{"pullreq":{"merge_queue":{"status_checks":{"require_identifiers":["ci","lint"]},` +
			`"group_size":2,"checks_concurrency":2,"max_check_duration_seconds":60}}}`),
	}}
}

func svcEntry(prID, queueID, order int64, state enum.MergeQueueEntryState) *types.MergeQueueEntry {
	return &types.MergeQueueEntry{
		PullReqID: prID, MergeQueueID: queueID, OrderIndex: order, State: state,
		BaseCommitSHA: sha.None, HeadCommitSHA: sha.None, MergeCommitSHA: sha.None,
		MergeBaseSHA: sha.None, ChecksCommitSHA: sha.None,
	}
}

func TestSvcViolation(t *testing.T) {
	v := Violation("main")
	assert.Equal(t, RuleIdentifier, v.Rule.Identifier)
	assert.Equal(t, enum.RuleStateActive, v.Rule.State)
	require.Len(t, v.Violations, 1)
	assert.Equal(t, RuleIdentifier, v.Violations[0].Code)
	assert.Contains(t, v.Violations[0].Message, `"main"`)
	assert.True(t, v.IsCritical())
}

func TestSvcBranchInQueue(t *testing.T) {
	ctx := context.Background()
	env := svcNewEnv(t)

	env.entries.branches = map[string]struct{}{"main": {}}
	inQueue, err := env.svc.IsBranchInQueue(ctx, 1, "main")
	require.NoError(t, err)
	assert.True(t, inQueue)

	violations, err := env.svc.BranchInQueueViolations(ctx, 1, "main")
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, RuleIdentifier, violations[0].Rule.Identifier)

	got, err := env.svc.BranchesWithPullReqInQueue(ctx, 1, []string{"main", "dev"})
	require.NoError(t, err)
	assert.Equal(t, env.entries.branches, got)

	env.entries.branches = nil
	inQueue, err = env.svc.IsBranchInQueue(ctx, 1, "dev")
	require.NoError(t, err)
	assert.False(t, inQueue)
	violations, err = env.svc.BranchInQueueViolations(ctx, 1, "dev")
	require.NoError(t, err)
	assert.Empty(t, violations)

	env.entries.branchesErr = errors.New("db down")
	_, err = env.svc.IsBranchInQueue(ctx, 1, "main")
	require.Error(t, err)
	_, err = env.svc.BranchInQueueViolations(ctx, 1, "main")
	require.Error(t, err)
	_, err = env.svc.BranchesWithPullReqInQueue(ctx, 1, nil)
	require.Error(t, err)
}

func TestSvcFindOrCreateMergeQueue(t *testing.T) {
	ctx := context.Background()

	t.Run("creates a missing queue", func(t *testing.T) {
		env := svcNewEnv(t)
		q, err := env.svc.FindOrCreateMergeQueue(ctx, 1, "main")
		require.NoError(t, err)
		assert.Equal(t, "main", q.Branch)
		assert.NotZero(t, q.ID)
		assert.NotZero(t, q.Created)
	})
	t.Run("returns an existing queue", func(t *testing.T) {
		env := svcNewEnv(t)
		existing := env.queues.add(&types.MergeQueue{RepoID: 1, Branch: "main"})
		q, err := env.svc.FindOrCreateMergeQueue(ctx, 1, "main")
		require.NoError(t, err)
		assert.Same(t, existing, q)
	})
	t.Run("lookup failure", func(t *testing.T) {
		env := svcNewEnv(t)
		env.queues.findErr = errors.New("db down")
		_, err := env.svc.FindOrCreateMergeQueue(ctx, 1, "main")
		require.Error(t, err)
	})
	t.Run("create failure", func(t *testing.T) {
		env := svcNewEnv(t)
		env.queues.createErr = errors.New("db down")
		_, err := env.svc.FindOrCreateMergeQueue(ctx, 1, "main")
		require.Error(t, err)
	})
	t.Run("concurrent creation conflict falls back to find", func(t *testing.T) {
		env := svcNewEnv(t)
		env.queues.createErr = gitnesserrors.Conflict("exists")
		_, err := env.svc.FindOrCreateMergeQueue(ctx, 1, "main")
		require.Error(t, err, "the queue still does not exist in the fake")
	})
}

func TestSvcGetLastCommitSHA(t *testing.T) {
	env := svcNewEnv(t)
	env.git.ref = sha.Must(svcSHAOne)
	got, err := env.svc.getLastCommitSHA(context.Background(), env.repo, "main")
	require.NoError(t, err)
	assert.Equal(t, svcSHAOne, got.String())

	env.git.refErr = errors.New("git down")
	_, err = env.svc.getLastCommitSHA(context.Background(), env.repo, "main")
	require.Error(t, err)
}

func TestSvcBuildMergeQueueInfo(t *testing.T) {
	ctx := context.Background()
	setup := protection.MergeQueueSetup{RequiredChecks: []string{"ci", "lint"}}

	env := svcNewEnv(t)
	q := env.queues.add(&types.MergeQueue{RepoID: 1, Branch: "main"})
	env.entries.entries[10] = svcEntry(10, q.ID, 1, enum.MergeQueueEntryStateMergePending)
	leader := svcEntry(11, q.ID, 2, enum.MergeQueueEntryStateChecksInProgress)
	leader.ChecksCommitSHA = sha.Must(svcSHAOne)
	leader.MergeMethod = enum.MergeMethodSquash
	env.entries.entries[11] = leader
	env.checks.checks = []types.Check{
		{Identifier: "ci", Status: enum.CheckStatusSuccess},
		{Identifier: "extra", Status: enum.CheckStatusRunning},
	}

	t.Run("pull request without checks commit lists required checks as pending", func(t *testing.T) {
		info, err := env.svc.BuildMergeQueueInfo(ctx, env.repo, env.entries.entries[10], setup)
		require.NoError(t, err)
		assert.Equal(t, 0, info.PullRequestsAhead)
		require.Len(t, info.Checks, 2)
		assert.Equal(t, "ci", info.Checks[0].Check.Identifier)
		assert.True(t, info.Checks[0].Required)
		assert.Equal(t, enum.CheckStatusPending, info.Checks[0].Check.Status)
		assert.Equal(t, "lint", info.Checks[1].Check.Identifier)
	})
	t.Run("leader reports real checks plus missing required ones", func(t *testing.T) {
		info, err := env.svc.BuildMergeQueueInfo(ctx, env.repo, leader, setup)
		require.NoError(t, err)
		assert.Equal(t, 1, info.PullRequestsAhead)
		assert.Equal(t, enum.MergeQueueEntryStateChecksInProgress, info.State)
		assert.Equal(t, enum.MergeMethodSquash, info.MergeMethod)

		byIdent := map[string]types.PullReqCheck{}
		for _, c := range info.Checks {
			byIdent[c.Check.Identifier] = c
		}
		require.Len(t, byIdent, 3)
		assert.True(t, byIdent["ci"].Required)
		assert.Equal(t, enum.CheckStatusSuccess, byIdent["ci"].Check.Status)
		assert.False(t, byIdent["extra"].Required)
		assert.True(t, byIdent["lint"].Required)
		assert.Equal(t, enum.CheckStatusPending, byIdent["lint"].Check.Status)
		assert.Equal(t, svcSHAOne, byIdent["lint"].Check.CommitSHA)
	})
	t.Run("pull request that is not queued", func(t *testing.T) {
		missing := svcEntry(99, q.ID, 3, enum.MergeQueueEntryStateMergePending)
		_, err := env.svc.BuildMergeQueueInfo(ctx, env.repo, missing, setup)
		require.Error(t, err)
	})
	t.Run("store failures", func(t *testing.T) {
		env.entries.listErr = errors.New("db down")
		_, err := env.svc.BuildMergeQueueInfo(ctx, env.repo, leader, setup)
		require.Error(t, err)
		env.entries.listErr = nil

		env.checks.listErr = errors.New("db down")
		_, err = env.svc.BuildMergeQueueInfo(ctx, env.repo, leader, setup)
		require.Error(t, err)
		env.checks.listErr = nil
	})
}

func TestSvcListMergeQueue(t *testing.T) {
	ctx := context.Background()
	setup := protection.MergeQueueSetup{RequiredChecks: []string{"ci"}}

	t.Run("no queue yields an empty list", func(t *testing.T) {
		env := svcNewEnv(t)
		items, err := env.svc.ListMergeQueue(ctx, env.repo, "main", setup)
		require.NoError(t, err)
		assert.Empty(t, items)
		assert.NotNil(t, items)
	})
	t.Run("lists entries in order with their pull requests", func(t *testing.T) {
		env := svcNewEnv(t)
		q := env.queues.add(&types.MergeQueue{RepoID: 1, Branch: "main"})
		env.entries.entries[10] = svcEntry(10, q.ID, 2, enum.MergeQueueEntryStateMergePending)
		env.entries.entries[11] = svcEntry(11, q.ID, 1, enum.MergeQueueEntryStateMergePending)
		env.prs.prs[10] = &types.PullReq{ID: 10, Number: 100}
		env.prs.prs[11] = &types.PullReq{ID: 11, Number: 101}

		items, err := env.svc.ListMergeQueue(ctx, env.repo, "main", setup)
		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, int64(101), items[0].PullReq.Number)
		assert.Equal(t, 0, items[0].MergeQueueInfo.PullRequestsAhead)
		assert.Equal(t, int64(100), items[1].PullReq.Number)
		assert.Equal(t, 1, items[1].MergeQueueInfo.PullRequestsAhead)
	})
	t.Run("failures", func(t *testing.T) {
		env := svcNewEnv(t)
		env.queues.findErr = errors.New("db down")
		_, err := env.svc.ListMergeQueue(ctx, env.repo, "main", setup)
		require.Error(t, err)

		env = svcNewEnv(t)
		env.queues.add(&types.MergeQueue{RepoID: 1, Branch: "main"})
		env.entries.listErr = errors.New("db down")
		_, err = env.svc.ListMergeQueue(ctx, env.repo, "main", setup)
		require.Error(t, err)

		env = svcNewEnv(t)
		q := env.queues.add(&types.MergeQueue{RepoID: 1, Branch: "main"})
		env.entries.entries[10] = svcEntry(10, q.ID, 1, enum.MergeQueueEntryStateMergePending)
		_, err = env.svc.ListMergeQueue(ctx, env.repo, "main", setup)
		require.Error(t, err, "the pull request of an entry is missing")
	})
}

func TestSvcListChecks(t *testing.T) {
	env := svcNewEnv(t)
	env.checks.checks = []types.Check{{Identifier: "ci"}}
	got, err := env.svc.ListChecks(context.Background(), 1, sha.Must(svcSHAOne))
	require.NoError(t, err)
	assert.Len(t, got, 1)

	env.checks.listErr = errors.New("db down")
	_, err = env.svc.ListChecks(context.Background(), 1, sha.Must(svcSHAOne))
	require.Error(t, err)
}

func TestSvcEnqueue(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T) (*svcEnv, *types.PullReq) {
		env := svcNewEnv(t)
		pr := &types.PullReq{ID: 10, Number: 1, State: enum.PullReqStateOpen, TargetBranch: "main", TargetRepoID: 1}
		env.prs.prs[10] = pr
		return env, pr
	}

	t.Run("success", func(t *testing.T) {
		env, pr := setup(t)
		gotPR, entry, err := env.svc.Enqueue(ctx, pr, env.repo, 5, enum.MergeMethodSquash, "title", "msg", true)
		require.NoError(t, err)
		assert.Equal(t, enum.PullReqSubStateMergeQueue, gotPR.SubState)
		assert.Equal(t, enum.MergeQueueEntryStateMergePending, entry.State)
		assert.Equal(t, enum.MergeMethodSquash, entry.MergeMethod)
		assert.Equal(t, "title", entry.CommitTitle)
		assert.True(t, entry.DeleteSourceBranch)
		assert.Equal(t, int64(5), entry.CreatedBy)
		assert.Equal(t, int64(1), entry.OrderIndex)
		assert.True(t, entry.MergeCommitSHA.IsEmpty())
		assert.Contains(t, env.entries.entries, int64(10))
		assert.Equal(t, []int64{10}, env.prs.updated)
		require.Len(t, env.activity.payloads, 1)
		assert.Equal(t, 1, env.producer.count(), "queue updated event is published")
	})
	t.Run("sequence numbers increase", func(t *testing.T) {
		env, pr := setup(t)
		_, first, err := env.svc.Enqueue(ctx, pr, env.repo, 5, enum.MergeMethodMerge, "", "", false)
		require.NoError(t, err)

		second := &types.PullReq{ID: 11, State: enum.PullReqStateOpen, TargetBranch: "main", TargetRepoID: 1}
		env.prs.prs[11] = second
		_, secondEntry, err := env.svc.Enqueue(ctx, second, env.repo, 5, enum.MergeMethodMerge, "", "", false)
		require.NoError(t, err)
		assert.Greater(t, secondEntry.OrderIndex, first.OrderIndex)
		assert.Equal(t, first.MergeQueueID, secondEntry.MergeQueueID)
	})
	t.Run("activity failure is not fatal", func(t *testing.T) {
		env, pr := setup(t)
		env.activity.err = errors.New("db down")
		_, _, err := env.svc.Enqueue(ctx, pr, env.repo, 5, enum.MergeMethodSquash, "", "", false)
		require.NoError(t, err)
	})
	t.Run("invalid merge method", func(t *testing.T) {
		env, pr := setup(t)
		_, _, err := env.svc.Enqueue(ctx, pr, env.repo, 5, enum.MergeMethod("bogus"), "", "", false)
		require.Error(t, err)
	})
	t.Run("fast forward is not supported", func(t *testing.T) {
		env, pr := setup(t)
		_, _, err := env.svc.Enqueue(ctx, pr, env.repo, 5, enum.MergeMethodFastForward, "", "", false)
		require.Error(t, err)
	})
	t.Run("pull request must be queueable", func(t *testing.T) {
		env, pr := setup(t)
		pr.IsDraft = true
		_, _, err := env.svc.Enqueue(ctx, pr, env.repo, 5, enum.MergeMethodSquash, "", "", false)
		require.Error(t, err)
	})
	t.Run("full queue", func(t *testing.T) {
		env, pr := setup(t)
		env.entries.count = MaximumQueueSize
		_, _, err := env.svc.Enqueue(ctx, pr, env.repo, 5, enum.MergeMethodSquash, "", "", false)
		require.Error(t, err)
	})
	t.Run("queue creation failure", func(t *testing.T) {
		env, pr := setup(t)
		env.queues.createErr = errors.New("db down")
		_, _, err := env.svc.Enqueue(ctx, pr, env.repo, 5, enum.MergeMethodSquash, "", "", false)
		require.Error(t, err)
	})
	t.Run("sequence reservation failure", func(t *testing.T) {
		env, pr := setup(t)
		env.queues.updateErr = errors.New("db down")
		_, _, err := env.svc.Enqueue(ctx, pr, env.repo, 5, enum.MergeMethodSquash, "", "", false)
		require.Error(t, err)
	})
	t.Run("pull request changed while enqueueing", func(t *testing.T) {
		env, pr := setup(t)
		env.prs.prs[10] = &types.PullReq{ID: 10, State: enum.PullReqStateClosed}
		_, _, err := env.svc.Enqueue(ctx, pr, env.repo, 5, enum.MergeMethodSquash, "", "", false)
		require.Error(t, err)
	})
}

func TestSvcPrioritize(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T) (*svcEnv, *types.MergeQueue) {
		env := svcNewEnv(t)
		q := env.queues.add(&types.MergeQueue{RepoID: 1, Branch: "main", OrderSequence: 3})
		for i := int64(1); i <= 3; i++ {
			env.entries.entries[10+i] = svcEntry(10+i, q.ID, i, enum.MergeQueueEntryStateMergePending)
		}
		return env, q
	}

	t.Run("moves the entry to the front", func(t *testing.T) {
		env, q := setup(t)
		require.NoError(t, env.svc.Prioritize(ctx, 13))

		entries, err := env.entries.ListForMergeQueue(ctx, q.ID)
		require.NoError(t, err)
		var order []int64
		for _, e := range entries {
			order = append(order, e.PullReqID)
		}
		assert.Equal(t, []int64{13, 11, 12}, order)
		assert.Equal(t, int64(6), q.OrderSequence)
		assert.Equal(t, 1, env.producer.count())
	})
	t.Run("entry that is not queued", func(t *testing.T) {
		env, _ := setup(t)
		require.ErrorIs(t, env.svc.Prioritize(ctx, 999), ErrNotInQueue)
	})
	t.Run("entry already at the head", func(t *testing.T) {
		env, _ := setup(t)
		err := env.svc.Prioritize(ctx, 11)
		require.ErrorIs(t, err, ErrAlreadyHead)
		assert.Equal(t, 0, env.producer.count())
	})
	t.Run("store failures", func(t *testing.T) {
		env, _ := setup(t)
		env.queues.updateErr = errors.New("db down")
		require.Error(t, env.svc.Prioritize(ctx, 13))

		env, _ = setup(t)
		env.entries.listErr = errors.New("db down")
		require.Error(t, env.svc.Prioritize(ctx, 13))

		env, q := setup(t)
		delete(env.queues.queues, q.ID)
		require.Error(t, env.svc.Prioritize(ctx, 13))
	})
}

func TestSvcRemove(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T) (*svcEnv, *types.MergeQueue) {
		env := svcNewEnv(t)
		q := env.queues.add(&types.MergeQueue{RepoID: 1, Branch: "main"})
		for i := int64(1); i <= 3; i++ {
			e := svcEntry(10+i, q.ID, i, enum.MergeQueueEntryStateChecksPending)
			e.MergeCommitSHA = sha.Must(svcSHAOne)
			env.entries.entries[10+i] = e
			// Merged pull requests don't produce an activity, which keeps the test independent of the system session.
			env.prs.prs[10+i] = &types.PullReq{
				ID: 10 + i, TargetRepoID: 1, TargetBranch: "main", State: enum.PullReqStateMerged,
			}
		}
		return env, q
	}

	payload := types.PullRequestActivityPayloadMergeQueueRemove{Reason: enum.MergeQueueRemovalReasonCheckFail}

	t.Run("removes the entry and resets the following ones", func(t *testing.T) {
		env, _ := setup(t)
		require.NoError(t, env.svc.Remove(ctx, 11, payload))

		assert.Equal(t, []int64{11}, env.entries.deleted)
		assert.ElementsMatch(t, []int64{12, 13}, env.entries.updated)
		assert.Equal(t, enum.MergeQueueEntryStateMergePending, env.entries.entries[12].State)
		assert.True(t, env.entries.entries[12].MergeCommitSHA.IsEmpty())
		assert.Equal(t, 1, env.producer.count())
	})
	t.Run("removing the last entry resets nothing", func(t *testing.T) {
		env, _ := setup(t)
		require.NoError(t, env.svc.Remove(ctx, 13, payload))
		assert.Empty(t, env.entries.updated)
	})
	t.Run("entry that is not queued", func(t *testing.T) {
		env, _ := setup(t)
		require.ErrorIs(t, env.svc.Remove(ctx, 999, payload), ErrNotInQueue)
	})
	t.Run("pull request is missing", func(t *testing.T) {
		env, _ := setup(t)
		delete(env.prs.prs, 11)
		require.Error(t, env.svc.Remove(ctx, 11, payload))
	})
	t.Run("queue vanished", func(t *testing.T) {
		env, q := setup(t)
		delete(env.queues.queues, q.ID)
		require.Error(t, env.svc.Remove(ctx, 11, payload))
	})
	t.Run("entry listing failure", func(t *testing.T) {
		env, _ := setup(t)
		env.entries.listErr = errors.New("db down")
		require.Error(t, env.svc.Remove(ctx, 11, payload))
	})
	t.Run("pull request was not found in the queue's entry list", func(t *testing.T) {
		env, q := setup(t)
		env.entries.entries[11].MergeQueueID = q.ID + 100
		err := env.svc.remove(ctx, 11, payload)
		require.ErrorIs(t, err, ErrNotInQueue)
	})
}

func TestSvcRemoveAll_NoQueue(t *testing.T) {
	env := svcNewEnv(t)
	require.NoError(t, env.svc.RemoveAll(context.Background(), env.repo, "main",
		types.PullRequestActivityPayloadMergeQueueRemove{Reason: enum.MergeQueueRemovalReasonNoQueue}))

	env.queues.findErr = errors.New("db down")
	require.Error(t, env.svc.RemoveAll(context.Background(), env.repo, "main",
		types.PullRequestActivityPayloadMergeQueueRemove{}))
}

func TestSvcTriggerEvents(t *testing.T) {
	env := svcNewEnv(t)
	env.svc.TriggerBranchDeleted(context.Background(), 1, "main")
	env.svc.TriggerBranchUpdated(context.Background(), 1, "main")
	assert.Equal(t, 2, env.producer.count())
}

func TestSvcHandlerUpdated(t *testing.T) {
	ctx := context.Background()
	event := &events.Event[*mergequeueevents.UpdatedPayload]{
		ID: "e", Payload: &mergequeueevents.UpdatedPayload{Base: mergequeueevents.Base{RepoID: 1, Branch: "main"}},
	}

	t.Run("repository is missing", func(t *testing.T) {
		env := svcNewEnv(t)
		delete(env.repoCache.repos, 1)
		require.Error(t, env.svc.handlerUpdated(ctx, event))
	})
	t.Run("rule listing fails", func(t *testing.T) {
		env := svcNewEnv(t)
		env.rules.err = errors.New("db down")
		require.Error(t, env.svc.handlerUpdated(ctx, event))
	})
	t.Run("no merge queue configured", func(t *testing.T) {
		env := svcNewEnv(t)
		require.NoError(t, env.svc.handlerUpdated(ctx, event))
	})
	t.Run("merge queue configured but no queue exists", func(t *testing.T) {
		env := svcNewEnv(t)
		env.rules.rules = svcActiveRules()
		require.NoError(t, env.svc.handlerUpdated(ctx, event))
	})
	t.Run("queue lookup fails", func(t *testing.T) {
		env := svcNewEnv(t)
		env.rules.rules = svcActiveRules()
		env.queues.findErr = errors.New("db down")
		require.Error(t, env.svc.handlerUpdated(ctx, event))
	})
	t.Run("reprocess errors are returned", func(t *testing.T) {
		env := svcNewEnv(t)
		env.rules.rules = svcActiveRules()
		env.queues.add(&types.MergeQueue{RepoID: 1, Branch: "main"})
		env.git.refErr = errors.New("git down")
		require.Error(t, env.svc.handlerUpdated(ctx, event))
	})
}

func TestSvcHandlerBranchDeleted(t *testing.T) {
	ctx := context.Background()
	event := &events.Event[*mergequeueevents.BranchDeletedPayload]{
		ID: "e", Payload: &mergequeueevents.BranchDeletedPayload{Base: mergequeueevents.Base{RepoID: 1, Branch: "main"}},
	}

	env := svcNewEnv(t)
	require.NoError(t, env.svc.handlerBranchDeleted(ctx, event), "nothing queued for the branch")

	env.queues.findErr = errors.New("db down")
	require.Error(t, env.svc.handlerBranchDeleted(ctx, event))

	delete(env.repoCache.repos, 1)
	require.Error(t, env.svc.handlerBranchDeleted(ctx, event))
}

func TestSvcHandlerCheckFinished(t *testing.T) {
	ctx := context.Background()
	payload := func(status enum.CheckStatus, commit string, bypassed bool) *events.Event[*checkevents.ReportedPayload] {
		return &events.Event[*checkevents.ReportedPayload]{
			ID: "e",
			Payload: &checkevents.ReportedPayload{
				Base: checkevents.Base{RepoID: 1, SHA: commit}, Identifier: "ci", Status: status, Bypassed: bypassed,
			},
		}
	}

	setup := func(t *testing.T, state enum.MergeQueueEntryState) (*svcEnv, *types.MergeQueueEntry) {
		env := svcNewEnv(t)
		env.rules.rules = svcActiveRules()
		q := env.queues.add(&types.MergeQueue{RepoID: 1, Branch: "main"})
		entry := svcEntry(10, q.ID, 1, state)
		entry.MergeCommitSHA = sha.Must(svcSHAOne)
		entry.ChecksCommitSHA = sha.Must(svcSHAOne)
		env.entries.entries[10] = entry
		env.entries.byMergeSHA = map[sha.SHA]*types.MergeQueueEntry{sha.Must(svcSHAOne): entry}
		env.prs.prs[10] = &types.PullReq{
			ID: 10, TargetRepoID: 1, TargetBranch: "main", State: enum.PullReqStateMerged,
		}
		return env, entry
	}

	t.Run("running checks are ignored", func(t *testing.T) {
		env, _ := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		require.NoError(t, env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusRunning, svcSHAOne, false)))
		assert.Empty(t, env.entries.deleted)
	})
	t.Run("invalid commit sha is discarded", func(t *testing.T) {
		env, _ := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		err := env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusSuccess, "bad", false))
		require.Error(t, err)
		assert.True(t, errors.Is(err, events.NewDiscardEventError(nil)))
	})
	t.Run("missing repository", func(t *testing.T) {
		env, _ := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		delete(env.repoCache.repos, 1)
		require.Error(t, env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusSuccess, svcSHAOne, false)))
	})
	t.Run("check of a non merge queue commit", func(t *testing.T) {
		env, _ := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		require.NoError(t, env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusSuccess, svcSHATwo, false)))
	})
	t.Run("entry lookup failure", func(t *testing.T) {
		env, _ := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		env.entries.findMergeEr = errors.New("db down")
		require.Error(t, env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusSuccess, svcSHAOne, false)))
	})
	t.Run("queue lookup failure", func(t *testing.T) {
		env, entry := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		entry.MergeQueueID = 999
		require.Error(t, env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusSuccess, svcSHAOne, false)))
	})
	t.Run("rule listing failure", func(t *testing.T) {
		env, _ := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		env.rules.err = errors.New("db down")
		require.Error(t, env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusSuccess, svcSHAOne, false)))
	})
	t.Run("failed check removes the entry and reprocesses", func(t *testing.T) {
		env, _ := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		env.checks.byIdent = types.Check{Link: "https://ci.example/1"}
		env.git.refErr = errors.New("git down")

		err := env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusFailure, svcSHAOne, false))
		require.Error(t, err, "reprocess fails because the git fake is down")
		assert.Equal(t, []int64{10}, env.entries.deleted)
	})
	t.Run("failed check for an entry that already left the queue", func(t *testing.T) {
		env, entry := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		delete(env.entries.entries, entry.PullReqID)
		require.NoError(t, env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusFailure, svcSHAOne, false)))
	})
	t.Run("failed check without check details still removes", func(t *testing.T) {
		env, _ := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		env.checks.identEr = errors.New("not found")
		env.git.refErr = errors.New("git down")
		require.Error(t, env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusFailure, svcSHAOne, false)))
		assert.Equal(t, []int64{10}, env.entries.deleted)
	})
	t.Run("success of a non leader entry is ignored", func(t *testing.T) {
		env, _ := setup(t, enum.MergeQueueEntryStateMergeGroup)
		require.NoError(t, env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusSuccess, svcSHAOne, false)))
		assert.Empty(t, env.entries.deleted)
	})
	t.Run("success waits for the remaining required checks", func(t *testing.T) {
		env, _ := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		env.checks.checks = []types.Check{{Identifier: "ci", Status: enum.CheckStatusSuccess}}
		require.NoError(t, env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusSuccess, svcSHAOne, false)))
		assert.Empty(t, env.entries.deleted)
	})
	t.Run("success check listing failure", func(t *testing.T) {
		env, _ := setup(t, enum.MergeQueueEntryStateChecksInProgress)
		env.checks.listErr = errors.New("db down")
		require.Error(t, env.svc.handlerCheckFinished(ctx, payload(enum.CheckStatusSuccess, svcSHAOne, false)))
	})
}

func TestSvcOverdueJobHandle(t *testing.T) {
	ctx := context.Background()
	started := int64(100)
	deadline := int64(200)

	t.Run("listing failure", func(t *testing.T) {
		env := svcNewEnv(t)
		env.entries.overdueErr = errors.New("db down")
		job := &jobOverdueChecks{mergeQueueEntryStore: env.entries, service: env.svc}
		_, err := job.Handle(ctx, "", nil)
		require.Error(t, err)
	})
	t.Run("nothing overdue", func(t *testing.T) {
		env := svcNewEnv(t)
		job := &jobOverdueChecks{mergeQueueEntryStore: env.entries, service: env.svc}
		_, err := job.Handle(ctx, "", nil)
		require.NoError(t, err)
	})
	t.Run("overdue entries that already left the queue are skipped", func(t *testing.T) {
		env := svcNewEnv(t)
		q := env.queues.add(&types.MergeQueue{RepoID: 1, Branch: "main"})
		entry := svcEntry(10, q.ID, 1, enum.MergeQueueEntryStateChecksInProgress)
		entry.ChecksCommitSHA = sha.Must(svcSHAOne)
		entry.ChecksStarted = &started
		entry.ChecksDeadline = &deadline
		env.entries.overdue = []*types.MergeQueueEntry{entry}
		env.checks.checks = []types.Check{
			{Identifier: "done", Status: enum.CheckStatusSuccess},
			{Identifier: "slow", Status: enum.CheckStatusRunning, Link: "https://ci.example/slow"},
		}
		job := &jobOverdueChecks{mergeQueueEntryStore: env.entries, service: env.svc}

		_, err := job.Handle(ctx, "", nil)
		require.NoError(t, err, "removal failures are logged and the job continues")

		ident, link := job.getCheckInfo(ctx, entry, &repoCache{}, zeroLogger())
		assert.Equal(t, "slow", ident)
		assert.Equal(t, "https://ci.example/slow", link)
	})
	t.Run("check info is empty when lookups fail", func(t *testing.T) {
		env := svcNewEnv(t)
		job := &jobOverdueChecks{mergeQueueEntryStore: env.entries, service: env.svc}
		entry := svcEntry(10, 404, 1, enum.MergeQueueEntryStateChecksInProgress)

		ident, link := job.getCheckInfo(ctx, entry, &repoCache{}, zeroLogger())
		assert.Empty(t, ident)
		assert.Empty(t, link)

		q := env.queues.add(&types.MergeQueue{RepoID: 1, Branch: "main"})
		entry.MergeQueueID = q.ID
		env.checks.listErr = errors.New("db down")
		ident, link = job.getCheckInfo(ctx, entry, &repoCache{}, zeroLogger())
		assert.Empty(t, ident)
		assert.Empty(t, link)
	})
}

func zeroLogger() zerolog.Logger { return zerolog.Nop() }
