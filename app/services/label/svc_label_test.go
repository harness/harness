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

package label

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	gitnesserrors "github.com/harness/gitness/errors"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type svcTx struct{}

func (svcTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

// svcLabelStore is an in-memory store.LabelStore.
type svcLabelStore struct {
	store.LabelStore
	labels    map[int64]*types.Label
	nextID    int64
	defineErr error
	updateErr error
	listErr   error
	countErr  error
	deleted   []string
	listedIn  []int64
	increment []int
}

func newSvcLabelStore() *svcLabelStore {
	return &svcLabelStore{labels: map[int64]*types.Label{}}
}

func (s *svcLabelStore) add(l *types.Label) *types.Label {
	s.nextID++
	l.ID = s.nextID
	s.labels[l.ID] = l
	return l
}

func (s *svcLabelStore) Define(_ context.Context, l *types.Label) error {
	if s.defineErr != nil {
		return s.defineErr
	}
	s.add(l)
	return nil
}

func (s *svcLabelStore) Update(_ context.Context, l *types.Label) error { return s.updateErr }

func (s *svcLabelStore) Find(_ context.Context, spaceID, repoID *int64, key string) (*types.Label, error) {
	for _, l := range s.labels {
		sameScope := (repoID != nil && l.RepoID != nil && *l.RepoID == *repoID) ||
			(spaceID != nil && l.SpaceID != nil && *l.SpaceID == *spaceID)
		if sameScope && l.Key == key {
			return l, nil
		}
	}
	return nil, gitnessstore.ErrResourceNotFound
}

func (s *svcLabelStore) FindByID(_ context.Context, id int64) (*types.Label, error) {
	l, ok := s.labels[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return l, nil
}

func (s *svcLabelStore) Delete(_ context.Context, _, _ *int64, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

func (s *svcLabelStore) sorted() []*types.Label {
	res := make([]*types.Label, 0, len(s.labels))
	for _, l := range s.labels {
		res = append(res, l)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res
}

func (s *svcLabelStore) List(_ context.Context, _, _ *int64, _ *types.LabelFilter) ([]*types.Label, error) {
	return s.sorted(), s.listErr
}

func (s *svcLabelStore) ListInScopes(
	_ context.Context, repoID int64, spaceIDs []int64, _ *types.LabelFilter,
) ([]*types.Label, error) {
	s.listedIn = append([]int64{repoID}, spaceIDs...)
	return s.sorted(), s.listErr
}

func (s *svcLabelStore) CountInRepo(context.Context, int64, *types.LabelFilter) (int64, error) {
	return int64(len(s.labels)), s.countErr
}

func (s *svcLabelStore) CountInSpace(context.Context, int64, *types.LabelFilter) (int64, error) {
	return int64(len(s.labels)), s.countErr
}

func (s *svcLabelStore) CountInScopes(context.Context, int64, []int64, *types.LabelFilter) (int64, error) {
	return int64(len(s.labels)), s.countErr
}

func (s *svcLabelStore) ListInfosInScopes(
	_ context.Context, _ int64, _ []int64, _ *types.AssignableLabelFilter,
) ([]*types.LabelInfo, error) {
	var infos []*types.LabelInfo
	for _, l := range s.sorted() {
		infos = append(infos, &types.LabelInfo{
			ID: l.ID, Key: l.Key, Scope: l.Scope, SpaceID: l.SpaceID, RepoID: l.RepoID, Type: l.Type,
		})
	}
	return infos, s.listErr
}

func (s *svcLabelStore) IncrementValueCount(_ context.Context, id int64, inc int) (int64, error) {
	s.increment = append(s.increment, inc)
	l, ok := s.labels[id]
	if !ok {
		return 0, gitnessstore.ErrResourceNotFound
	}
	l.ValueCount += int64(inc)
	return l.ValueCount, nil
}

// svcValueStore is an in-memory store.LabelValueStore.
type svcValueStore struct {
	store.LabelValueStore
	values    map[int64]*types.LabelValue
	nextID    int64
	defineErr error
	deleteErr error
	listErr   error
	countErr  error
	deleted   []string
}

func newSvcValueStore() *svcValueStore {
	return &svcValueStore{values: map[int64]*types.LabelValue{}}
}

func (s *svcValueStore) add(v *types.LabelValue) *types.LabelValue {
	s.nextID++
	v.ID = s.nextID
	s.values[v.ID] = v
	return v
}

func (s *svcValueStore) Define(_ context.Context, v *types.LabelValue) error {
	if s.defineErr != nil {
		return s.defineErr
	}
	s.add(v)
	return nil
}

func (s *svcValueStore) Update(context.Context, *types.LabelValue) error { return nil }

func (s *svcValueStore) Delete(_ context.Context, _ int64, value string) error {
	s.deleted = append(s.deleted, value)
	return s.deleteErr
}

func (s *svcValueStore) DeleteMany(_ context.Context, _ int64, values []string) error {
	s.deleted = append(s.deleted, values...)
	return s.deleteErr
}

func (s *svcValueStore) FindByLabelID(_ context.Context, labelID int64, value string) (*types.LabelValue, error) {
	for _, v := range s.values {
		if v.LabelID == labelID && v.Value == value {
			return v, nil
		}
	}
	return nil, gitnessstore.ErrResourceNotFound
}

func (s *svcValueStore) FindByID(_ context.Context, id int64) (*types.LabelValue, error) {
	v, ok := s.values[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return v, nil
}

func (s *svcValueStore) List(_ context.Context, labelID int64, _ types.ListQueryFilter) ([]*types.LabelValue, error) {
	var res []*types.LabelValue
	for _, v := range s.values {
		if v.LabelID == labelID {
			res = append(res, v)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res, s.listErr
}

func (s *svcValueStore) Count(context.Context, int64, types.ListQueryFilter) (int64, error) {
	return 42, s.countErr
}

func (s *svcValueStore) ListInfosByLabelIDs(
	_ context.Context, labelIDs []int64,
) (map[int64][]*types.LabelValueInfo, error) {
	res := map[int64][]*types.LabelValueInfo{}
	for _, id := range labelIDs {
		res[id] = []*types.LabelValueInfo{{}}
	}
	return res, nil
}

// svcAssignStore is an in-memory store.PullReqLabelAssignmentStore.
type svcAssignStore struct {
	store.PullReqLabelAssignmentStore
	assigned   map[int64]*types.PullReqLabel
	value      *types.LabelValue
	assignErr  error
	deleted    bool
	findErr    error
	valueErr   error
	unassigned []int64
	listed     map[int64]*types.LabelAssignment
	byPullreq  map[int64][]*types.LabelPullReqAssignmentInfo
	counts     map[int64]int64
	listErr    error
}

func (s *svcAssignStore) FindByLabelID(_ context.Context, _, labelID int64) (*types.PullReqLabel, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	l, ok := s.assigned[labelID]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return l, nil
}

func (s *svcAssignStore) Assign(_ context.Context, l *types.PullReqLabel) error {
	if s.assignErr != nil {
		return s.assignErr
	}
	if s.assigned == nil {
		s.assigned = map[int64]*types.PullReqLabel{}
	}
	s.assigned[l.LabelID] = l
	return nil
}

func (s *svcAssignStore) Unassign(_ context.Context, _, labelID int64) (bool, error) {
	s.unassigned = append(s.unassigned, labelID)
	return s.deleted, nil
}

func (s *svcAssignStore) FindValueByLabelID(context.Context, int64, int64) (*types.LabelValue, error) {
	return s.value, s.valueErr
}

func (s *svcAssignStore) ListAssigned(context.Context, int64) (map[int64]*types.LabelAssignment, error) {
	return s.listed, s.listErr
}

func (s *svcAssignStore) ListAssignedByPullreqIDs(
	context.Context, []int64,
) (map[int64][]*types.LabelPullReqAssignmentInfo, error) {
	return s.byPullreq, s.listErr
}

func (s *svcAssignStore) CountPullreqAssignments(context.Context, []int64) (map[int64]int64, error) {
	return s.counts, s.listErr
}

type svcSpaceStore struct {
	store.SpaceStore
	level     int64
	levelErr  error
	ancestors []int64
	ancErr    error
}

func (s *svcSpaceStore) GetTreeLevel(context.Context, int64) (int64, error) {
	return s.level, s.levelErr
}

func (s *svcSpaceStore) GetAncestorIDs(context.Context, int64) ([]int64, error) {
	return s.ancestors, s.ancErr
}

type svcSpaceCache struct {
	spaces map[int64]*types.SpaceCore
}

func (c svcSpaceCache) Stats() (int64, int64) { return 0, 0 }
func (c svcSpaceCache) Get(_ context.Context, id int64) (*types.SpaceCore, error) {
	s, ok := c.spaces[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return s, nil
}
func (c svcSpaceCache) Evict(context.Context, int64) {}

type svcEnv struct {
	svc    *Service
	labels *svcLabelStore
	values *svcValueStore
	assign *svcAssignStore
	spaces *svcSpaceStore
}

func svcNewEnv() *svcEnv {
	env := &svcEnv{
		labels: newSvcLabelStore(),
		values: newSvcValueStore(),
		assign: &svcAssignStore{},
		spaces: &svcSpaceStore{ancestors: []int64{10, 11}, level: 2},
	}
	finder := refcache.NewSpaceFinder(svcSpaceCache{spaces: map[int64]*types.SpaceCore{
		10: {ID: 10, Path: "root/space"},
		11: {ID: 11, Path: "root"},
	}}, nil, nil, cache.Evictor[*types.SpaceCore]{})
	env.svc = New(svcTx{}, env.spaces, env.labels, env.values, env.assign, nil, nil, finder)
	return env
}

func svcInt(v int64) *int64 { return &v }

func TestSvcDefine(t *testing.T) {
	ctx := context.Background()

	t.Run("repo label has scope zero", func(t *testing.T) {
		env := svcNewEnv()
		l, err := env.svc.Define(ctx, 5, nil, svcInt(1), &types.DefineLabelInput{
			Key: "bug", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed, Description: "d",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), l.Scope)
		assert.Equal(t, int64(5), l.CreatedBy)
		assert.Equal(t, int64(5), l.UpdatedBy)
		assert.Equal(t, "bug", l.Key)
		assert.NotZero(t, l.ID)
	})
	t.Run("space label uses tree level", func(t *testing.T) {
		env := svcNewEnv()
		env.spaces.level = 3
		l, err := env.svc.Define(ctx, 5, svcInt(10), nil, &types.DefineLabelInput{Key: "k"})
		require.NoError(t, err)
		assert.Equal(t, int64(3), l.Scope)
	})
	t.Run("tree level failure", func(t *testing.T) {
		env := svcNewEnv()
		env.spaces.levelErr = errors.New("boom")
		_, err := env.svc.Define(ctx, 5, svcInt(10), nil, &types.DefineLabelInput{Key: "k"})
		require.Error(t, err)
	})
	t.Run("store failure", func(t *testing.T) {
		env := svcNewEnv()
		env.labels.defineErr = errors.New("boom")
		_, err := env.svc.Define(ctx, 5, nil, svcInt(1), &types.DefineLabelInput{Key: "k"})
		require.Error(t, err)
	})
}

func TestSvcUpdate(t *testing.T) {
	ctx := context.Background()
	str := func(s string) *string { return &s }

	env := svcNewEnv()
	existing := env.labels.add(&types.Label{
		RepoID: svcInt(1), Key: "old", Color: enum.LabelColorRed, UpdatedBy: 5, Type: enum.LabelTypeStatic,
	})
	color := enum.LabelColorBlue
	typ := enum.LabelTypeDynamic

	t.Run("no changes", func(t *testing.T) {
		l, err := env.svc.Update(ctx, 5, nil, svcInt(1), "old", &types.UpdateLabelInput{Key: str("old")})
		require.NoError(t, err)
		assert.Equal(t, existing.ID, l.ID)
		assert.Zero(t, l.Updated)
	})
	t.Run("changes applied", func(t *testing.T) {
		l, err := env.svc.Update(ctx, 6, nil, svcInt(1), "old", &types.UpdateLabelInput{
			Key: str("new"), Description: str("desc"), Color: &color, Type: &typ,
		})
		require.NoError(t, err)
		assert.Equal(t, "new", l.Key)
		assert.Equal(t, "desc", l.Description)
		assert.Equal(t, color, l.Color)
		assert.Equal(t, typ, l.Type)
		assert.Equal(t, int64(6), l.UpdatedBy)
		assert.NotZero(t, l.Updated)
	})
	t.Run("unknown label", func(t *testing.T) {
		_, err := env.svc.Update(ctx, 5, nil, svcInt(1), "missing", &types.UpdateLabelInput{})
		require.Error(t, err)
	})
	t.Run("store failure", func(t *testing.T) {
		failing := svcNewEnv()
		failing.labels.add(&types.Label{RepoID: svcInt(1), Key: "k", UpdatedBy: 1})
		failing.labels.updateErr = errors.New("boom")
		_, err := failing.svc.Update(ctx, 2, nil, svcInt(1), "k", &types.UpdateLabelInput{})
		require.Error(t, err)
	})
}

func TestSvcSave(t *testing.T) {
	ctx := context.Background()

	newInput := func(id int64, values ...*types.SaveLabelValueInput) *types.SaveInput {
		return &types.SaveInput{
			Label: types.SaveLabelInput{
				ID: id, DefineLabelInput: types.DefineLabelInput{
					Key: "area", Type: enum.LabelTypeDynamic, Color: enum.LabelColorBlue,
				},
			},
			Values: values,
		}
	}
	value := func(id int64, v string) *types.SaveLabelValueInput {
		return &types.SaveLabelValueInput{ID: id, DefineValueInput: types.DefineValueInput{
			Value: v, Color: enum.LabelColorGreen,
		}}
	}

	t.Run("defines new label with values", func(t *testing.T) {
		env := svcNewEnv()
		out, err := env.svc.Save(ctx, 5, nil, svcInt(1), newInput(0, value(0, "b"), value(0, "a")))
		require.NoError(t, err)
		assert.Equal(t, "area", out.Key)
		require.Len(t, out.Values, 2)
		assert.Equal(t, "a", out.Values[0].Value, "values are sorted")
		assert.Equal(t, int64(2), out.ValueCount)
	})
	t.Run("updates existing label, values and deletes the rest", func(t *testing.T) {
		env := svcNewEnv()
		l := env.labels.add(&types.Label{RepoID: svcInt(1), Key: "area", Type: enum.LabelTypeDynamic, ValueCount: 2})
		keep := env.values.add(&types.LabelValue{LabelID: l.ID, Value: "keep", Color: enum.LabelColorRed})
		env.values.add(&types.LabelValue{LabelID: l.ID, Value: "drop"})

		out, err := env.svc.Save(ctx, 5, nil, svcInt(1), newInput(l.ID, value(keep.ID, "keep"), value(0, "fresh")))
		require.NoError(t, err)
		assert.Equal(t, []string{"drop"}, env.values.deleted)
		require.Len(t, out.Values, 2)
		assert.Equal(t, "fresh", out.Values[0].Value)
		assert.Equal(t, "keep", out.Values[1].Value)
		assert.Equal(t, enum.LabelColorGreen, out.Values[1].Color)
		assert.Equal(t, []int{0}, env.labels.increment)
	})
	t.Run("label from another scope is rejected", func(t *testing.T) {
		env := svcNewEnv()
		l := env.labels.add(&types.Label{RepoID: svcInt(2), Key: "area"})
		_, err := env.svc.Save(ctx, 5, nil, svcInt(1), newInput(l.ID))
		require.Error(t, err)
		assert.Equal(t, gitnesserrors.StatusInvalidArgument, gitnesserrors.AsStatus(err))
	})
	t.Run("duplicate value is a conflict", func(t *testing.T) {
		env := svcNewEnv()
		env.values.defineErr = gitnessstore.ErrDuplicate
		_, err := env.svc.Save(ctx, 5, nil, svcInt(1), newInput(0, value(0, "dup")))
		require.Error(t, err)
		assert.Equal(t, gitnesserrors.StatusConflict, gitnesserrors.AsStatus(err))
	})
	t.Run("store failures", func(t *testing.T) {
		env := svcNewEnv()
		env.labels.defineErr = errors.New("boom")
		_, err := env.svc.Save(ctx, 5, nil, svcInt(1), newInput(0))
		require.Error(t, err)

		env = svcNewEnv()
		env.values.listErr = errors.New("boom")
		_, err = env.svc.Save(ctx, 5, nil, svcInt(1), newInput(0))
		require.Error(t, err)

		env = svcNewEnv()
		env.values.deleteErr = errors.New("boom")
		_, err = env.svc.Save(ctx, 5, nil, svcInt(1), newInput(0))
		require.Error(t, err)

		env = svcNewEnv()
		env.values.defineErr = errors.New("boom")
		_, err = env.svc.Save(ctx, 5, nil, svcInt(1), newInput(0, value(0, "x")))
		require.Error(t, err)

		env = svcNewEnv()
		l := env.labels.add(&types.Label{RepoID: svcInt(1), Key: "area"})
		env.labels.updateErr = errors.New("boom")
		_, err = env.svc.Save(ctx, 5, nil, svcInt(1), newInput(l.ID))
		require.Error(t, err)
	})
}

func TestSvcFindAndList(t *testing.T) {
	ctx := context.Background()
	env := svcNewEnv()
	l := env.labels.add(&types.Label{RepoID: svcInt(1), SpaceID: nil, Key: "bug"})
	env.values.add(&types.LabelValue{LabelID: l.ID, Value: "v"})

	t.Run("find", func(t *testing.T) {
		got, err := env.svc.Find(ctx, nil, svcInt(1), "bug")
		require.NoError(t, err)
		assert.Equal(t, l.ID, got.ID)

		got, err = env.svc.FindByID(ctx, l.ID)
		require.NoError(t, err)
		assert.Equal(t, "bug", got.Key)
	})
	t.Run("find with values", func(t *testing.T) {
		got, err := env.svc.FindWithValues(ctx, nil, svcInt(1), "bug")
		require.NoError(t, err)
		assert.Len(t, got.Values, 1)

		_, err = env.svc.FindWithValues(ctx, nil, svcInt(1), "nope")
		require.Error(t, err)

		env.values.listErr = errors.New("boom")
		_, err = env.svc.FindWithValues(ctx, nil, svcInt(1), "bug")
		require.Error(t, err)
		env.values.listErr = nil
	})
	t.Run("list in repo", func(t *testing.T) {
		labels, total, err := env.svc.List(ctx, nil, svcInt(1), &types.LabelFilter{})
		require.NoError(t, err)
		assert.Len(t, labels, 1)
		assert.Equal(t, int64(1), total)
	})
	t.Run("list in space", func(t *testing.T) {
		labels, total, err := env.svc.List(ctx, svcInt(10), nil, &types.LabelFilter{})
		require.NoError(t, err)
		assert.Len(t, labels, 1)
		assert.Equal(t, int64(1), total)
	})
	t.Run("list inherited from repo", func(t *testing.T) {
		labels, total, err := env.svc.List(ctx, svcInt(10), svcInt(1), &types.LabelFilter{Inherited: true})
		require.NoError(t, err)
		assert.Len(t, labels, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, []int64{1, 10, 11}, env.labels.listedIn)
	})
	t.Run("list inherited from space", func(t *testing.T) {
		_, _, err := env.svc.List(ctx, svcInt(10), nil, &types.LabelFilter{Inherited: true})
		require.NoError(t, err)
		assert.Equal(t, []int64{0, 10, 11}, env.labels.listedIn)
	})
	t.Run("list with pullreq counts", func(t *testing.T) {
		env.assign.counts = map[int64]int64{l.ID: 7}
		labels, _, err := env.svc.List(ctx, nil, svcInt(1), &types.LabelFilter{IncludePullreqCount: true})
		require.NoError(t, err)
		assert.Equal(t, int64(7), labels[0].PullreqCount)

		env.assign.listErr = errors.New("boom")
		_, _, err = env.svc.List(ctx, nil, svcInt(1), &types.LabelFilter{IncludePullreqCount: true})
		require.Error(t, err)
		env.assign.listErr = nil
	})
	t.Run("list failures", func(t *testing.T) {
		failing := svcNewEnv()
		failing.labels.countErr = errors.New("boom")
		_, _, err := failing.svc.List(ctx, nil, svcInt(1), &types.LabelFilter{})
		require.Error(t, err)
		_, _, err = failing.svc.List(ctx, svcInt(10), nil, &types.LabelFilter{})
		require.Error(t, err)
		_, _, err = failing.svc.List(ctx, svcInt(10), svcInt(1), &types.LabelFilter{Inherited: true})
		require.Error(t, err)

		failing = svcNewEnv()
		failing.labels.listErr = errors.New("boom")
		_, _, err = failing.svc.List(ctx, nil, svcInt(1), &types.LabelFilter{})
		require.Error(t, err)
		_, _, err = failing.svc.List(ctx, svcInt(10), nil, &types.LabelFilter{})
		require.Error(t, err)
		_, _, err = failing.svc.List(ctx, svcInt(10), svcInt(1), &types.LabelFilter{Inherited: true})
		require.Error(t, err)

		failing = svcNewEnv()
		failing.spaces.ancErr = errors.New("boom")
		_, _, err = failing.svc.List(ctx, svcInt(10), svcInt(1), &types.LabelFilter{Inherited: true})
		require.Error(t, err)
		_, _, err = failing.svc.List(ctx, svcInt(10), nil, &types.LabelFilter{Inherited: true})
		require.Error(t, err)
	})
	t.Run("delete", func(t *testing.T) {
		require.NoError(t, env.svc.Delete(ctx, nil, svcInt(1), "bug"))
		assert.Equal(t, []string{"bug"}, env.labels.deleted)
	})
}

func TestSvcLabelValues(t *testing.T) {
	ctx := context.Background()

	t.Run("define value increments the value count", func(t *testing.T) {
		env := svcNewEnv()
		l := env.labels.add(&types.Label{RepoID: svcInt(1), Key: "k"})
		v, err := env.svc.DefineValue(ctx, 5, l.ID, &types.DefineValueInput{Value: "v", Color: enum.LabelColorRed})
		require.NoError(t, err)
		assert.Equal(t, "v", v.Value)
		assert.Equal(t, int64(1), env.labels.labels[l.ID].ValueCount)

		env.values.defineErr = errors.New("boom")
		_, err = env.svc.DefineValue(ctx, 5, l.ID, &types.DefineValueInput{Value: "w"})
		require.Error(t, err)
	})
	t.Run("update value", func(t *testing.T) {
		env := svcNewEnv()
		l := env.labels.add(&types.Label{RepoID: svcInt(1), Key: "k"})
		env.values.add(&types.LabelValue{LabelID: l.ID, Value: "old", UpdatedBy: 5})
		newValue := "new"
		color := enum.LabelColorPink

		same, err := env.svc.UpdateValue(ctx, 5, l.ID, "old", &types.UpdateValueInput{})
		require.NoError(t, err)
		assert.Zero(t, same.Updated)

		changed, err := env.svc.UpdateValue(ctx, 6, l.ID, "old", &types.UpdateValueInput{Value: &newValue, Color: &color})
		require.NoError(t, err)
		assert.Equal(t, "new", changed.Value)
		assert.Equal(t, color, changed.Color)
		assert.Equal(t, int64(6), changed.UpdatedBy)

		_, err = env.svc.UpdateValue(ctx, 5, l.ID, "missing", &types.UpdateValueInput{})
		require.Error(t, err)
	})
	t.Run("list values", func(t *testing.T) {
		env := svcNewEnv()
		l := env.labels.add(&types.Label{RepoID: svcInt(1), Key: "k"})
		env.values.add(&types.LabelValue{LabelID: l.ID, Value: "a"})

		vals, count, err := env.svc.ListValues(ctx, nil, svcInt(1), "k", types.ListQueryFilter{
			Pagination: types.Pagination{Page: 1, Size: 10},
		})
		require.NoError(t, err)
		assert.Len(t, vals, 1)
		assert.Equal(t, int64(1), count, "single short page does not need a count query")

		_, count, err = env.svc.ListValues(ctx, nil, svcInt(1), "k", types.ListQueryFilter{
			Pagination: types.Pagination{Page: 2, Size: 10},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(42), count)

		_, _, err = env.svc.ListValues(ctx, nil, svcInt(1), "missing", types.ListQueryFilter{})
		require.Error(t, err)

		env.values.listErr = errors.New("boom")
		_, _, err = env.svc.ListValues(ctx, nil, svcInt(1), "k", types.ListQueryFilter{})
		require.Error(t, err)
		env.values.listErr = nil

		env.values.countErr = errors.New("boom")
		_, _, err = env.svc.ListValues(ctx, nil, svcInt(1), "k", types.ListQueryFilter{
			Pagination: types.Pagination{Page: 2, Size: 10},
		})
		require.Error(t, err)
	})
	t.Run("delete value", func(t *testing.T) {
		env := svcNewEnv()
		l := env.labels.add(&types.Label{RepoID: svcInt(1), Key: "k", ValueCount: 3})
		require.NoError(t, env.svc.DeleteValue(ctx, nil, svcInt(1), "k", "v"))
		assert.Equal(t, []string{"v"}, env.values.deleted)
		assert.Equal(t, int64(2), env.labels.labels[l.ID].ValueCount)

		require.Error(t, env.svc.DeleteValue(ctx, nil, svcInt(1), "missing", "v"))

		env.values.deleteErr = errors.New("boom")
		require.Error(t, env.svc.DeleteValue(ctx, nil, svcInt(1), "k", "v"))
	})
}

func TestSvcApplyChanges(t *testing.T) {
	label := &types.Label{Key: "a", UpdatedBy: 1}
	_, changed := applyChanges(1, label, &types.UpdateLabelInput{})
	assert.False(t, changed)

	_, changed = applyChanges(2, label, &types.UpdateLabelInput{})
	assert.True(t, changed, "a different principal alone counts as a change")
	assert.Equal(t, int64(2), label.UpdatedBy)

	desc := "d"
	_, changed = applyChanges(2, label, &types.UpdateLabelInput{Description: &desc})
	assert.True(t, changed)
	_, changed = applyChanges(2, label, &types.UpdateLabelInput{Description: &desc})
	assert.False(t, changed)
}

func TestSvcCheckLabelInScope(t *testing.T) {
	tests := []struct {
		name    string
		spaceID *int64
		repoID  *int64
		label   types.Label
		wantErr bool
	}{
		{"same repo", nil, svcInt(1), types.Label{RepoID: svcInt(1)}, false},
		{"other repo", nil, svcInt(1), types.Label{RepoID: svcInt(2)}, true},
		{"space label requested for repo", nil, svcInt(1), types.Label{SpaceID: svcInt(3)}, true},
		{"same space", svcInt(3), nil, types.Label{SpaceID: svcInt(3)}, false},
		{"other space", svcInt(3), nil, types.Label{SpaceID: svcInt(4)}, true},
		{"repo label requested for space", svcInt(3), nil, types.Label{RepoID: svcInt(1)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkLabelInScope(tt.spaceID, tt.repoID, &tt.label)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
