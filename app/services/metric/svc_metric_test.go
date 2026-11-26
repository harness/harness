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

package metric

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	repoevents "github.com/harness/gitness/app/events/repo"
	ruleevents "github.com/harness/gitness/app/events/rule"
	userevents "github.com/harness/gitness/app/events/user"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/events"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/posthog/posthog-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type svcSubmission struct {
	user       *types.PrincipalInfo
	object     Object
	verb       Verb
	properties map[string]any
}

type svcSubmitter struct {
	submissions []svcSubmission
	err         error
}

func (s *svcSubmitter) SubmitGroups(context.Context) error { return nil }

func (s *svcSubmitter) Submit(
	_ context.Context, user *types.PrincipalInfo, object Object, verb Verb, props map[string]any,
) error {
	s.submissions = append(s.submissions, svcSubmission{user, object, verb, props})
	return s.err
}

func (s *svcSubmitter) last(t *testing.T) svcSubmission {
	t.Helper()
	require.NotEmpty(t, s.submissions)
	return s.submissions[len(s.submissions)-1]
}

type svcCache[K comparable, V any] struct {
	values map[K]V
}

func (c svcCache[K, V]) Stats() (int64, int64)    { return 0, 0 }
func (c svcCache[K, V]) Evict(context.Context, K) {}
func (c svcCache[K, V]) Get(_ context.Context, key K) (V, error) {
	v, ok := c.values[key]
	if !ok {
		var zero V
		return zero, gitnessstore.ErrResourceNotFound
	}
	return v, nil
}

type svcPrincipalCache struct {
	svcCache[int64, *types.PrincipalInfo]
}

func (c svcPrincipalCache) Map(context.Context, []int64) (map[int64]*types.PrincipalInfo, error) {
	return c.values, nil
}

type svcPublicAccess struct {
	publicaccess.Service
	public map[string]bool
	err    error
}

func (s svcPublicAccess) Get(_ context.Context, _ enum.PublicResourceType, path string) (bool, error) {
	return s.public[path], s.err
}

type svcPRStore struct {
	store.PullReqStore
	prs map[int64]*types.PullReq
}

func (s svcPRStore) Find(_ context.Context, id int64) (*types.PullReq, error) {
	pr, ok := s.prs[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return pr, nil
}

type svcRuleStore struct {
	store.RuleStore
	rules map[int64]*types.Rule
}

func (s svcRuleStore) Find(_ context.Context, id int64) (*types.Rule, error) {
	r, ok := s.rules[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return r, nil
}

type svcRepoStore struct {
	store.RepoStore
	deleted *types.Repository
	err     error
}

func (s svcRepoStore) FindDeleted(context.Context, int64, *int64) (*types.Repository, error) {
	return s.deleted, s.err
}

type svcEnv struct {
	submitter    *svcSubmitter
	principals   svcPrincipalCache
	repoFinder   refcache.RepoFinder
	spaceFinder  refcache.SpaceFinder
	publicAccess svcPublicAccess
	prStore      svcPRStore
	ruleStore    svcRuleStore
	repoCache    svcCache[int64, *types.RepositoryCore]
}

func svcNewEnv() *svcEnv {
	srcRepoID := int64(2)
	_ = srcRepoID
	repoCache := svcCache[int64, *types.RepositoryCore]{values: map[int64]*types.RepositoryCore{
		1: {ID: 1, ParentID: 10, Identifier: "repo", Path: "root/repo"},
		2: {ID: 2, ParentID: 10, Identifier: "fork", Path: "root/fork"},
	}}
	env := &svcEnv{
		submitter: &svcSubmitter{},
		principals: svcPrincipalCache{svcCache[int64, *types.PrincipalInfo]{
			values: map[int64]*types.PrincipalInfo{
				5: {ID: 5, UID: "alice", Email: "alice@example.com"},
				6: {ID: 6, UID: "bob", Email: "bob@example.com"},
			}}},
		publicAccess: svcPublicAccess{public: map[string]bool{"root/repo": true, "root": true}},
		repoCache:    repoCache,
		prStore:      svcPRStore{prs: map[int64]*types.PullReq{}},
		ruleStore:    svcRuleStore{rules: map[int64]*types.Rule{}},
	}
	env.repoFinder = refcache.NewRepoFinder(
		svcRepoStore{deleted: &types.Repository{ID: 9, Identifier: "gone", Path: "root/gone"}},
		nil, repoCache, nil, cache.Evictor[*types.RepositoryCore]{},
	)
	env.spaceFinder = refcache.NewSpaceFinder(
		svcCache[int64, *types.SpaceCore]{values: map[int64]*types.SpaceCore{
			10: {ID: 10, ParentID: 1, Identifier: "root", Path: "root"},
		}}, nil, nil, cache.Evictor[*types.SpaceCore]{})
	return env
}

func svcEvt[T any](p T) *events.Event[T] { return &events.Event[T]{ID: "e", Payload: p} }

func TestSvcPrepareProps(t *testing.T) {
	existing := map[string]any{"a": 1}
	assert.Equal(t, existing, prepareProps(existing))
	fresh := prepareProps(nil)
	assert.NotNil(t, fresh)
	assert.Empty(t, fresh)
}

func TestSvcUserHandlers(t *testing.T) {
	ctx := context.Background()

	t.Run("create reports the creator", func(t *testing.T) {
		env := svcNewEnv()
		h := handlersUser{principalInfoCache: env.principals, submitter: env.submitter}
		require.NoError(t, h.Create(ctx, svcEvt(&userevents.CreatedPayload{
			Base: userevents.Base{PrincipalID: 5}, CreatedPrincipalID: 6,
		})))
		got := env.submitter.last(t)
		assert.Equal(t, ObjectUser, got.object)
		assert.Equal(t, VerbUserCreate, got.verb)
		assert.Equal(t, "bob", got.user.UID)
		assert.Equal(t, "bob", got.properties[userName])
		assert.Equal(t, "alice", got.properties[userCreatedByName])
		assert.Equal(t, "alice@example.com", got.properties[userCreatedByEmail])
		assert.Equal(t, int64(5), got.properties[userCreatedByID])
	})
	t.Run("register and login", func(t *testing.T) {
		env := svcNewEnv()
		h := handlersUser{principalInfoCache: env.principals, submitter: env.submitter}
		require.NoError(t, h.Register(ctx, svcEvt(&userevents.RegisteredPayload{Base: userevents.Base{PrincipalID: 5}})))
		assert.Equal(t, VerbUserCreate, env.submitter.last(t).verb)
		require.NoError(t, h.Login(ctx, svcEvt(&userevents.LoggedInPayload{Base: userevents.Base{PrincipalID: 5}})))
		got := env.submitter.last(t)
		assert.Equal(t, VerbUserLogin, got.verb)
		assert.Equal(t, "alice@example.com", got.properties[userEmail])
	})
	t.Run("failures", func(t *testing.T) {
		env := svcNewEnv()
		h := handlersUser{principalInfoCache: env.principals, submitter: env.submitter}
		require.Error(t, h.Login(ctx, svcEvt(&userevents.LoggedInPayload{Base: userevents.Base{PrincipalID: 99}})))
		require.Error(t, h.Create(ctx, svcEvt(&userevents.CreatedPayload{
			Base: userevents.Base{PrincipalID: 99}, CreatedPrincipalID: 5,
		})))

		env.submitter.err = errors.New("boom")
		require.Error(t, h.Login(ctx, svcEvt(&userevents.LoggedInPayload{Base: userevents.Base{PrincipalID: 5}})))
	})
}

func TestSvcRepoHandlers(t *testing.T) {
	ctx := context.Background()
	newHandlers := func(env *svcEnv) handlersRepo {
		return handlersRepo{
			principalInfoCache: env.principals, repoFinder: env.repoFinder,
			publicAccess: env.publicAccess, submitter: env.submitter,
		}
	}

	t.Run("create sets privacy and import info", func(t *testing.T) {
		env := svcNewEnv()
		h := newHandlers(env)
		require.NoError(t, h.Create(ctx, svcEvt(&repoevents.CreatedPayload{
			Base: repoevents.Base{RepoID: 1, PrincipalID: 5}, IsPublic: false, IsMigrated: true, ImportedFrom: "github",
		})))
		got := env.submitter.last(t)
		assert.Equal(t, ObjectRepository, got.object)
		assert.Equal(t, VerbRepoCreate, got.verb)
		assert.Equal(t, true, got.properties[repoPrivate])
		assert.Equal(t, true, got.properties[repoMigrated])
		assert.Equal(t, true, got.properties[repoImported])
		assert.Equal(t, "github", got.properties[repoImportedFrom])
		assert.Equal(t, "root/repo", got.properties[repoPath])
	})
	t.Run("push derives privacy from public access", func(t *testing.T) {
		env := svcNewEnv()
		h := newHandlers(env)
		require.NoError(t, h.Push(ctx, svcEvt(&repoevents.PushedPayload{Base: repoevents.Base{RepoID: 1, PrincipalID: 5}})))
		got := env.submitter.last(t)
		assert.Equal(t, VerbRepoPush, got.verb)
		assert.Equal(t, false, got.properties[repoPrivate], "root/repo is public")

		require.NoError(t, h.Push(ctx, svcEvt(&repoevents.PushedPayload{Base: repoevents.Base{RepoID: 2, PrincipalID: 5}})))
		assert.Equal(t, true, env.submitter.last(t).properties[repoPrivate], "root/fork is not public")
	})
	t.Run("soft delete uses the deleted repository", func(t *testing.T) {
		env := svcNewEnv()
		h := newHandlers(env)
		require.NoError(t, h.SoftDelete(ctx, svcEvt(&repoevents.SoftDeletedPayload{
			Base: repoevents.Base{PrincipalID: 5}, RepoPath: "9", Deleted: 1,
		})))
		got := env.submitter.last(t)
		assert.Equal(t, VerbRepoDelete, got.verb)
		assert.Equal(t, "gone", got.properties[repoName])
		assert.NotContains(t, got.properties, repoPrivate, "no public access lookup for deleted repositories")
	})
	t.Run("failures", func(t *testing.T) {
		env := svcNewEnv()
		h := newHandlers(env)
		require.Error(t, h.Push(ctx, svcEvt(&repoevents.PushedPayload{Base: repoevents.Base{RepoID: 99, PrincipalID: 5}})))
		require.Error(t, h.Push(ctx, svcEvt(&repoevents.PushedPayload{Base: repoevents.Base{RepoID: 1, PrincipalID: 99}})))

		env.publicAccess.err = errors.New("boom")
		h = newHandlers(env)
		require.Error(t, h.Push(ctx, svcEvt(&repoevents.PushedPayload{Base: repoevents.Base{RepoID: 1, PrincipalID: 5}})))

		env = svcNewEnv()
		env.submitter.err = errors.New("boom")
		require.Error(t, newHandlers(env).Push(ctx, svcEvt(&repoevents.PushedPayload{
			Base: repoevents.Base{RepoID: 1, PrincipalID: 5},
		})))

		env = svcNewEnv()
		env.repoFinder = refcache.NewRepoFinder(svcRepoStore{err: errors.New("boom")},
			nil, env.repoCache, nil, cache.Evictor[*types.RepositoryCore]{})
		require.Error(t, newHandlers(env).SoftDelete(ctx, svcEvt(&repoevents.SoftDeletedPayload{
			Base: repoevents.Base{PrincipalID: 5}, RepoPath: "1", Deleted: 1,
		})))
	})
}

func TestSvcPullReqHandlers(t *testing.T) {
	ctx := context.Background()
	fork := int64(2)
	missing := int64(77)
	squash := enum.MergeMethodSquash

	setup := func() (*svcEnv, handlersPullReq) {
		env := svcNewEnv()
		env.prStore.prs[100] = &types.PullReq{
			ID: 100, Number: 7, TargetRepoID: 1, SourceRepoID: &fork, SourceBranch: "feat", TargetBranch: "main",
			MergeMethod: &squash,
		}
		return env, handlersPullReq{
			principalInfoCache: env.principals, repoFinder: env.repoFinder, pullReqStore: env.prStore,
			publicAccess: env.publicAccess, submitter: env.submitter,
		}
	}
	base := pullreqevents.Base{PullReqID: 100, PrincipalID: 5}

	tests := []struct {
		name   string
		invoke func(h handlersPullReq) error
		verb   Verb
	}{
		{"create", func(h handlersPullReq) error {
			return h.Create(ctx, svcEvt(&pullreqevents.CreatedPayload{Base: base}))
		}, VerbPullReqCreate},
		{"close", func(h handlersPullReq) error {
			return h.Close(ctx, svcEvt(&pullreqevents.ClosedPayload{Base: base}))
		}, VerbPullReqClose},
		{"reopen", func(h handlersPullReq) error {
			return h.Reopen(ctx, svcEvt(&pullreqevents.ReopenedPayload{Base: base}))
		}, VerbPullReqReopen},
		{"merge", func(h handlersPullReq) error {
			return h.Merge(ctx, svcEvt(&pullreqevents.MergedPayload{Base: base}))
		}, VerbPullReqMerge},
		{"comment", func(h handlersPullReq) error {
			return h.CommentCreate(ctx, svcEvt(&pullreqevents.CommentCreatedPayload{Base: base, IsReply: true}))
		}, VerbPullReqComment},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, h := setup()
			require.NoError(t, tt.invoke(h))
			got := env.submitter.last(t)
			assert.Equal(t, ObjectPullRequest, got.object)
			assert.Equal(t, tt.verb, got.verb)
			assert.Equal(t, int64(7), got.properties[prNumber])
			assert.Equal(t, "feat", got.properties[prSourceBranch])
			assert.Equal(t, "main", got.properties[prTargetBranch])
			assert.Equal(t, "squash", got.properties[prMergeMethod])
			assert.Equal(t, "root/repo", got.properties[repoPath])
			assert.Equal(t, "root/fork", got.properties[prSourceRepoPath])
			assert.Equal(t, "fork", got.properties[prSourceRepoName])
			assert.Equal(t, int64(2), got.properties[prSourceRepoID])
		})
	}

	t.Run("comment reply flag", func(t *testing.T) {
		env, h := setup()
		require.NoError(t, h.CommentCreate(ctx, svcEvt(&pullreqevents.CommentCreatedPayload{Base: base, IsReply: true})))
		assert.Equal(t, true, env.submitter.last(t).properties[prCommentReply])
	})
	t.Run("same source and target repo has no source info", func(t *testing.T) {
		env, h := setup()
		same := int64(1)
		env.prStore.prs[100].SourceRepoID = &same
		env.prStore.prs[100].MergeMethod = nil
		require.NoError(t, h.Create(ctx, svcEvt(&pullreqevents.CreatedPayload{Base: base})))
		got := env.submitter.last(t)
		assert.NotContains(t, got.properties, prSourceRepoID)
		assert.NotContains(t, got.properties, prMergeMethod)
	})
	t.Run("deleted source repo is tolerated", func(t *testing.T) {
		env, h := setup()
		env.prStore.prs[100].SourceRepoID = &missing
		require.NoError(t, h.Create(ctx, svcEvt(&pullreqevents.CreatedPayload{Base: base})))
		assert.NotContains(t, env.submitter.last(t).properties, prSourceRepoID)
	})
	t.Run("failures", func(t *testing.T) {
		_, h := setup()
		require.Error(t, h.Create(ctx, svcEvt(&pullreqevents.CreatedPayload{
			Base: pullreqevents.Base{PullReqID: 999, PrincipalID: 5},
		})))
		require.Error(t, h.Create(ctx, svcEvt(&pullreqevents.CreatedPayload{
			Base: pullreqevents.Base{PullReqID: 100, PrincipalID: 99},
		})))

		env, h := setup()
		env.prStore.prs[100].TargetRepoID = 99
		require.Error(t, h.Create(ctx, svcEvt(&pullreqevents.CreatedPayload{Base: base})))

		env, h = setup()
		env.submitter.err = errors.New("boom")
		require.Error(t, h.Create(ctx, svcEvt(&pullreqevents.CreatedPayload{Base: base})))
	})
}

func TestSvcRuleHandlers(t *testing.T) {
	ctx := context.Background()
	spaceID := int64(10)
	repoID := int64(1)

	setup := func(rule *types.Rule) (*svcEnv, handlersRule) {
		env := svcNewEnv()
		env.ruleStore.rules[1] = rule
		return env, handlersRule{
			principalInfoCache: env.principals, spaceFinder: env.spaceFinder, repoFinder: env.repoFinder,
			ruleStore: env.ruleStore, publicAccess: env.publicAccess, submitter: env.submitter,
		}
	}
	event := svcEvt(&ruleevents.CreatedPayload{Base: ruleevents.Base{RuleID: 1, PrincipalID: 5}})

	t.Run("space rule", func(t *testing.T) {
		env, h := setup(&types.Rule{Identifier: "r", Type: enum.RuleType("branch"), SpaceID: &spaceID})
		require.NoError(t, h.Create(ctx, event))
		got := env.submitter.last(t)
		assert.Equal(t, ObjectRule, got.object)
		assert.Equal(t, VerbRuleCreate, got.verb)
		assert.Equal(t, "r", got.properties[ruleName])
		assert.Equal(t, "branch", got.properties[ruleType])
		assert.Equal(t, "root", got.properties[spacePath])
		assert.Equal(t, false, got.properties[spacePrivate])
	})
	t.Run("repo rule", func(t *testing.T) {
		env, h := setup(&types.Rule{Identifier: "r", Type: enum.RuleType("branch"), RepoID: &repoID})
		require.NoError(t, h.Create(ctx, event))
		got := env.submitter.last(t)
		assert.Equal(t, "root/repo", got.properties[repoPath])
	})
	t.Run("rule without scope", func(t *testing.T) {
		env, h := setup(&types.Rule{Identifier: "r"})
		require.NoError(t, h.Create(ctx, event))
		assert.NotContains(t, env.submitter.last(t).properties, repoPath)
	})
	t.Run("failures", func(t *testing.T) {
		_, h := setup(&types.Rule{})
		require.Error(t, h.Create(ctx, svcEvt(&ruleevents.CreatedPayload{Base: ruleevents.Base{RuleID: 99, PrincipalID: 5}})))

		missingSpace := int64(404)
		_, h = setup(&types.Rule{SpaceID: &missingSpace})
		require.Error(t, h.Create(ctx, event))

		missingRepo := int64(404)
		_, h = setup(&types.Rule{RepoID: &missingRepo})
		require.Error(t, h.Create(ctx, event))

		_, h = setup(&types.Rule{RepoID: &repoID})
		require.Error(t, h.Create(ctx, svcEvt(&ruleevents.CreatedPayload{Base: ruleevents.Base{RuleID: 1, PrincipalID: 99}})))

		env, h := setup(&types.Rule{RepoID: &repoID})
		env.submitter.err = errors.New("boom")
		require.Error(t, h.Create(ctx, event))

		env, h = setup(&types.Rule{SpaceID: &spaceID})
		env.publicAccess.err = errors.New("boom")
		h.publicAccess = env.publicAccess
		require.Error(t, h.Create(ctx, event))
	})
}

func TestSvcFillData(t *testing.T) {
	ctx := context.Background()

	t.Run("an explicit privacy value is kept without a lookup", func(t *testing.T) {
		props, err := fillRepoData(ctx, map[string]any{repoPrivate: true},
			&types.RepositoryCore{ID: 1, Path: "root/repo"}, svcPublicAccess{err: errors.New("must not be called")})
		require.NoError(t, err)
		assert.Equal(t, true, props[repoPrivate])

		props, err = fillSpaceData(ctx, map[string]any{spacePrivate: false},
			&types.SpaceCore{ID: 1}, svcPublicAccess{err: errors.New("must not be called")})
		require.NoError(t, err)
		assert.Equal(t, false, props[spacePrivate])
	})
	t.Run("a nil public access service skips the lookup", func(t *testing.T) {
		props, err := fillRepoData(ctx, nil, &types.RepositoryCore{ID: 1}, nil)
		require.NoError(t, err)
		assert.NotContains(t, props, repoPrivate)
		props, err = fillSpaceData(ctx, nil, &types.SpaceCore{ID: 1}, nil)
		require.NoError(t, err)
		assert.NotContains(t, props, spacePrivate)
	})
}

type svcSettingsStore struct {
	store.SettingsStore
	values   map[string]json.RawMessage
	findErr  error
	upsertEr error
}

func (s *svcSettingsStore) Find(_ context.Context, _ enum.SettingsScope, _ int64, key string) (json.RawMessage, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	v, ok := s.values[key]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return v, nil
}

func (s *svcSettingsStore) Upsert(
	_ context.Context, _ enum.SettingsScope, _ int64, key string, v json.RawMessage,
) error {
	if s.upsertEr != nil {
		return s.upsertEr
	}
	s.values[key] = v
	return nil
}

func TestSvcNewValues(t *testing.T) {
	ctx := context.Background()
	enabled := &types.Config{InstanceID: "host-1"}
	enabled.Metric.Enabled = true

	t.Run("disabled by config", func(t *testing.T) {
		v, err := NewValues(ctx, &types.Config{}, nil)
		require.NoError(t, err)
		assert.False(t, v.Enabled)
		assert.Empty(t, v.InstallID)
	})
	t.Run("disabled by DO_NOT_TRACK", func(t *testing.T) {
		t.Setenv("DO_NOT_TRACK", "true")
		v, err := NewValues(ctx, enabled, nil)
		require.NoError(t, err)
		assert.False(t, v.Enabled)
	})
	t.Run("generates and persists an install id", func(t *testing.T) {
		t.Setenv("DO_NOT_TRACK", "false")
		st := &svcSettingsStore{values: map[string]json.RawMessage{}}
		srv := settings.NewService(st, refcache.SpaceFinder{})

		v, err := NewValues(ctx, enabled, srv)
		require.NoError(t, err)
		assert.True(t, v.Enabled)
		assert.Equal(t, "host-1", v.Hostname)
		assert.NotEmpty(t, v.InstallID)
		assert.Len(t, st.values, 1)

		again, err := NewValues(ctx, enabled, srv)
		require.NoError(t, err)
		assert.Equal(t, v.InstallID, again.InstallID, "the persisted install id is reused")
	})
	t.Run("settings failures", func(t *testing.T) {
		t.Setenv("DO_NOT_TRACK", "")
		srv := settings.NewService(&svcSettingsStore{values: map[string]json.RawMessage{}, findErr: errors.New("boom")},
			refcache.SpaceFinder{})
		_, err := NewValues(ctx, enabled, srv)
		require.Error(t, err)

		srv = settings.NewService(&svcSettingsStore{values: map[string]json.RawMessage{}, upsertEr: errors.New("boom")},
			refcache.SpaceFinder{})
		_, err = NewValues(ctx, enabled, srv)
		require.Error(t, err)
	})
}

type svcPostHogClient struct {
	posthog.Client
	messages []posthog.Message
	err      error
}

func (c *svcPostHogClient) Enqueue(m posthog.Message) error {
	c.messages = append(c.messages, m)
	return c.err
}

type svcPrincipalStore struct {
	store.PrincipalStore
	users []*types.User
	err   error
}

func (s svcPrincipalStore) ListUsers(context.Context, *types.UserFilter) ([]*types.User, error) {
	return s.users, s.err
}

func TestSvcNewPostHog(t *testing.T) {
	ctx := context.Background()

	t.Run("disabled values", func(t *testing.T) {
		ph, err := NewPostHog(ctx, &types.Config{}, &Values{Enabled: false}, nil, nil)
		require.NoError(t, err)
		assert.Nil(t, ph)

		ph, err = NewPostHog(ctx, &types.Config{}, &Values{Enabled: true}, nil, nil)
		require.NoError(t, err)
		assert.Nil(t, ph, "an empty install id disables the client")
	})
	t.Run("missing api key", func(t *testing.T) {
		ph, err := NewPostHog(ctx, &types.Config{}, &Values{Enabled: true, InstallID: "i"}, nil, nil)
		require.NoError(t, err)
		assert.Nil(t, ph)
	})
}

func TestSvcPostHogSubmit(t *testing.T) {
	ctx := context.Background()
	client := &svcPostHogClient{}
	ph := &PostHog{client: client, installID: "install", hostname: "host"}

	assert.Equal(t, "install:alice", ph.uniqueUserID("alice"))
	require.NoError(t, ph.SubmitGroups(ctx))

	t.Run("nil receiver is a no-op", func(t *testing.T) {
		var disabled *PostHog
		require.NoError(t, disabled.Submit(ctx, nil, ObjectUser, VerbUserLogin, nil))
	})
	t.Run("anonymous event", func(t *testing.T) {
		require.NoError(t, ph.Submit(ctx, nil, ObjectRepository, VerbRepoPush, map[string]any{"k": "v"}))
		capture, ok := client.messages[len(client.messages)-1].(posthog.Capture)
		require.True(t, ok)
		assert.Equal(t, "repo:push", capture.Event)
		assert.Empty(t, capture.DistinctId)
		assert.Equal(t, "v", capture.Properties["k"])
	})
	t.Run("event of a user", func(t *testing.T) {
		user := &types.PrincipalInfo{ID: 5, UID: "alice", Email: "alice@example.com", Type: enum.PrincipalTypeUser}
		require.NoError(t, ph.Submit(ctx, user, ObjectUser, VerbUserLogin, nil))
		capture, ok := client.messages[len(client.messages)-1].(posthog.Capture)
		require.True(t, ok)
		assert.Equal(t, "install:alice", capture.DistinctId)
		assert.Equal(t, "user:login", capture.Event)
		assert.Contains(t, capture.Properties, "$set")
		assert.Contains(t, capture.Properties, "$set_once")
	})
	t.Run("enqueue failure", func(t *testing.T) {
		failing := &PostHog{client: &svcPostHogClient{err: errors.New("boom")}, installID: "i"}
		require.Error(t, failing.Submit(ctx, nil, ObjectUser, VerbUserLogin, nil))
	})
}

func TestSvcPostHogDefaultGroup(t *testing.T) {
	newPostHog := func(users []*types.User, err error, clientErr error) (*PostHog, *svcPostHogClient) {
		client := &svcPostHogClient{err: clientErr}
		return &PostHog{
			client: client, installID: "install", hostname: "host", principalStore: svcPrincipalStore{users: users, err: err},
		}, client
	}

	t.Run("submits the install group", func(t *testing.T) {
		ph, client := newPostHog([]*types.User{{Email: "admin@example.com", Created: 123}}, nil, nil)
		require.NoError(t, ph.submitDefaultGroup(context.Background()))
		require.Len(t, client.messages, 1)
		identify, ok := client.messages[0].(posthog.GroupIdentify)
		require.True(t, ok)
		assert.Equal(t, postHogGroupInstall, identify.Type)
		assert.Equal(t, "install", identify.Key)
		assert.Equal(t, "admin@example.com", identify.Properties["email"])
		assert.Equal(t, "host", identify.Properties["hostname"])
	})
	t.Run("failures", func(t *testing.T) {
		ph, _ := newPostHog(nil, errors.New("boom"), nil)
		require.Error(t, ph.submitDefaultGroup(context.Background()))
		ph, _ = newPostHog(nil, nil, nil)
		require.Error(t, ph.submitDefaultGroup(context.Background()), "no users found")
		ph, _ = newPostHog([]*types.User{{Email: "a"}}, nil, errors.New("boom"))
		require.Error(t, ph.submitDefaultGroup(context.Background()))
	})
	t.Run("once loop stops when the context is canceled", func(t *testing.T) {
		ph, _ := newPostHog(nil, nil, nil)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		done := make(chan struct{})
		go func() {
			ph.submitDefaultGroupOnce(ctx)
			close(done)
		}()
		<-done
	})
}
