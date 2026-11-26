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

package space

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

type svbTx struct{}

func (svbTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

// svbErrs fails the named method when present.
type svbErrs map[string]error

type svbSpaceStore struct {
	store.SpaceStore

	errs        svbErrs
	byRef       map[string]*types.Space
	ancestors   []*types.Space
	root        *types.Space
	descendants []int64
	subSpaces   []*types.Space
	updated     *types.Space
	rootUpdates [][]int64
	softDeleted []int64
	calls       []string
}

func (f *svbSpaceStore) FindForUpdate(_ context.Context, id int64) (*types.Space, error) {
	f.calls = append(f.calls, "FindForUpdate")
	return &types.Space{ID: id}, f.errs["FindForUpdate"]
}

func (f *svbSpaceStore) FindByRef(_ context.Context, ref string) (*types.Space, error) {
	if err := f.errs["FindByRef:"+ref]; err != nil {
		return nil, err
	}
	s, ok := f.byRef[ref]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return s, nil
}

func (f *svbSpaceStore) GetAncestors(context.Context, int64) ([]*types.Space, error) {
	return f.ancestors, f.errs["GetAncestors"]
}

func (f *svbSpaceStore) GetRootSpace(context.Context, int64) (*types.Space, error) {
	return f.root, f.errs["GetRootSpace"]
}

func (f *svbSpaceStore) GetDescendantsIDs(context.Context, int64) ([]int64, error) {
	return f.descendants, f.errs["GetDescendantsIDs"]
}

func (f *svbSpaceStore) Update(_ context.Context, s *types.Space) error {
	f.updated = s
	return f.errs["Update"]
}

func (f *svbSpaceStore) UpdateRootSpace(_ context.Context, ids []int64, _ int64, _ string) error {
	f.rootUpdates = append(f.rootUpdates, ids)
	return f.errs["UpdateRootSpace"]
}

func (f *svbSpaceStore) List(context.Context, int64, *types.SpaceFilter) ([]*types.Space, error) {
	return f.subSpaces, f.errs["List"]
}

func (f *svbSpaceStore) SoftDelete(_ context.Context, s *types.Space, _ int64) error {
	f.softDeleted = append(f.softDeleted, s.ID)
	return f.errs["SoftDelete"]
}

type svbPathStore struct {
	store.SpacePathStore

	errs     svbErrs
	inserted *types.SpacePathSegment
	deleted  []int64
}

func (f *svbPathStore) DeletePrimarySegment(context.Context, int64) error {
	return f.errs["DeletePrimarySegment"]
}

func (f *svbPathStore) InsertSegment(_ context.Context, seg *types.SpacePathSegment) error {
	f.inserted = seg
	return f.errs["InsertSegment"]
}

func (f *svbPathStore) DeletePathsAndDescendandPaths(_ context.Context, id int64) error {
	f.deleted = append(f.deleted, id)
	return f.errs["DeletePathsAndDescendandPaths"]
}

type svbRepoStore struct {
	store.RepoStore

	errs        svbErrs
	repos       map[int64]*types.Repository
	idsByParent []int64
	rootUpdates [][]int64
}

func (f *svbRepoStore) Find(_ context.Context, id int64) (*types.Repository, error) {
	if err := f.errs["Find"]; err != nil {
		return nil, err
	}
	r, ok := f.repos[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return r, nil
}

func (f *svbRepoStore) ListIDsByParentSpaceIDs(context.Context, []int64) ([]int64, error) {
	return f.idsByParent, f.errs["ListIDsByParentSpaceIDs"]
}

func (f *svbRepoStore) UpdateRootSpace(_ context.Context, ids []int64, _ int64, _ string) error {
	f.rootUpdates = append(f.rootUpdates, ids)
	return f.errs["RepoUpdateRootSpace"]
}

func (f *svbRepoStore) UpdateParent(context.Context, int64, int64) (int64, error) {
	return 3, f.errs["RepoUpdateParent"]
}

type svbPullReqStore struct {
	store.PullReqStore

	errs svbErrs
	ids  [][]int64
}

func (f *svbPullReqStore) UpdateRootSpace(_ context.Context, ids []int64, _ int64, _ string) error {
	f.ids = append(f.ids, ids)
	return f.errs["PullReqUpdateRootSpace"]
}

type svbRuleStore struct {
	store.RuleStore

	errs    svbErrs
	rules   []types.Rule
	updated []types.Rule
}

func (f *svbRuleStore) List(context.Context, []types.RuleParentInfo, *types.RuleFilter) ([]types.Rule, error) {
	return f.rules, f.errs["RuleList"]
}

func (f *svbRuleStore) Update(_ context.Context, r *types.Rule) error {
	f.updated = append(f.updated, *r)
	return f.errs["RuleUpdate"]
}

func (f *svbRuleStore) UpdateParentSpace(context.Context, int64, int64) (int64, error) {
	return 5, f.errs["RuleUpdateParent"]
}

type svbLabelStore struct {
	store.LabelStore

	err error
}

func (f *svbLabelStore) UpdateParentSpace(context.Context, int64, int64) (int64, error) {
	return 4, f.err
}

type svbWebhookStore struct {
	store.WebhookStore

	err error
}

func (f *svbWebhookStore) UpdateParentSpace(context.Context, int64, int64) (int64, error) {
	return 6, f.err
}

type svbEncrypter struct {
	encErr error
	decErr error
}

func (e svbEncrypter) Encrypt(s string) ([]byte, error) {
	if e.encErr != nil {
		return nil, e.encErr
	}
	return []byte("enc:" + s), nil
}

func (e svbEncrypter) Decrypt(b []byte) (string, error) {
	if e.decErr != nil {
		return "", e.decErr
	}
	return string(b[len("enc:"):]), nil
}

type svbMover struct {
	out MoveResourcesOutput
	err error
}

func (m svbMover) MoveResources(context.Context, int64, int64) (MoveResourcesOutput, error) {
	return m.out, m.err
}

type svbEnv struct {
	svc    *Service
	spaces *svbSpaceStore
	paths  *svbPathStore
	repos  *svbRepoStore
	pulls  *svbPullReqStore
	rules  *svbRuleStore
}

func svbNewEnv() *svbEnv {
	e := &svbEnv{
		spaces: &svbSpaceStore{errs: svbErrs{}},
		paths:  &svbPathStore{errs: svbErrs{}},
		repos:  &svbRepoStore{errs: svbErrs{}},
		pulls:  &svbPullReqStore{errs: svbErrs{}},
		rules:  &svbRuleStore{errs: svbErrs{}},
	}
	e.spaces.root = &types.Space{ID: 1}
	e.svc = &Service{
		tx:             svbTx{},
		encrypter:      svbEncrypter{},
		repoStore:      e.repos,
		spaceStore:     e.spaces,
		spacePathStore: e.paths,
		rulesStore:     e.rules,
		pullreqStore:   e.pulls,
		resourceMover:  svbMover{},
	}
	return e
}

func svbSession() *auth.Session {
	return &auth.Session{Principal: types.Principal{ID: 11}, Metadata: &auth.EmptyMetadata{}}
}

func TestSvbJobDefRoundTrip(t *testing.T) {
	svc := svbNewEnv().svc
	in := Input{SourceSpacePath: "a/b", DestinationSpacePath: "c/d"}

	def, err := svc.getJobDef(svc.JobUIDFromSpacePath("a/b"), in)
	require.NoError(t, err)
	require.Equal(t, "space-move-a/b", def.UID)
	require.Equal(t, jobType, def.Type)
	require.Equal(t, moveJobMaxRetries, def.MaxRetries)

	got, err := svc.getJobInput(def.Data)
	require.NoError(t, err)
	require.Equal(t, in, got)

	svc.encrypter = svbEncrypter{encErr: errSvbBoom}
	_, err = svc.getJobDef("x", in)
	require.ErrorIs(t, err, errSvbBoom)

	svc.encrypter = svbEncrypter{decErr: errSvbBoom}
	_, err = svc.getJobInput(def.Data)
	require.ErrorIs(t, err, errSvbBoom)

	_, err = svc.getJobInput("%%%not-base64")
	require.Error(t, err)

	svc.encrypter = svbEncrypter{}
	_, err = svc.getJobInput("ZW5jOm5vdC1qc29u") // base64("enc:not-json")
	require.Error(t, err)
}

func TestSvbHandleValidation(t *testing.T) {
	ctx := context.Background()
	env := svbNewEnv()
	svc := env.svc

	handle := func(in Input) error {
		def, err := svc.getJobDef("uid", in)
		require.NoError(t, err)
		_, err = svc.Handle(ctx, def.Data, nil)
		return err
	}

	_, err := svc.Handle(ctx, "%%%", nil)
	require.Error(t, err)
	require.ErrorContains(t, handle(Input{DestinationSpacePath: "d"}), "source space path is required")
	require.ErrorContains(t, handle(Input{SourceSpacePath: "s"}), "destination space path is required")

	// Source space missing: nothing to move.
	require.NoError(t, handle(Input{SourceSpacePath: "s", DestinationSpacePath: "d"}))

	env.spaces.errs["FindByRef:s"] = errSvbBoom
	require.ErrorIs(t, handle(Input{SourceSpacePath: "s", DestinationSpacePath: "d"}), errSvbBoom)
	delete(env.spaces.errs, "FindByRef:s")

	env.spaces.byRef = map[string]*types.Space{"s": {ID: 2, Identifier: "s"}}
	env.spaces.errs["FindByRef:d"] = errSvbBoom
	require.ErrorContains(t, handle(Input{SourceSpacePath: "s", DestinationSpacePath: "d"}),
		"failed to find destination space")
	delete(env.spaces.errs, "FindByRef:d")

	// Destination missing and path without a parent: DisectLeaf fails.
	require.Error(t, handle(Input{SourceSpacePath: "s", DestinationSpacePath: ""}))

	// Destination exists: resources move, which fails on lock/identity checks.
	env.spaces.byRef["d"] = &types.Space{ID: 2}
	require.ErrorContains(t, handle(Input{SourceSpacePath: "s", DestinationSpacePath: "d"}),
		"cannot be the same")
}

func TestSvbMoveNoAuth(t *testing.T) {
	ctx := context.Background()
	newName := "renamed"

	setup := func() (*svbEnv, *types.Space) {
		env := svbNewEnv()
		env.spaces.byRef = map[string]*types.Space{
			"parent": {ID: 20, RootSpaceID: 99, RootSpaceIdentifier: "root99"},
		}
		env.spaces.ancestors = []*types.Space{{ID: 1}, {ID: 20}}
		env.spaces.descendants = []int64{7, 8}
		env.repos.idsByParent = []int64{100}
		return env, &types.Space{ID: 7, Identifier: "old", RootSpaceID: 5}
	}

	t.Run("move with rename and root change", func(t *testing.T) {
		env, space := setup()
		err := env.svc.MoveNoAuth(ctx, svbSession(), space, &newName, "parent")
		require.NoError(t, err)
		require.Equal(t, "renamed", space.Identifier)
		require.Equal(t, int64(20), space.ParentID)
		require.Equal(t, int64(99), space.RootSpaceID)
		require.Equal(t, "root99", space.RootSpaceIdentifier)
		require.True(t, env.paths.inserted.IsPrimary)
		require.Equal(t, int64(11), env.paths.inserted.CreatedBy)
		require.Equal(t, "renamed", env.paths.inserted.Identifier)
		require.Equal(t, [][]int64{{7, 8}}, env.spaces.rootUpdates)
		require.Equal(t, [][]int64{{100}}, env.repos.rootUpdates)
		require.Equal(t, [][]int64{{100}}, env.pulls.ids)
	})

	t.Run("same root skips propagation", func(t *testing.T) {
		env, space := setup()
		space.RootSpaceID = 99
		require.NoError(t, env.svc.MoveNoAuth(ctx, svbSession(), space, nil, "parent"))
		require.Equal(t, "old", space.Identifier)
		require.Empty(t, env.spaces.rootUpdates)
	})

	failures := []string{
		"FindForUpdate", "FindByRef:parent", "DeletePrimarySegment", "InsertSegment", "GetAncestors", "Update",
		"GetDescendantsIDs", "ListIDsByParentSpaceIDs", "UpdateRootSpace", "RepoUpdateRootSpace",
		"PullReqUpdateRootSpace",
	}
	for _, fail := range failures {
		t.Run("fails on "+fail, func(t *testing.T) {
			env, space := setup()
			err := errSvbBoom
			env.spaces.errs[fail] = err
			env.paths.errs[fail] = err
			env.repos.errs[fail] = err
			env.pulls.errs[fail] = err
			require.ErrorIs(t, env.svc.MoveNoAuth(ctx, svbSession(), space, nil, "parent"), errSvbBoom)
		})
	}
}

func TestSvbMoveSpaceResourcesInTxErrors(t *testing.T) {
	ctx := context.Background()
	src := &types.Space{ID: 1, RootSpaceID: 10}
	dst := &types.Space{ID: 2, RootSpaceID: 20, RootSpaceIdentifier: "r20"}

	t.Run("same space", func(t *testing.T) {
		out, err := svbNewEnv().svc.moveSpaceResourcesInTx(ctx, src, src)
		require.ErrorContains(t, err, "cannot be the same")
		require.Equal(t, MoveResourcesOutput{}, out)
	})

	t.Run("lock failure", func(t *testing.T) {
		env := svbNewEnv()
		env.spaces.errs["FindForUpdate"] = errSvbBoom
		out, err := env.svc.moveSpaceResourcesInTx(ctx, src, dst)
		require.ErrorIs(t, err, errSvbBoom)
		require.Equal(t, MoveResourcesOutput{}, out)
	})

	t.Run("list repos failure", func(t *testing.T) {
		env := svbNewEnv()
		env.repos.errs["ListIDsByParentSpaceIDs"] = errSvbBoom
		_, err := env.svc.moveSpaceResourcesInTx(ctx, src, dst)
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("mover failure", func(t *testing.T) {
		env := svbNewEnv()
		env.svc.resourceMover = svbMover{err: errSvbBoom}
		_, err := env.svc.moveSpaceResourcesInTx(ctx, src, dst)
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("root space propagation failures", func(t *testing.T) {
		env := svbNewEnv()
		env.repos.idsByParent = []int64{5}
		env.repos.errs["RepoUpdateRootSpace"] = errSvbBoom
		_, err := env.svc.moveSpaceResourcesInTx(ctx, src, dst)
		require.ErrorContains(t, err, "moved repos root space")

		env = svbNewEnv()
		env.pulls.errs["PullReqUpdateRootSpace"] = errSvbBoom
		_, err = env.svc.moveSpaceResourcesInTx(ctx, src, dst)
		require.ErrorContains(t, err, "moved pull requests root space")
	})

	t.Run("cleanup failure", func(t *testing.T) {
		env := svbNewEnv()
		env.spaces.errs["GetAncestors"] = errSvbBoom
		_, err := env.svc.moveSpaceResourcesInTx(ctx, src, dst)
		require.ErrorContains(t, err, "clean up parent space resources")
	})
}

func TestSvbCleanUpStaleSpaceResources(t *testing.T) {
	ctx := context.Background()
	space := &types.Space{ID: 7}

	target := func(include, exclude []int64) json.RawMessage {
		b, err := json.Marshal(map[string]any{
			"include": map[string]any{"ids": include},
			"exclude": map[string]any{"ids": exclude},
		})
		require.NoError(t, err)
		return b
	}

	setup := func() *svbEnv {
		env := svbNewEnv()
		env.spaces.root = &types.Space{ID: 1}
		// root (skipped), the space itself (skipped) and one real ancestor.
		env.spaces.ancestors = []*types.Space{{ID: 1}, {ID: 7}, {ID: 3}}
		env.spaces.descendants = []int64{8}
		env.repos.repos = map[int64]*types.Repository{
			10: {ID: 10, ParentID: 7}, // in the moved subtree
			11: {ID: 11, ParentID: 8}, // in the moved subtree (descendant)
			12: {ID: 12, ParentID: 3}, // stays
		}
		return env
	}

	t.Run("removes repos of moved subtree", func(t *testing.T) {
		env := setup()
		env.rules.rules = []types.Rule{
			{ID: 1, RepoTarget: target([]int64{10, 12}, []int64{11, 13})},
			{ID: 2, RepoTarget: target([]int64{12}, nil)},
			{ID: 3},
		}
		require.NoError(t, env.svc.cleanUpStaleSpaceResources(ctx, space))
		require.Len(t, env.rules.updated, 1)
		require.Equal(t, int64(1), env.rules.updated[0].ID)

		var got struct {
			Include struct {
				IDs []int64 `json:"ids"`
			} `json:"include"`
			Exclude struct {
				IDs []int64 `json:"ids"`
			} `json:"exclude"`
		}
		require.NoError(t, json.Unmarshal(env.rules.updated[0].RepoTarget, &got))
		require.Equal(t, []int64{12}, got.Include.IDs)
		require.Empty(t, got.Exclude.IDs)
	})

	for _, fail := range []string{"GetAncestors", "GetRootSpace", "GetDescendantsIDs", "RuleList"} {
		t.Run("fails on "+fail, func(t *testing.T) {
			env := setup()
			env.spaces.errs[fail] = errSvbBoom
			env.rules.errs[fail] = errSvbBoom
			require.ErrorIs(t, env.svc.cleanUpStaleSpaceResources(ctx, space), errSvbBoom)
		})
	}

	t.Run("rule update failure", func(t *testing.T) {
		env := setup()
		env.rules.rules = []types.Rule{{ID: 1, RepoTarget: target([]int64{10}, nil)}}
		env.rules.errs["RuleUpdate"] = errSvbBoom
		require.ErrorIs(t, env.svc.cleanUpStaleSpaceResources(ctx, space), errSvbBoom)
	})

	t.Run("bad rule target", func(t *testing.T) {
		env := setup()
		env.rules.rules = []types.Rule{{ID: 1, RepoTarget: json.RawMessage(`bad`)}}
		require.ErrorContains(t, env.svc.cleanUpStaleSpaceResources(ctx, space), "failed to clean up rule 1")
	})

	t.Run("repo lookup failure", func(t *testing.T) {
		env := setup()
		env.repos.errs["Find"] = errSvbBoom
		env.rules.rules = []types.Rule{{ID: 1, RepoTarget: target([]int64{10}, nil)}}
		require.ErrorIs(t, env.svc.cleanUpStaleSpaceResources(ctx, space), errSvbBoom)

		env = setup()
		env.repos.errs["Find"] = errSvbBoom
		env.rules.rules = []types.Rule{{ID: 1, RepoTarget: target(nil, []int64{10})}}
		require.ErrorIs(t, env.svc.cleanUpStaleSpaceResources(ctx, space), errSvbBoom)
	})
}

func TestSvbSoftDeleteInner(t *testing.T) {
	ctx := context.Background()
	space := &types.Space{ID: 7}

	t.Run("deletes subspaces then the space", func(t *testing.T) {
		env := svbNewEnv()
		env.spaces.subSpaces = []*types.Space{{ID: 8}, {ID: 9}}
		require.NoError(t, env.svc.SoftDeleteInner(ctx, svbSession(), space, 123))
		require.Equal(t, []int64{8, 9, 7}, env.spaces.softDeleted)
		require.Equal(t, []int64{7}, env.paths.deleted)
	})

	for _, fail := range []string{"List", "FindForUpdate", "SoftDelete", "DeletePathsAndDescendandPaths"} {
		t.Run("fails on "+fail, func(t *testing.T) {
			env := svbNewEnv()
			env.spaces.subSpaces = []*types.Space{{ID: 8}}
			env.spaces.errs[fail] = errSvbBoom
			env.paths.errs[fail] = errSvbBoom
			require.ErrorIs(t, env.svc.SoftDeleteInner(ctx, svbSession(), space, 123), errSvbBoom)
		})
	}
}

func TestSvbCodeResourceMover(t *testing.T) {
	ctx := context.Background()
	mk := func(env *svbEnv, labelErr, webhookErr error) *CodeResourceMover {
		return NewCodeResourceMover(env.repos, &svbLabelStore{err: labelErr}, env.rules, &svbWebhookStore{err: webhookErr})
	}

	env := svbNewEnv()
	out, err := mk(env, nil, nil).MoveResources(ctx, 1, 2)
	require.NoError(t, err)
	require.Equal(t, int64(3), out.RepoCount)
	require.Equal(t, int64(4), out.LabelCount)
	require.Equal(t, int64(5), out.RuleCount)
	require.Equal(t, int64(6), out.WebhookCount)

	env.repos.errs["RepoUpdateParent"] = errSvbBoom
	_, err = mk(env, nil, nil).MoveResources(ctx, 1, 2)
	require.ErrorContains(t, err, "failed to move repos")
	delete(env.repos.errs, "RepoUpdateParent")

	_, err = mk(env, errSvbBoom, nil).MoveResources(ctx, 1, 2)
	require.ErrorContains(t, err, "failed to update labels")

	env.rules.errs["RuleUpdateParent"] = errSvbBoom
	_, err = mk(env, nil, nil).MoveResources(ctx, 1, 2)
	require.ErrorContains(t, err, "failed to update rules")
	delete(env.rules.errs, "RuleUpdateParent")

	_, err = mk(env, nil, errSvbBoom).MoveResources(ctx, 1, 2)
	require.ErrorContains(t, err, "failed to update webhooks")

	require.NotNil(t, ProvideCodeResourceMover(env.repos, &svbLabelStore{}, env.rules, &svbWebhookStore{}))
}

func TestSvbNoopResourceMover(t *testing.T) {
	mover := ProvideNoopResourceMover()
	_, err := mover.MoveResources(context.Background(), 1, 2)
	require.Error(t, err)
}

func TestSvbNewService(t *testing.T) {
	require.NotNil(t, NewService(nil, nil, nil, nil, nil, nil, nil, nil, nil, refcache.SpaceFinder{}, nil, nil, nil))
}
