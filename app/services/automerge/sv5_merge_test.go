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

package automerge

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
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

const (
	sv5SystemPrincipalID = 77
	sv5SHAa              = "1111111111111111111111111111111111111111"
	sv5SHAb              = "2222222222222222222222222222222222222222"
	sv5SHAc              = "3333333333333333333333333333333333333333"
)

type sv5PrincipalStore struct {
	store.PrincipalStore

	err error
}

func (s sv5PrincipalStore) FindServiceByUID(context.Context, string) (*types.Service, error) {
	return &types.Service{ID: sv5SystemPrincipalID, UID: "sys", DisplayName: "System", Admin: true}, nil
}

func (s sv5PrincipalStore) Find(_ context.Context, id int64) (*types.Principal, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &types.Principal{ID: id, Email: "user@example.com", DisplayName: "User"}, nil
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

type sv5Git struct {
	git.Interface

	headSHA    sha.SHA
	branchSHA  sha.SHA
	mergeOut   git.MergeOutput
	mergeErr   error
	getRefErr  error
	merges     int
	updateRefs int
	deleted    []string
}

func (g *sv5Git) GetRef(context.Context, git.GetRefParams) (git.GetRefResponse, error) {
	return git.GetRefResponse{SHA: g.headSHA}, g.getRefErr
}

func (g *sv5Git) GetBranch(context.Context, *git.GetBranchParams) (*git.GetBranchOutput, error) {
	return &git.GetBranchOutput{Branch: git.Branch{SHA: g.branchSHA}}, nil
}

func (g *sv5Git) IsAncestor(context.Context, git.IsAncestorParams) (git.IsAncestorOutput, error) {
	return git.IsAncestorOutput{Ancestor: true}, nil
}

func (g *sv5Git) GetTreeNode(context.Context, *git.GetTreeNodeParams) (*git.GetTreeNodeOutput, error) {
	return nil, gitnesserrors.NotFound("no codeowners")
}

func (g *sv5Git) Merge(context.Context, *git.MergeParams) (git.MergeOutput, error) {
	g.merges++
	return g.mergeOut, g.mergeErr
}

func (g *sv5Git) UpdateRefs(context.Context, git.UpdateRefsParams) error {
	g.updateRefs++
	return nil
}

func (g *sv5Git) DeleteBranch(_ context.Context, p *git.DeleteBranchParams) error {
	g.deleted = append(g.deleted, p.BranchName)
	return nil
}

type sv5RepoStore struct{ store.RepoStore }

func (sv5RepoStore) Find(context.Context, int64) (*types.Repository, error) {
	return &types.Repository{ID: 1, NumOpenPulls: 3}, nil
}

func (sv5RepoStore) UpdateOptLock(
	_ context.Context, r *types.Repository, fn func(*types.Repository) error,
) (*types.Repository, error) {
	cp := *r
	return &cp, fn(&cp)
}

type sv5PullReqStore struct {
	store.PullReqStore

	pr      *types.PullReq
	findErr error
	updates int
}

func (s *sv5PullReqStore) Find(context.Context, int64) (*types.PullReq, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	cp := *s.pr
	return &cp, nil
}

func (s *sv5PullReqStore) Update(_ context.Context, pr *types.PullReq) error {
	s.updates++
	*s.pr = *pr
	return nil
}

type sv5ActivityStore struct {
	store.PullReqActivityStore

	err   error
	count int
}

func (s *sv5ActivityStore) CreateWithPayload(
	context.Context, *types.PullReq, int64, types.PullReqActivityPayload, *types.PullReqActivityMetadata,
) (*types.PullReqActivity, error) {
	s.count++
	return &types.PullReqActivity{}, s.err
}

type sv5AutoMergeStore struct {
	store.AutoMergeStore

	entry     *types.AutoMerge
	findErr   error
	deleteErr error
	deletes   int
}

func (s *sv5AutoMergeStore) Find(context.Context, int64) (*types.AutoMerge, error) {
	return s.entry, s.findErr
}

func (s *sv5AutoMergeStore) Delete(context.Context, int64) (bool, error) {
	s.deletes++
	return true, s.deleteErr
}

type sv5CheckStore struct{ store.CheckStore }

func (sv5CheckStore) ListResults(context.Context, int64, string) ([]types.CheckResult, error) {
	return nil, nil
}

type sv5ReviewerStore struct{ store.PullReqReviewerStore }

func (sv5ReviewerStore) List(context.Context, int64) ([]*types.PullReqReviewer, error) {
	return nil, nil
}

type sv5RuleStore struct {
	store.RuleStore

	rules []types.RuleInfoInternal
	err   error
}

func (s *sv5RuleStore) ListAllRepoRules(
	context.Context, int64, ...enum.RuleType,
) ([]types.RuleInfoInternal, error) {
	return s.rules, s.err
}

type sv5RepoIDCache struct{ store.RepoIDCache }

func (sv5RepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if id == 404 {
		return nil, errSv5Boom
	}
	if id == 405 {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return &types.RepositoryCore{ID: id, GitUID: "git-uid", ParentID: 3, Path: "space/repo", DefaultBranch: "main"}, nil
}

type sv5SSE struct {
	sse.Streamer

	published []enum.SSEType
}

func (s *sv5SSE) Publish(_ context.Context, _ int64, t enum.SSEType, _ any) {
	s.published = append(s.published, t)
}

type sv5URLProvider struct{ gitnessurl.Provider }

func (sv5URLProvider) GetInternalAPIURL(context.Context) string { return "http://internal.test" }

type sv5UserGroups struct{ usergroup.Service }

func (sv5UserGroups) ListUserIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return nil, nil
}

func (sv5UserGroups) MapGroupIDsToPrincipals(context.Context, []int64) (map[int64][]*types.Principal, error) {
	return map[int64][]*types.Principal{}, nil
}

type sv5Instrument struct{}

func (sv5Instrument) Close(context.Context) error { return nil }

type sv5Producer struct{}

func (sv5Producer) Send(context.Context, string, map[string]any) (string, error) { return "1", nil }

type sv5Env struct {
	svc        *Service
	git        *sv5Git
	prs        *sv5PullReqStore
	activity   *sv5ActivityStore
	autoM      *sv5AutoMergeStore
	rules      *sv5RuleStore
	sse        *sv5SSE
	principals sv5PrincipalStore
}

func sv5Rule(definition string) types.RuleInfoInternal {
	return types.RuleInfoInternal{
		RuleInfo:   types.RuleInfo{ID: 1, Identifier: "r", Type: protection.TypeBranch, State: enum.RuleStateActive},
		RepoTarget: json.RawMessage(`{}`),
		Pattern:    json.RawMessage(`{"default":true}`),
		Definition: json.RawMessage(definition),
	}
}

func sv5OpenPR() *types.PullReq {
	srcRepoID := int64(1)
	return &types.PullReq{
		ID: 7, Number: 12, SourceRepoID: &srcRepoID, TargetRepoID: 1,
		SourceBranch: "feature", TargetBranch: "main", SourceSHA: sv5SHAa,
		State: enum.PullReqStateOpen, SubState: enum.PullReqSubStateAutoMerge,
		MergeCheckStatus: enum.MergeCheckStatusMergeable, ActivitySeq: 4,
	}
}

func sv5NewEnv(t *testing.T) *sv5Env {
	t.Helper()
	sv5Bootstrap(t)

	g := &sv5Git{
		headSHA:   sha.Must(sv5SHAa),
		branchSHA: sha.Must(sv5SHAb),
		mergeOut: git.MergeOutput{
			HeadSHA: sha.Must(sv5SHAa), BaseSHA: sha.Must(sv5SHAb), MergeSHA: sha.Must(sv5SHAc),
		},
	}
	env := &sv5Env{
		git:      g,
		prs:      &sv5PullReqStore{pr: sv5OpenPR()},
		activity: &sv5ActivityStore{},
		autoM: &sv5AutoMergeStore{entry: &types.AutoMerge{
			PullReqID: 7, RequestedBy: 5, MergeMethod: enum.MergeMethodSquash,
		}},
		rules:      &sv5RuleStore{rules: []types.RuleInfoInternal{sv5Rule(`{}`)}},
		sse:        &sv5SSE{},
		principals: sv5PrincipalStore{},
	}

	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errSv5Boom },
		sv5Producer{},
		events.NewNoopCollector(),
		false,
	)
	require.NoError(t, err)
	reporter, err := pullreqevents.NewReporter(system)
	require.NoError(t, err)

	finder := refcache.NewRepoFinder(nil, nil, sv5RepoIDCache{}, nil, cache.Evictor[*types.RepositoryCore]{})

	mergeSvc := merge.NewService(
		g, sv5Tx{}, reporter, finder, sv5RepoStore{}, env.prs, env.activity, sv5CheckStore{},
		sv5ReviewerStore{}, env.autoM, codeowners.New(nil, g, codeowners.Config{FilePaths: []string{"CODEOWNERS"}}, nil, nil),
		sv5UserGroups{}, sv5URLProvider{}, env.sse, sv5InstrumentSvc{}, nil,
	)

	manager := protection.NewManager(env.rules)
	require.NoError(t, manager.Register(protection.TypeBranch, func() protection.Definition {
		return &protection.Branch{}
	}))

	env.svc = &Service{
		git:               g,
		tx:                sv5Tx{},
		mergeService:      mergeSvc,
		repoFinder:        finder,
		pullreqStore:      env.prs,
		activityStore:     env.activity,
		principalStore:    env.principals,
		autoMergeStore:    env.autoM,
		protectionManager: manager,
		sseStreamer:       env.sse,
		locker: locker.NewLocker(lock.NewInMemory(lock.Config{
			App: "test", Expiry: 10 * time.Second, Tries: 2, RetryDelay: 10 * time.Millisecond,
			DriftFactor: 0.01, TimeoutFactor: 0.25,
		})),
	}
	return env
}

func sv5Input(method enum.MergeMethod) types.AutoMergeInput {
	return types.AutoMergeInput{
		Principal:   types.Principal{ID: 5, Email: "user@example.com", DisplayName: "User"},
		MergeMethod: method,
	}
}

func TestSv5MergeSuccess(t *testing.T) {
	env := sv5NewEnv(t)
	env.rules.rules = []types.RuleInfoInternal{sv5Rule(`{"pullreq":{"merge":{"delete_branch":true}}}`)}

	pr, deleted, err := env.svc.Merge(context.Background(), env.prs.pr, sv5Input(enum.MergeMethodSquash))
	require.NoError(t, err)
	require.NotNil(t, pr.Merged)
	require.Equal(t, enum.PullReqStateMerged, pr.State)
	require.True(t, deleted)
	require.Equal(t, []string{"feature"}, env.git.deleted)
	require.Equal(t, 1, env.git.merges)
	require.Equal(t, 1, env.git.updateRefs)
	require.Contains(t, env.sse.published, enum.SSETypePullReqUpdated)
}

func TestSv5MergeKeepsBranchWithoutRuleOrInput(t *testing.T) {
	env := sv5NewEnv(t)

	_, deleted, err := env.svc.Merge(context.Background(), env.prs.pr, sv5Input(enum.MergeMethodRebase))
	require.NoError(t, err)
	require.False(t, deleted)
	require.Empty(t, env.git.deleted)
}

func TestSv5MergeDeleteBranchByInput(t *testing.T) {
	env := sv5NewEnv(t)
	in := sv5Input(enum.MergeMethodMerge)
	in.DeleteBranch = true

	_, deleted, err := env.svc.Merge(context.Background(), env.prs.pr, in)
	require.NoError(t, err)
	require.True(t, deleted)
}

func TestSv5MergeErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		mutate func(env *sv5Env, pr *types.PullReq)
		method enum.MergeMethod
		is     error
		msg    string
	}{
		{"draft", func(_ *sv5Env, pr *types.PullReq) { pr.IsDraft = true }, enum.MergeMethodMerge, ErrNotEligible, ""},
		{"no source repo", func(_ *sv5Env, pr *types.PullReq) { pr.SourceRepoID = nil }, enum.MergeMethodMerge,
			ErrNotEligible, ""},
		{"target repo lookup", func(_ *sv5Env, pr *types.PullReq) { pr.TargetRepoID = 404 }, enum.MergeMethodMerge,
			errSv5Boom, "find target repo"},
		{"source repo lookup", func(_ *sv5Env, pr *types.PullReq) {
			id := int64(404)
			pr.SourceRepoID = &id
		}, enum.MergeMethodMerge, errSv5Boom, "source repository"},
		{"head ref", func(env *sv5Env, _ *types.PullReq) { env.git.getRefErr = errSv5Boom }, enum.MergeMethodMerge,
			errSv5Boom, "commit SHAs"},
		{"rules listing", func(env *sv5Env, _ *types.PullReq) { env.rules.err = errSv5Boom }, enum.MergeMethodMerge,
			errSv5Boom, "protection rules"},
		{"rule violation", func(env *sv5Env, _ *types.PullReq) {
			env.rules.rules = []types.RuleInfoInternal{sv5Rule(`{"pullreq":{"merge":{"block":true}}}`)}
		}, enum.MergeMethodMerge, ErrRuleViolation, ""},
		{"method not allowed", func(env *sv5Env, _ *types.PullReq) {
			env.rules.rules = []types.RuleInfoInternal{
				sv5Rule(`{"pullreq":{"merge":{"strategies_allowed":["squash"]}}}`)}
		}, enum.MergeMethodRebase, ErrMethodNotAllowed, ""},
		{"merge failure", func(env *sv5Env, _ *types.PullReq) { env.git.mergeErr = errSv5Boom }, enum.MergeMethodMerge,
			errSv5Boom, "merge pull request"},
		{"conflict", func(env *sv5Env, _ *types.PullReq) { env.git.mergeOut = git.MergeOutput{} }, enum.MergeMethodMerge,
			ErrConflict, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := sv5NewEnv(t)
			pr := sv5OpenPR()
			tc.mutate(env, pr)
			_, _, err := env.svc.Merge(ctx, pr, sv5Input(tc.method))
			require.ErrorIs(t, err, tc.is)
			if tc.msg != "" {
				require.ErrorContains(t, err, tc.msg)
			}
		})
	}
}

func TestSv5AutoMergeOutcomes(t *testing.T) {
	ctx := context.Background()

	t.Run("not eligible is a no-op", func(t *testing.T) {
		env := sv5NewEnv(t)
		pr := sv5OpenPR()
		pr.SubState = enum.PullReqSubStateNone
		require.NoError(t, env.svc.autoMerge(ctx, pr))
		require.Zero(t, env.git.merges)
	})

	t.Run("merges", func(t *testing.T) {
		env := sv5NewEnv(t)
		require.NoError(t, env.svc.autoMerge(ctx, env.prs.pr))
		require.Equal(t, 1, env.git.merges)
	})

	t.Run("auto merge entry missing is discarded", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.autoM.findErr = gitnessstore.ErrResourceNotFound
		err := env.svc.autoMerge(ctx, env.prs.pr)
		require.Error(t, err)
		require.True(t, sv5IsDiscard(err))
	})

	t.Run("auto merge lookup failure is retried", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.autoM.findErr = errSv5Boom
		err := env.svc.autoMerge(ctx, env.prs.pr)
		require.ErrorIs(t, err, errSv5Boom)
		require.False(t, sv5IsDiscard(err))
	})

	t.Run("principal lookup failure is retried", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.svc.principalStore = sv5PrincipalStore{err: errSv5Boom}
		require.ErrorIs(t, env.svc.autoMerge(ctx, env.prs.pr), errSv5Boom)
	})

	t.Run("rule violation and conflict are discarded", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.rules.rules = []types.RuleInfoInternal{sv5Rule(`{"pullreq":{"merge":{"block":true}}}`)}
		require.True(t, sv5IsDiscard(env.svc.autoMerge(ctx, env.prs.pr)))

		env = sv5NewEnv(t)
		env.git.mergeOut = git.MergeOutput{}
		require.True(t, sv5IsDiscard(env.svc.autoMerge(ctx, env.prs.pr)))
	})

	t.Run("other merge failure is retried", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.git.mergeErr = errSv5Boom
		err := env.svc.autoMerge(ctx, env.prs.pr)
		require.ErrorIs(t, err, errSv5Boom)
		require.False(t, sv5IsDiscard(err))
	})

	t.Run("not eligible while merging is discarded", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.svc.repoFinder = refcache.NewRepoFinder(
			nil, nil, sv5RepoIDCache{}, nil, cache.Evictor[*types.RepositoryCore]{})
		env.prs.pr.SourceRepoID = nil
		require.True(t, sv5IsDiscard(env.svc.autoMerge(ctx, env.prs.pr)))
	})

	t.Run("method not allowed disables auto merge", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.rules.rules = []types.RuleInfoInternal{
			sv5Rule(`{"pullreq":{"merge":{"strategies_allowed":["rebase"]}}}`)}
		require.NoError(t, env.svc.autoMerge(ctx, env.prs.pr))
		require.Equal(t, enum.PullReqSubStateNone, env.prs.pr.SubState)
		require.Equal(t, 1, env.autoM.deletes)
		require.Equal(t, 1, env.activity.count)
		require.Equal(t, []enum.SSEType{enum.SSETypePullReqAutoMergeDisabled}, env.sse.published)
	})
}

func TestSv5DisableAutoMerge(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		env := sv5NewEnv(t)
		require.NoError(t, env.svc.disableAutoMerge(ctx, 7, enum.MergeMethodSquash))
		require.Equal(t, enum.PullReqSubStateNone, env.prs.pr.SubState)
		require.Equal(t, int64(5), env.prs.pr.ActivitySeq)
	})

	failing := map[string]func(env *sv5Env){
		"pull request lookup": func(env *sv5Env) { env.prs.findErr = errSv5Boom },
		"already disabled":    func(env *sv5Env) { env.prs.pr.SubState = enum.PullReqSubStateNone },
		"repo lookup":         func(env *sv5Env) { env.prs.pr.TargetRepoID = 404 },
		"auto merge delete":   func(env *sv5Env) { env.autoM.deleteErr = errSv5Boom },
		"activity":            func(env *sv5Env) { env.activity.err = errSv5Boom },
	}
	for name, mutate := range failing {
		t.Run(name, func(t *testing.T) {
			env := sv5NewEnv(t)
			mutate(env)
			require.Error(t, env.svc.disableAutoMerge(ctx, 7, enum.MergeMethodSquash))
			require.Empty(t, env.sse.published)
		})
	}
}

type sv5InstrumentSvc struct{ sv5Instrument }

func (sv5InstrumentSvc) Track(context.Context, instrument.Event) error { return nil }

func sv5IsDiscard(err error) bool {
	return err != nil && strings.Contains(err.Error(), "discarding requested due to")
}
