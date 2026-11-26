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
	"testing"

	"github.com/harness/gitness/app/services/autolink"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/rules"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/audit"
	mocksse "github.com/harness/gitness/mocks/sse"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// crpLabelStore is an in-memory store.LabelStore holding a single label.
type crpLabelStore struct {
	store.LabelStore
	label     *types.Label
	err       error
	defineErr error
	updateErr error
	incErr    error
	labels    []*types.Label
	count     int64
	deleted   bool
}

func (s *crpLabelStore) Find(context.Context, *int64, *int64, string) (*types.Label, error) {
	if s.err != nil {
		return nil, s.err
	}

	label := *s.label

	return &label, nil
}

func (s *crpLabelStore) Define(_ context.Context, lbl *types.Label) error {
	if s.defineErr != nil {
		return s.defineErr
	}

	lbl.ID = 77

	return nil
}

func (s *crpLabelStore) Update(context.Context, *types.Label) error { return s.updateErr }

func (s *crpLabelStore) Delete(context.Context, *int64, *int64, string) error {
	s.deleted = true

	return s.err
}

func (s *crpLabelStore) IncrementValueCount(context.Context, int64, int) (int64, error) {
	return 1, s.incErr
}

func (s *crpLabelStore) List(context.Context, *int64, *int64, *types.LabelFilter) ([]*types.Label, error) {
	return s.labels, s.err
}

func (s *crpLabelStore) CountInRepo(context.Context, int64, *types.LabelFilter) (int64, error) {
	return s.count, s.err
}

// crpLabelValueStore is an in-memory store.LabelValueStore.
type crpLabelValueStore struct {
	store.LabelValueStore
	value     *types.LabelValue
	values    []*types.LabelValue
	count     int64
	defineErr error
	findErr   error
	updateErr error
	listErr   error
	countErr  error
	deleteErr error
}

func (s *crpLabelValueStore) Define(_ context.Context, v *types.LabelValue) error {
	if s.defineErr != nil {
		return s.defineErr
	}

	v.ID = 55

	return nil
}

func (s *crpLabelValueStore) FindByLabelID(context.Context, int64, string) (*types.LabelValue, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}

	value := *s.value

	return &value, nil
}

func (s *crpLabelValueStore) Update(context.Context, *types.LabelValue) error { return s.updateErr }

func (s *crpLabelValueStore) Delete(context.Context, int64, string) error { return s.deleteErr }

func (s *crpLabelValueStore) List(context.Context, int64, types.ListQueryFilter) ([]*types.LabelValue, error) {
	return s.values, s.listErr
}

func (s *crpLabelValueStore) Count(context.Context, int64, types.ListQueryFilter) (int64, error) {
	return s.count, s.countErr
}

func crpWithLabels(e *crpEnv, labels *crpLabelStore, values *crpLabelValueStore) {
	e.c.labelSvc = label.New(crpTx{}, nil, labels, values, nil, nil, nil, e.c.spaceFinder)
}

func TestCrpLabels(t *testing.T) {
	ctx := context.Background()
	existing := &types.Label{ID: 5, Key: "bug", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed}

	newEnv := func(t *testing.T) (*crpEnv, *crpLabelStore, *crpLabelValueStore) {
		t.Helper()
		e := crpNewEnv(t)
		labels := &crpLabelStore{label: existing}
		values := &crpLabelValueStore{value: &types.LabelValue{ID: 9, LabelID: 5, Value: "p1", Color: enum.LabelColorRed}}
		crpWithLabels(e, labels, values)

		return e, labels, values
	}

	t.Run("define", func(t *testing.T) {
		e, labels, _ := newEnv(t)

		out, err := e.c.DefineLabel(ctx, crpSession(), crpRepoRef, &types.DefineLabelInput{
			Key: crpFeatureBranch, Type: enum.LabelTypeStatic, Color: enum.LabelColorBlue, Description: "  d  ",
		})

		require.NoError(t, err)
		assert.EqualValues(t, 77, out.ID)
		assert.Equal(t, "d", out.Description)
		assert.Equal(t, crpRepoID, *out.RepoID)

		labels.defineErr = errCrpBoom
		_, err = e.c.DefineLabel(ctx, crpSession(), crpRepoRef, &types.DefineLabelInput{Key: "x"})
		require.ErrorIs(t, err, errCrpBoom)

		_, err = e.c.DefineLabel(ctx, crpSession(), crpRepoRef, &types.DefineLabelInput{})
		require.Error(t, err)
	})

	t.Run("find", func(t *testing.T) {
		e, labels, values := newEnv(t)
		values.values = []*types.LabelValue{{ID: 1, Value: "a"}}

		plain, err := e.c.FindLabel(ctx, crpSession(), crpRepoRef, "bug", false)
		require.NoError(t, err)
		assert.Equal(t, "bug", plain.Key)
		assert.Empty(t, plain.Values)

		withValues, err := e.c.FindLabel(ctx, crpSession(), crpRepoRef, "bug", true)
		require.NoError(t, err)
		require.Len(t, withValues.Values, 1)

		values.listErr = errCrpBoom
		_, err = e.c.FindLabel(ctx, crpSession(), crpRepoRef, "bug", true)
		require.ErrorIs(t, err, errCrpBoom)

		labels.err = errCrpBoom
		_, err = e.c.FindLabel(ctx, crpSession(), crpRepoRef, "bug", true)
		require.ErrorIs(t, err, errCrpBoom)
		_, err = e.c.FindLabel(ctx, crpSession(), crpRepoRef, "bug", false)
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("list", func(t *testing.T) {
		e, labels, _ := newEnv(t)
		labels.labels = []*types.Label{existing}
		labels.count = 11

		out, count, err := e.c.ListLabels(ctx, crpSession(), crpRepoRef, &types.LabelFilter{})
		require.NoError(t, err)
		assert.Equal(t, []*types.Label{existing}, out)
		assert.EqualValues(t, 11, count)

		labels.err = errCrpBoom
		_, _, err = e.c.ListLabels(ctx, crpSession(), crpRepoRef, &types.LabelFilter{})
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("update", func(t *testing.T) {
		e, labels, _ := newEnv(t)
		desc := "new description"

		out, err := e.c.UpdateLabel(ctx, crpSession(), crpRepoRef, "bug", &types.UpdateLabelInput{Description: &desc})
		require.NoError(t, err)
		assert.Equal(t, desc, out.Description)

		bad := enum.LabelType("nonsense")
		_, err = e.c.UpdateLabel(ctx, crpSession(), crpRepoRef, "bug", &types.UpdateLabelInput{Type: &bad})
		require.Error(t, err)

		labels.updateErr = errCrpBoom
		_, err = e.c.UpdateLabel(ctx, crpSession(), crpRepoRef, "bug", &types.UpdateLabelInput{Description: &desc})
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("save rejects invalid input", func(t *testing.T) {
		e, _, _ := newEnv(t)

		_, err := e.c.SaveLabel(ctx, crpSession(), crpRepoRef, &types.SaveInput{})

		require.Error(t, err)
	})

	t.Run("delete", func(t *testing.T) {
		e, labels, _ := newEnv(t)

		require.NoError(t, e.c.DeleteLabel(ctx, crpSession(), crpRepoRef, "bug"))
		assert.True(t, labels.deleted)

		labels.err = errCrpBoom
		require.ErrorIs(t, e.c.DeleteLabel(ctx, crpSession(), crpRepoRef, "bug"), errCrpBoom)
	})

	t.Run("define value", func(t *testing.T) {
		e, labels, values := newEnv(t)

		out, err := e.c.DefineLabelValue(ctx, crpSession(), crpRepoRef, "bug",
			&types.DefineValueInput{Value: "p2", Color: enum.LabelColorGreen})
		require.NoError(t, err)
		assert.EqualValues(t, 55, out.ID)
		assert.EqualValues(t, 5, out.LabelID)

		_, err = e.c.DefineLabelValue(ctx, crpSession(), crpRepoRef, "bug", &types.DefineValueInput{})
		require.Error(t, err)

		values.defineErr = errCrpBoom
		_, err = e.c.DefineLabelValue(ctx, crpSession(), crpRepoRef, "bug", &types.DefineValueInput{Value: "p3"})
		require.ErrorIs(t, err, errCrpBoom)

		labels.err = errCrpBoom
		_, err = e.c.DefineLabelValue(ctx, crpSession(), crpRepoRef, "bug", &types.DefineValueInput{Value: "p3"})
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("update value", func(t *testing.T) {
		e, labels, values := newEnv(t)
		newValue := "p0"

		out, err := e.c.UpdateLabelValue(
			ctx, crpSession(), crpRepoRef, "bug", "p1", &types.UpdateValueInput{Value: &newValue})
		require.NoError(t, err)
		assert.Equal(t, "p0", out.Value)

		blank := ""
		_, err = e.c.UpdateLabelValue(ctx, crpSession(), crpRepoRef, "bug", "p1", &types.UpdateValueInput{Value: &blank})
		require.Error(t, err)

		values.updateErr = errCrpBoom
		_, err = e.c.UpdateLabelValue(ctx, crpSession(), crpRepoRef, "bug", "p1", &types.UpdateValueInput{Value: &newValue})
		require.ErrorIs(t, err, errCrpBoom)

		values.findErr = errCrpBoom
		_, err = e.c.UpdateLabelValue(ctx, crpSession(), crpRepoRef, "bug", "p1", &types.UpdateValueInput{Value: &newValue})
		require.ErrorIs(t, err, errCrpBoom)

		labels.err = errCrpBoom
		_, err = e.c.UpdateLabelValue(ctx, crpSession(), crpRepoRef, "bug", "p1", &types.UpdateValueInput{Value: &newValue})
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("delete value", func(t *testing.T) {
		e, labels, values := newEnv(t)

		require.NoError(t, e.c.DeleteLabelValue(ctx, crpSession(), crpRepoRef, "bug", "p1"))

		values.deleteErr = errCrpBoom
		require.ErrorIs(t, e.c.DeleteLabelValue(ctx, crpSession(), crpRepoRef, "bug", "p1"), errCrpBoom)

		labels.err = errCrpBoom
		require.ErrorIs(t, e.c.DeleteLabelValue(ctx, crpSession(), crpRepoRef, "bug", "p1"), errCrpBoom)
	})

	t.Run("list values", func(t *testing.T) {
		e, labels, values := newEnv(t)
		values.values = []*types.LabelValue{{ID: 1}}
		values.count = 40

		out, count, err := e.c.ListLabelValues(ctx, crpSession(), crpRepoRef, "bug",
			types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}})
		require.NoError(t, err)
		assert.Len(t, out, 1)
		assert.EqualValues(t, 1, count)

		_, count, err = e.c.ListLabelValues(ctx, crpSession(), crpRepoRef, "bug",
			types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 1}})
		require.NoError(t, err)
		assert.EqualValues(t, 40, count)

		values.countErr = errCrpBoom
		_, _, err = e.c.ListLabelValues(ctx, crpSession(), crpRepoRef, "bug",
			types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 1}})
		require.ErrorIs(t, err, errCrpBoom)

		values.listErr = errCrpBoom
		_, _, err = e.c.ListLabelValues(ctx, crpSession(), crpRepoRef, "bug", types.ListQueryFilter{})
		require.ErrorIs(t, err, errCrpBoom)

		labels.err = errCrpBoom
		_, _, err = e.c.ListLabelValues(ctx, crpSession(), crpRepoRef, "bug", types.ListQueryFilter{})
		require.ErrorIs(t, err, errCrpBoom)
	})
}

// crpAutoLinkStore is an in-memory store.AutoLinkStore.
type crpAutoLinkStore struct {
	store.AutoLinkStore
	links    map[int64]*types.AutoLink
	list     []*types.AutoLink
	count    int64
	err      error
	listErr  error
	writeErr error
	deleted  []int64
}

func (s *crpAutoLinkStore) Find(_ context.Context, id int64) (*types.AutoLink, error) {
	if s.err != nil {
		return nil, s.err
	}

	link, ok := s.links[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}

	clone := *link

	return &clone, nil
}

func (s *crpAutoLinkStore) Create(_ context.Context, link *types.AutoLink) error {
	if s.writeErr != nil {
		return s.writeErr
	}

	link.ID = 31

	return nil
}

func (s *crpAutoLinkStore) Update(context.Context, *types.AutoLink) error { return s.writeErr }

func (s *crpAutoLinkStore) Delete(_ context.Context, id int64) error {
	s.deleted = append(s.deleted, id)

	return s.writeErr
}

func (s *crpAutoLinkStore) List(context.Context, *int64, *int64, *types.AutoLinkFilter) ([]*types.AutoLink, error) {
	return s.list, s.listErr
}

func (s *crpAutoLinkStore) Count(context.Context, *int64, *int64, *types.AutoLinkFilter) (int64, error) {
	return s.count, s.err
}

func TestCrpAutolinks(t *testing.T) {
	ctx := context.Background()
	repoID := crpRepoID

	newEnv := func(t *testing.T) (*crpEnv, *crpAutoLinkStore) {
		t.Helper()
		e := crpNewEnv(t)
		links := &crpAutoLinkStore{links: map[int64]*types.AutoLink{
			1: {ID: 1, RepoID: &repoID, Type: enum.AutoLinkTypeRegex, Pattern: "JIRA-\\d+", TargetURL: "https://example.com/x"},
			2: {ID: 2, RepoID: ptrTo(int64(999)), Type: enum.AutoLinkTypeRegex, Pattern: "x", TargetURL: "https://example.com"},
		}}
		e.c.autolinkSvc = autolink.NewService(crpTx{}, e.c.spaceStore, e.c.repoStore, links)

		return e, links
	}

	t.Run("create", func(t *testing.T) {
		e, links := newEnv(t)
		in := &autolink.AutoLinkInput{
			Type: enum.AutoLinkTypePrefixWithNumValue, Pattern: "BUG-", TargetURL: "https://example.com/BUG-<num>",
		}

		out, err := e.c.AutolinkCreate(ctx, crpSession(), crpRepoRef, in)
		require.NoError(t, err)
		assert.EqualValues(t, 31, out.ID)
		assert.Equal(t, "BUG-([0-9]+)", out.Pattern)

		_, err = e.c.AutolinkCreate(ctx, crpSession(), crpRepoRef, &autolink.AutoLinkInput{})
		require.Error(t, err)

		links.writeErr = errCrpBoom
		_, err = e.c.AutolinkCreate(ctx, crpSession(), crpRepoRef, in)
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("find", func(t *testing.T) {
		e, _ := newEnv(t)

		out, err := e.c.AutolinkFind(ctx, crpSession(), crpRepoRef, 1)
		require.NoError(t, err)
		assert.EqualValues(t, 1, out.ID)

		_, err = e.c.AutolinkFind(ctx, crpSession(), crpRepoRef, 2)
		require.Error(t, err, "autolinks of other repositories are not visible")

		_, err = e.c.AutolinkFind(ctx, crpSession(), crpRepoRef, 3)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})

	t.Run("list", func(t *testing.T) {
		e, links := newEnv(t)
		links.list = []*types.AutoLink{
			{Type: enum.AutoLinkTypePrefixWithNumValue, Pattern: "A-"},
			{Type: enum.AutoLinkTypePrefixWithAlphanumericValue, Pattern: "B-"},
			{Type: enum.AutoLinkTypeRegex, Pattern: "C"},
		}
		links.count = 3

		out, count, err := e.c.AutolinkList(ctx, crpSession(), crpRepoRef, &types.AutoLinkFilter{})
		require.NoError(t, err)
		assert.EqualValues(t, 3, count)
		assert.Equal(t, "^A-(\\d+)$", out[0].Pattern)
		assert.Equal(t, "^B-([\\w-]+)$", out[1].Pattern)
		assert.Equal(t, "C", out[2].Pattern)

		links.listErr = errCrpBoom
		_, _, err = e.c.AutolinkList(ctx, crpSession(), crpRepoRef, &types.AutoLinkFilter{})
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("update", func(t *testing.T) {
		e, links := newEnv(t)
		pattern := "ISSUE-\\d+"

		out, err := e.c.AutolinkUpdate(ctx, crpSession(), crpRepoRef, 1, &autolink.AutoLinkUpdateInput{Pattern: &pattern})
		require.NoError(t, err)
		assert.Equal(t, pattern, out.Pattern)
		assert.EqualValues(t, 42, out.UpdatedBy)

		badURL := "ftp://example.com"
		_, err = e.c.AutolinkUpdate(ctx, crpSession(), crpRepoRef, 1, &autolink.AutoLinkUpdateInput{TargetURL: &badURL})
		require.Error(t, err)

		_, err = e.c.AutolinkUpdate(ctx, crpSession(), crpRepoRef, 2, &autolink.AutoLinkUpdateInput{Pattern: &pattern})
		require.Error(t, err)

		links.writeErr = errCrpBoom
		_, err = e.c.AutolinkUpdate(ctx, crpSession(), crpRepoRef, 1, &autolink.AutoLinkUpdateInput{Pattern: &pattern})
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("delete", func(t *testing.T) {
		e, links := newEnv(t)

		require.NoError(t, e.c.AutolinkDelete(ctx, crpSession(), crpRepoRef, 1))
		assert.Equal(t, []int64{1}, links.deleted)

		require.Error(t, e.c.AutolinkDelete(ctx, crpSession(), crpRepoRef, 2))

		links.writeErr = errCrpBoom
		require.ErrorIs(t, e.c.AutolinkDelete(ctx, crpSession(), crpRepoRef, 1), errCrpBoom)
	})
}

func ptrTo[T any](v T) *T { return &v }

func TestCrpRules(t *testing.T) {
	ctx := context.Background()
	rule := func() *types.Rule {
		return &types.Rule{
			ID: 3, Identifier: "r1", Type: protection.TypeBranch, State: enum.RuleStateActive,
			Pattern: []byte(`{}`), RepoTarget: []byte(`{}`), Definition: []byte(`{}`),
		}
	}

	newEnv := func(t *testing.T) (*crpEnv, *mocksse.Streamer) {
		t.Helper()
		e := crpNewEnv(t)
		e.ruleStore.rule = rule()
		streamer := &mocksse.Streamer{}
		streamer.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return()
		e.c.rulesSvc = rules.NewService(
			crpTx{}, e.ruleStore, e.c.repoStore, e.c.spaceStore, e.c.protectionManager, audit.New(),
			e.c.instrumentation, crpPrincipalInfoCache{infos: map[int64]*types.PrincipalInfo{}}, nil,
			crpUserGroupService{}, nil, streamer, nil, e.idCache,
		)

		return e, streamer
	}

	t.Run("find", func(t *testing.T) {
		e, _ := newEnv(t)

		out, err := e.c.RuleFind(ctx, crpSession(), crpRepoRef, "r1")
		require.NoError(t, err)
		assert.Equal(t, "r1", out.Identifier)

		e.ruleStore.findErr = errCrpBoom
		_, err = e.c.RuleFind(ctx, crpSession(), crpRepoRef, "r1")
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("list", func(t *testing.T) {
		e, _ := newEnv(t)
		e.ruleStore.listRules = []types.Rule{*rule()}
		e.ruleStore.countVal = 50

		out, count, err := e.c.RuleList(ctx, crpSession(), crpRepoRef, false, &types.RuleFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
		})
		require.NoError(t, err)
		assert.Len(t, out, 1)
		assert.EqualValues(t, 1, count)

		_, count, err = e.c.RuleList(ctx, crpSession(), crpRepoRef, true, &types.RuleFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 1}},
		})
		require.NoError(t, err)
		assert.EqualValues(t, 50, count)

		e.ruleStore.listErr = errCrpBoom
		_, _, err = e.c.RuleList(ctx, crpSession(), crpRepoRef, false, &types.RuleFilter{})
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("delete", func(t *testing.T) {
		e, streamer := newEnv(t)

		require.NoError(t, e.c.RuleDelete(ctx, crpSession(), crpRepoRef, "r1"))
		streamer.AssertCalled(t, "Publish", e.repo.ParentID, enum.SSETypeRuleDeleted, mock.Anything)

		e.ruleStore.deleteErr = errCrpBoom
		require.ErrorIs(t, e.c.RuleDelete(ctx, crpSession(), crpRepoRef, "r1"), errCrpBoom)

		e.ruleStore.findErr = errCrpBoom
		require.ErrorIs(t, e.c.RuleDelete(ctx, crpSession(), crpRepoRef, "r1"), errCrpBoom)
	})

	t.Run("update without changes returns the rule", func(t *testing.T) {
		e, _ := newEnv(t)

		out, err := e.c.RuleUpdate(ctx, crpSession(), crpRepoRef, "r1", &rules.UpdateInput{})
		require.NoError(t, err)
		assert.Equal(t, "r1", out.Identifier)

		empty := ""
		_, err = e.c.RuleUpdate(ctx, crpSession(), crpRepoRef, "r1", &rules.UpdateInput{Identifier: &empty})
		require.Error(t, err)
	})

	t.Run("create rejects invalid input", func(t *testing.T) {
		e, _ := newEnv(t)

		_, err := e.c.RuleCreate(ctx, crpSession(), crpRepoRef, &rules.CreateInput{})

		require.Error(t, err)
	})
}

func TestCrpMergeQueueWrappers(t *testing.T) {
	ctx := context.Background()

	t.Run("branch is required", func(t *testing.T) {
		e := crpNewEnv(t)

		_, err := e.c.MergeQueueListEntries(ctx, crpSession(), crpRepoRef, "  ")
		require.Error(t, err)

		err = e.c.MergeQueueClear(ctx, crpSession(), crpRepoRef, "")
		require.Error(t, err)
	})

	t.Run("an empty merge queue is listed and cleared", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{})

		out, err := e.c.MergeQueueListEntries(ctx, crpSession(), crpRepoRef, crpDefaultBranch)
		require.NoError(t, err)
		assert.False(t, out.Active)
		assert.Empty(t, out.Entries)

		require.NoError(t, e.c.MergeQueueClear(ctx, crpSession(), crpRepoRef, crpDefaultBranch))
	})

	t.Run("failures", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{queueErr: errCrpBoom})

		_, err := e.c.MergeQueueListEntries(ctx, crpSession(), crpRepoRef, crpDefaultBranch)
		require.ErrorIs(t, err, errCrpBoom)

		require.ErrorIs(t, e.c.MergeQueueClear(ctx, crpSession(), crpRepoRef, crpDefaultBranch), errCrpBoom)

		e.ruleStore.err = errCrpBoom
		_, err = e.c.MergeQueueListEntries(ctx, crpSession(), crpRepoRef, crpDefaultBranch)
		require.ErrorIs(t, err, errCrpBoom)
	})
}
