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

package rules

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	ruleevents "github.com/harness/gitness/app/events/rule"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/events"
	mocksse "github.com/harness/gitness/mocks/sse"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

type svbTx struct{}

func (svbTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

type svbRuleStore struct {
	store.RuleStore

	rule      *types.Rule
	findErr   error
	list      []types.Rule
	count     int64
	created   *types.Rule
	updated   *types.Rule
	deletedID int64
	writeErr  error
	listErr   error
	countErr  error
	countCall int
}

func (f *svbRuleStore) FindByIdentifier(
	context.Context, enum.RuleParent, int64, string,
) (*types.Rule, error) {
	return f.rule, f.findErr
}

func (f *svbRuleStore) Create(_ context.Context, r *types.Rule) error {
	f.created = r
	r.ID = 42
	return f.writeErr
}

func (f *svbRuleStore) Update(_ context.Context, r *types.Rule) error {
	f.updated = r
	return f.writeErr
}

func (f *svbRuleStore) Delete(_ context.Context, id int64) error {
	f.deletedID = id
	return f.writeErr
}

func (f *svbRuleStore) List(context.Context, []types.RuleParentInfo, *types.RuleFilter) ([]types.Rule, error) {
	return f.list, f.listErr
}

func (f *svbRuleStore) Count(context.Context, []types.RuleParentInfo, *types.RuleFilter) (int64, error) {
	f.countCall++
	return f.count, f.countErr
}

type svbRepoStore struct {
	store.RepoStore

	repo *types.Repository
	err  error
}

func (f *svbRepoStore) Find(context.Context, int64) (*types.Repository, error) { return f.repo, f.err }

type svbSpaceStore struct {
	store.SpaceStore

	level int64
	ids   []int64
	err   error
}

func (f *svbSpaceStore) GetTreeLevel(context.Context, int64) (int64, error) { return f.level, f.err }

func (f *svbSpaceStore) GetAncestorIDs(context.Context, int64) ([]int64, error) { return f.ids, f.err }

type svbPrincipalCache struct {
	store.PrincipalInfoCache

	err error
}

func (f *svbPrincipalCache) Map(_ context.Context, ids []int64) (map[int64]*types.PrincipalInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := map[int64]*types.PrincipalInfo{}
	for _, id := range ids {
		out[id] = &types.PrincipalInfo{ID: id}
	}
	return out, nil
}

type svbUserGroupStore struct {
	store.UserGroupStore

	err error
}

func (f *svbUserGroupStore) Map(_ context.Context, ids []int64) (map[int64]*types.UserGroup, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := map[int64]*types.UserGroup{}
	for _, id := range ids {
		out[id] = &types.UserGroup{ID: id}
	}
	return out, nil
}

type svbRepoIDCache struct {
	store.RepoIDCache

	repos map[int64]*types.RepositoryCore
	err   error
}

func (f *svbRepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if f.err != nil {
		return nil, f.err
	}
	r, ok := f.repos[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return r, nil
}

type svbAudit struct {
	actions []audit.Action
	paths   []string
	err     error
}

func (a *svbAudit) Log(
	_ context.Context, _ types.Principal, _ audit.Resource, action audit.Action, path string, _ ...audit.Option,
) error {
	a.actions = append(a.actions, action)
	a.paths = append(a.paths, path)
	return a.err
}

type svbInstrument struct {
	events []instrument.Event
	err    error
}

func (i *svbInstrument) Track(_ context.Context, e instrument.Event) error {
	i.events = append(i.events, e)
	return i.err
}

func (i *svbInstrument) Close(context.Context) error { return nil }

type svbProducer struct{ sent int }

func (p *svbProducer) Send(context.Context, string, map[string]any) (string, error) {
	p.sent++
	return "1", nil
}

type svbValidator struct{ err error }

func (v svbValidator) Validate(context.Context, []int64, map[int64]*types.PrincipalInfo) error {
	return v.err
}

type svbEnv struct {
	svc   *Service
	rules *svbRuleStore
	audit *svbAudit
	instr *svbInstrument
	sse   *mocksse.Streamer
	prod  *svbProducer
	repos *svbRepoStore
	space *svbSpaceStore
	cache *svbRepoIDCache
}

func svbNewEnv(t *testing.T) *svbEnv {
	t.Helper()

	prod := &svbProducer{}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errSvbBoom },
		prod,
		events.NewNoopCollector(),
	)
	require.NoError(t, err)
	reporter, err := ruleevents.NewReporter(system)
	require.NoError(t, err)

	manager := protection.NewManager(nil)
	require.NoError(t, manager.Register(protection.TypeBranch, func() protection.Definition {
		return &protection.Branch{}
	}))

	env := &svbEnv{
		rules: &svbRuleStore{},
		audit: &svbAudit{},
		instr: &svbInstrument{},
		sse:   &mocksse.Streamer{},
		prod:  prod,
		repos: &svbRepoStore{repo: &types.Repository{ParentID: 77}},
		space: &svbSpaceStore{level: 2, ids: []int64{3, 4}},
		cache: &svbRepoIDCache{repos: map[int64]*types.RepositoryCore{5: {ID: 5}}},
	}
	env.sse.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
	env.svc = NewService(
		svbTx{}, env.rules, env.repos, env.space, manager, env.audit, env.instr,
		&svbPrincipalCache{}, &svbUserGroupStore{}, nil, reporter, env.sse, svbValidator{}, env.cache,
	)
	return env
}

func svbBranchRule() *types.Rule {
	return &types.Rule{
		ID:         9,
		Identifier: "r1",
		Type:       protection.TypeBranch,
		State:      enum.RuleStateActive,
		Pattern:    json.RawMessage(`{}`),
		RepoTarget: json.RawMessage(`{"include":{"ids":[5,6]},"exclude":{"ids":[5]}}`),
		Definition: json.RawMessage(`{"bypass":{"user_ids":[1]}}`),
	}
}

var svbPrincipal = &types.Principal{ID: 3}

func TestSvbCreateInputSanitize(t *testing.T) {
	good := func() *CreateInput {
		return &CreateInput{Identifier: "rule1", State: enum.RuleStateActive, Definition: json.RawMessage(`{}`)}
	}
	tests := []struct {
		name    string
		mod     func(in *CreateInput)
		wantErr bool
	}{
		{"ok", func(*CreateInput) {}, false},
		{"uid fallback", func(in *CreateInput) { in.UID, in.Identifier = "legacy", "" }, false},
		{"bad identifier", func(in *CreateInput) { in.Identifier = "bad id!" }, true},
		{"bad pattern", func(in *CreateInput) { in.Pattern.Default = true; in.Pattern.Include = []string{"["} }, true},
		{"bad repo target", func(in *CreateInput) { in.RepoTarget.Include.IDs = []int64{-1} }, true},
		{"bad state", func(in *CreateInput) { in.State = "nope" }, true},
		{"no definition", func(in *CreateInput) { in.Definition = nil }, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := good()
			tc.mod(in)
			err := in.sanitize()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, protection.TypeBranch, in.Type)
			require.NotEmpty(t, in.Identifier)
		})
	}
}

func TestSvbUpdateInputSanitize(t *testing.T) {
	str := func(s string) *string { return &s }
	state := func(s enum.RuleState) *enum.RuleState { return &s }
	raw := func(s string) *json.RawMessage { r := json.RawMessage(s); return &r }

	empty := &UpdateInput{}
	require.NoError(t, empty.sanitize())
	require.True(t, empty.isEmpty())

	in := &UpdateInput{UID: str("legacy")}
	require.NoError(t, in.sanitize())
	require.Equal(t, "legacy", *in.Identifier)
	require.False(t, in.isEmpty())

	require.Error(t, (&UpdateInput{Identifier: str("bad id!")}).sanitize())
	require.Error(t, (&UpdateInput{State: state("nope")}).sanitize())
	require.NoError(t, (&UpdateInput{State: state(enum.RuleStateActive)}).sanitize())
	require.Error(t, (&UpdateInput{Pattern: &protection.Pattern{Include: []string{"["}}}).sanitize())
	require.Error(t, (&UpdateInput{RepoTarget: &protection.RepoTarget{
		Include: protection.RepoTargetFilter{IDs: []int64{-1}},
	}}).sanitize())
	require.Error(t, (&UpdateInput{Definition: raw("")}).sanitize())
	require.NoError(t, (&UpdateInput{Definition: raw("{}")}).sanitize())
}

func TestSvbCreate(t *testing.T) {
	ctx := context.Background()
	newInput := func() *CreateInput {
		return &CreateInput{
			Identifier: "rule1", State: enum.RuleStateActive,
			Definition: json.RawMessage(`{"bypass":{"user_ids":[1]}}`),
			RepoTarget: protection.RepoTarget{Include: protection.RepoTargetFilter{IDs: []int64{5}}},
		}
	}

	t.Run("repo", func(t *testing.T) {
		env := svbNewEnv(t)
		rule, err := env.svc.Create(ctx, svbPrincipal, enum.RuleParentRepo, 10, "repo", "acct/org/proj/repo", newInput())
		require.NoError(t, err)
		require.Equal(t, int64(42), rule.ID)
		require.Equal(t, int64(10), *rule.RepoID)
		require.Nil(t, rule.SpaceID)
		require.Equal(t, ruleScopeRepo, rule.Scope)
		require.Contains(t, rule.Users, int64(1))
		require.Contains(t, rule.Repositories, int64(5))
		require.Equal(t, []audit.Action{audit.ActionCreated}, env.audit.actions)
		require.Equal(t, []string{"acct/org/proj"}, env.audit.paths)
		require.Len(t, env.instr.events, 1)
		require.Contains(t, env.instr.events[0].Properties, instrument.PropertyRepositoryID)
		require.Equal(t, 1, env.prod.sent)
		env.sse.AssertCalled(t, "Publish", int64(77), enum.SSETypeRuleCreated, mock.Anything)
	})

	t.Run("space", func(t *testing.T) {
		env := svbNewEnv(t)
		in := newInput()
		rule, err := env.svc.Create(ctx, svbPrincipal, enum.RuleParentSpace, 10, "space", "acct/org", in)
		require.NoError(t, err)
		require.Equal(t, int64(10), *rule.SpaceID)
		require.Equal(t, int64(2), rule.Scope)
		require.Equal(t, []string{"acct/org"}, env.audit.paths)
		require.Contains(t, env.instr.events[0].Properties, instrument.PropertySpaceID)
		env.sse.AssertCalled(t, "Publish", int64(10), enum.SSETypeRuleCreated, mock.Anything)
	})

	t.Run("side effect failures are tolerated", func(t *testing.T) {
		env := svbNewEnv(t)
		env.audit.err = errSvbBoom
		env.instr.err = errSvbBoom
		_, err := env.svc.Create(ctx, svbPrincipal, enum.RuleParentRepo, 10, "repo", "a/b/c/repo", newInput())
		require.NoError(t, err)
	})

	t.Run("invalid input", func(t *testing.T) {
		env := svbNewEnv(t)
		in := newInput()
		in.Identifier = "bad id!"
		_, err := env.svc.Create(ctx, svbPrincipal, enum.RuleParentRepo, 10, "repo", "p/repo", in)
		require.Error(t, err)
	})

	t.Run("tree level error", func(t *testing.T) {
		env := svbNewEnv(t)
		env.space.err = errSvbBoom
		_, err := env.svc.Create(ctx, svbPrincipal, enum.RuleParentSpace, 10, "space", "p", newInput())
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("invalid definition", func(t *testing.T) {
		env := svbNewEnv(t)
		in := newInput()
		in.Definition = json.RawMessage(`{"unknown_field":true}`)
		_, err := env.svc.Create(ctx, svbPrincipal, enum.RuleParentRepo, 10, "repo", "p/repo", in)
		require.Error(t, err)
		require.Nil(t, env.rules.created)
	})

	t.Run("merge queue not allowed on space", func(t *testing.T) {
		env := svbNewEnv(t)
		in := newInput()
		in.Definition = json.RawMessage(`{"pullreq":{"merge_queue":{"enabled":true}}}`)
		_, err := env.svc.Create(ctx, svbPrincipal, enum.RuleParentSpace, 10, "space", "p", in)
		require.Error(t, err)
	})

	t.Run("validator error", func(t *testing.T) {
		env := svbNewEnv(t)
		env.svc.ruleValidator = svbValidator{err: errSvbBoom}
		_, err := env.svc.Create(ctx, svbPrincipal, enum.RuleParentRepo, 10, "repo", "p/repo", newInput())
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("store error", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.writeErr = errSvbBoom
		_, err := env.svc.Create(ctx, svbPrincipal, enum.RuleParentRepo, 10, "repo", "p/repo", newInput())
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("principal lookup error", func(t *testing.T) {
		env := svbNewEnv(t)
		env.svc.principalInfoCache = &svbPrincipalCache{err: errSvbBoom}
		_, err := env.svc.Create(ctx, svbPrincipal, enum.RuleParentRepo, 10, "repo", "p/repo", newInput())
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("backfill error", func(t *testing.T) {
		env := svbNewEnv(t)
		env.cache.err = errSvbBoom
		_, err := env.svc.Create(ctx, svbPrincipal, enum.RuleParentRepo, 10, "repo", "p/repo", newInput())
		require.ErrorIs(t, err, errSvbBoom)
	})
}

func TestSvbFind(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)
	env.rules.rule = svbBranchRule()

	rule, err := env.svc.Find(ctx, enum.RuleParentRepo, 1, "r1")
	require.NoError(t, err)
	require.Contains(t, rule.Users, int64(1))
	require.Contains(t, rule.Repositories, int64(5))
	require.Contains(t, rule.Repositories, int64(5))
	require.NotContains(t, rule.Repositories, int64(6), "unknown repositories are skipped")

	env.rules.findErr = errSvbBoom
	_, err = env.svc.Find(ctx, enum.RuleParentRepo, 1, "r1")
	require.ErrorIs(t, err, errSvbBoom)

	env.rules.findErr = nil
	env.rules.rule = svbBranchRule()
	env.rules.rule.Definition = json.RawMessage(`not-json`)
	_, err = env.svc.Find(ctx, enum.RuleParentRepo, 1, "r1")
	require.Error(t, err)

	env.rules.rule = svbBranchRule()
	env.rules.rule.RepoTarget = json.RawMessage(`not-json`)
	_, err = env.svc.Find(ctx, enum.RuleParentRepo, 1, "r1")
	require.ErrorContains(t, err, "backfill")
}

func TestSvbDelete(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv(t)
	env.rules.rule = svbBranchRule()

	require.NoError(t, env.svc.Delete(ctx, svbPrincipal, enum.RuleParentRepo, 10, "repo", "a/b/c/repo", "r1"))
	require.Equal(t, int64(9), env.rules.deletedID)
	require.Equal(t, []audit.Action{audit.ActionDeleted}, env.audit.actions)
	env.sse.AssertCalled(t, "Publish", int64(77), enum.SSETypeRuleDeleted, mock.Anything)

	env.audit.err = errSvbBoom
	require.NoError(t, env.svc.Delete(ctx, svbPrincipal, enum.RuleParentSpace, 10, "space", "a/b", "r1"))

	env.rules.writeErr = errSvbBoom
	require.ErrorIs(t, env.svc.Delete(ctx, svbPrincipal, enum.RuleParentSpace, 10, "s", "a", "r1"), errSvbBoom)

	env.rules.findErr = errSvbBoom
	require.ErrorIs(t, env.svc.Delete(ctx, svbPrincipal, enum.RuleParentSpace, 10, "s", "a", "r1"), errSvbBoom)
}

func TestSvbUpdate(t *testing.T) {
	ctx := context.Background()
	str := func(s string) *string { return &s }
	raw := func(s string) *json.RawMessage { r := json.RawMessage(s); return &r }

	run := func(env *svbEnv, parent enum.RuleParent, in *UpdateInput) (*types.Rule, error) {
		return env.svc.Update(ctx, svbPrincipal, parent, 10, "repo", "a/b/c/repo", "r1", in)
	}

	t.Run("find error", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.findErr = errSvbBoom
		_, err := run(env, enum.RuleParentRepo, &UpdateInput{Description: str("x")})
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("invalid input", func(t *testing.T) {
		env := svbNewEnv(t)
		_, err := run(env, enum.RuleParentRepo, &UpdateInput{Identifier: str("bad id!")})
		require.Error(t, err)
	})

	t.Run("empty input returns rule", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.rule = svbBranchRule()
		rule, err := run(env, enum.RuleParentRepo, &UpdateInput{})
		require.NoError(t, err)
		require.Contains(t, rule.Users, int64(1))
		require.Nil(t, env.rules.updated)
	})

	t.Run("empty input with broken definition", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.rule = svbBranchRule()
		env.rules.rule.Definition = json.RawMessage(`bad`)
		_, err := run(env, enum.RuleParentRepo, &UpdateInput{})
		require.Error(t, err)
	})

	t.Run("update all fields", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.rule = svbBranchRule()
		state := enum.RuleStateDisabled
		rule, err := run(env, enum.RuleParentRepo, &UpdateInput{
			Identifier:  str("renamed"),
			State:       &state,
			Description: str("desc"),
			Pattern:     &protection.Pattern{Default: true},
			RepoTarget:  &protection.RepoTarget{Include: protection.RepoTargetFilter{IDs: []int64{5}}},
			Definition:  raw(`{"bypass":{"user_ids":[2]}}`),
		})
		require.NoError(t, err)
		require.Same(t, env.rules.updated, rule)
		require.Equal(t, "renamed", rule.Identifier)
		require.Equal(t, enum.RuleStateDisabled, rule.State)
		require.Equal(t, "desc", rule.Description)
		require.Contains(t, rule.Users, int64(2))
		require.Equal(t, []audit.Action{audit.ActionUpdated}, env.audit.actions)
		env.sse.AssertCalled(t, "Publish", int64(77), enum.SSETypeRuleUpdated, mock.Anything)
	})

	t.Run("space parent with audit failure", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.rule = svbBranchRule()
		env.audit.err = errSvbBoom
		_, err := run(env, enum.RuleParentSpace, &UpdateInput{Description: str("desc")})
		require.NoError(t, err)
		require.Equal(t, []string{"a/b/c/repo"}, env.audit.paths)
	})

	t.Run("no effective change skips store", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.rule = svbBranchRule()
		same := env.rules.rule.Description
		_, err := run(env, enum.RuleParentRepo, &UpdateInput{Description: &same})
		require.NoError(t, err)
		require.Nil(t, env.rules.updated)
	})

	t.Run("invalid definition", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.rule = svbBranchRule()
		_, err := run(env, enum.RuleParentRepo, &UpdateInput{Definition: raw(`{"unknown":1}`)})
		require.Error(t, err)
	})

	t.Run("validator error", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.rule = svbBranchRule()
		env.svc.ruleValidator = svbValidator{err: errSvbBoom}
		_, err := run(env, enum.RuleParentRepo, &UpdateInput{Description: str("new")})
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("backfill error", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.rule = svbBranchRule()
		env.cache.err = errSvbBoom
		_, err := run(env, enum.RuleParentRepo, &UpdateInput{Description: str("new")})
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("store error", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.rule = svbBranchRule()
		env.rules.writeErr = errSvbBoom
		_, err := run(env, enum.RuleParentRepo, &UpdateInput{Description: str("new")})
		require.ErrorIs(t, err, errSvbBoom)
	})
}

func TestSvbList(t *testing.T) {
	ctx := context.Background()
	rules := func() []types.Rule { return []types.Rule{*svbBranchRule()} }

	t.Run("repo not inherited, single page", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.list = rules()
		got, count, err := env.svc.List(ctx, 10, enum.RuleParentRepo, false, &types.RuleFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
		})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, int64(1), count)
		require.Zero(t, env.rules.countCall)
		require.Contains(t, got[0].Users, int64(1))
		require.Contains(t, got[0].Repositories, int64(5))
	})

	t.Run("repo inherited uses count", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.list = rules()
		env.rules.count = 55
		_, count, err := env.svc.List(ctx, 10, enum.RuleParentRepo, true, &types.RuleFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 10}},
		})
		require.NoError(t, err)
		require.Equal(t, int64(55), count)
	})

	t.Run("space inherited and not", func(t *testing.T) {
		for _, inherited := range []bool{true, false} {
			env := svbNewEnv(t)
			env.rules.list = rules()
			_, _, err := env.svc.List(ctx, 10, enum.RuleParentSpace, inherited, &types.RuleFilter{
				ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
			})
			require.NoError(t, err)
		}
	})

	t.Run("unsupported parent", func(t *testing.T) {
		env := svbNewEnv(t)
		_, _, err := env.svc.List(ctx, 10, enum.RuleParent("other"), false, &types.RuleFilter{})
		require.Error(t, err)
	})

	t.Run("lookup errors", func(t *testing.T) {
		env := svbNewEnv(t)
		env.repos.err = errSvbBoom
		_, _, err := env.svc.List(ctx, 10, enum.RuleParentRepo, true, &types.RuleFilter{})
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(t)
		env.space.err = errSvbBoom
		_, _, err = env.svc.List(ctx, 10, enum.RuleParentRepo, true, &types.RuleFilter{})
		require.ErrorIs(t, err, errSvbBoom)
		_, _, err = env.svc.List(ctx, 10, enum.RuleParentSpace, true, &types.RuleFilter{})
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("store errors", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.listErr = errSvbBoom
		_, _, err := env.svc.List(ctx, 10, enum.RuleParentSpace, false, &types.RuleFilter{})
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(t)
		env.rules.countErr = errSvbBoom
		_, _, err = env.svc.List(ctx, 10, enum.RuleParentSpace, false, &types.RuleFilter{})
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("rule enrichment errors", func(t *testing.T) {
		env := svbNewEnv(t)
		env.rules.list = rules()
		env.cache.err = errSvbBoom
		_, _, err := env.svc.List(ctx, 10, enum.RuleParentSpace, false, &types.RuleFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
		})
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(t)
		bad := svbBranchRule()
		bad.Definition = json.RawMessage(`bad`)
		env.rules.list = []types.Rule{*bad}
		_, _, err = env.svc.List(ctx, 10, enum.RuleParentSpace, false, &types.RuleFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
		})
		require.Error(t, err)
	})
}

func TestSvbCommonHelpers(t *testing.T) {
	require.Equal(t, audit.ResourceTypeBranchRule, ruleTypeToResourceType(protection.TypeBranch))
	require.Equal(t, audit.ResourceTypeTagRule, ruleTypeToResourceType(protection.TypeTag))
	require.Equal(t, audit.ResourceTypePushRule, ruleTypeToResourceType(protection.TypePush))
	require.Equal(t, audit.ResourceTypeBranchRule, ruleTypeToResourceType(enum.RuleType("other")))

	ctx := context.Background()
	env := svbNewEnv(t)
	env.svc.sendSSE(ctx, 5, enum.RuleParentRepo, enum.SSETypeRuleCreated, &types.Rule{})
	env.repos.err = errSvbBoom
	env.svc.sendSSE(ctx, 5, enum.RuleParentRepo, enum.SSETypeRuleCreated, &types.Rule{})
	env.sse.AssertNumberOfCalls(t, "Publish", 1)

	// User group lookup with groups present and failing store.
	env = svbNewEnv(t)
	def := &protection.Branch{}
	def.Bypass.UserGroupIDs = []int64{4}
	groups, ids, err := env.svc.getRuleUserGroups(ctx, def)
	require.NoError(t, err)
	require.Equal(t, []int64{4}, ids)
	require.Contains(t, groups, int64(4))

	env.svc.userGroupStore = &svbUserGroupStore{err: errSvbBoom}
	_, _, err = env.svc.getRuleUserGroups(ctx, def)
	require.ErrorIs(t, err, errSvbBoom)

	rule := svbBranchRule()
	rule.Definition = json.RawMessage(`{"bypass":{"user_group_ids":[4]}}`)
	_, _, _, _, err = env.svc.getRuleUserAndUserGroups(ctx, rule)
	require.ErrorContains(t, err, "user groups")
}
