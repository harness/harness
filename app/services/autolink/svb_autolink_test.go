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

package autolink

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type svbAutoLinkStore struct {
	store.AutoLinkStore

	items       map[int64]*types.AutoLink
	created     *types.AutoLink
	updated     *types.AutoLink
	deletedID   int64
	listResult  []*types.AutoLink
	err         error
	gotRepoID   int64
	gotSpaceIDs []int64
	inScope     bool
}

func (f *svbAutoLinkStore) Create(_ context.Context, al *types.AutoLink) error {
	if f.err != nil {
		return f.err
	}
	f.created = al
	return nil
}

func (f *svbAutoLinkStore) Update(_ context.Context, al *types.AutoLink) error {
	if f.err != nil {
		return f.err
	}
	f.updated = al
	return nil
}

func (f *svbAutoLinkStore) Find(_ context.Context, id int64) (*types.AutoLink, error) {
	if f.err != nil {
		return nil, f.err
	}
	al, ok := f.items[id]
	if !ok {
		return nil, usererror.ErrNotFound
	}
	return al, nil
}

func (f *svbAutoLinkStore) Delete(_ context.Context, id int64) error {
	if f.err != nil {
		return f.err
	}
	f.deletedID = id
	return nil
}

func (f *svbAutoLinkStore) Count(context.Context, *int64, *int64, *types.AutoLinkFilter) (int64, error) {
	return int64(len(f.listResult)), f.err
}

func (f *svbAutoLinkStore) List(
	context.Context, *int64, *int64, *types.AutoLinkFilter,
) ([]*types.AutoLink, error) {
	return f.listResult, f.err
}

func (f *svbAutoLinkStore) CountInScopes(
	_ context.Context, repoID int64, spaceIDs []int64, _ *types.AutoLinkFilter,
) (int64, error) {
	f.inScope = true
	f.gotRepoID = repoID
	f.gotSpaceIDs = spaceIDs
	return int64(len(f.listResult)), f.err
}

func (f *svbAutoLinkStore) ListInScopes(
	context.Context, int64, []int64, *types.AutoLinkFilter,
) ([]*types.AutoLink, error) {
	return f.listResult, f.err
}

type svbSpaceStore struct {
	store.SpaceStore

	ids []int64
	err error
}

func (f *svbSpaceStore) GetAncestorIDs(context.Context, int64) ([]int64, error) {
	return f.ids, f.err
}

func svbI64(v int64) *int64 { return &v }

func svbStr(v string) *string { return &v }

func svbNewService(als *svbAutoLinkStore, ss *svbSpaceStore) *Service {
	return &Service{autoLinkStore: als, spaceStore: ss}
}

func TestSvbAutoLinkSanitize(t *testing.T) {
	tests := []struct {
		name    string
		in      AutoLinkInput
		wantErr string
	}{
		{"ok regex", AutoLinkInput{enum.AutoLinkTypeRegex, `ABC-\d+`, "https://example.com/x"}, ""},
		{"empty pattern", AutoLinkInput{enum.AutoLinkTypeRegex, "", "https://example.com"}, "pattern is required"},
		{
			"space in prefix", AutoLinkInput{enum.AutoLinkTypePrefixWithNumValue, "A B", "https://e.com/A B<num>"},
			"cannot contain spaces",
		},
		{"bad regex", AutoLinkInput{enum.AutoLinkTypeRegex, "(", "https://e.com"}, "invalid regex"},
		{"no url", AutoLinkInput{enum.AutoLinkTypeRegex, "a", ""}, "target URL is required"},
		{
			"long url", AutoLinkInput{enum.AutoLinkTypeRegex, "a", "https://e.com/" + string(make([]byte, 2001))},
			"cannot exceed",
		},
		{"bad url", AutoLinkInput{enum.AutoLinkTypeRegex, "a", "http://[::1"}, "invalid target URL"},
		{"no host", AutoLinkInput{enum.AutoLinkTypeRegex, "a", "https:///path"}, "non-empty host"},
		{"bad scheme", AutoLinkInput{enum.AutoLinkTypeRegex, "a", "ftp://example.com"}, "scheme"},
		{
			"prefix num missing placeholder",
			AutoLinkInput{enum.AutoLinkTypePrefixWithNumValue, "JIRA-", "https://e.com/browse/"}, "prefix_num",
		},
		{
			"prefix num ok",
			AutoLinkInput{enum.AutoLinkTypePrefixWithNumValue, "JIRA-", "https://e.com/browse/JIRA-<num>"}, "",
		},
		{
			"prefix alnum missing placeholder",
			AutoLinkInput{enum.AutoLinkTypePrefixWithAlphanumericValue, "REF-", "https://e.com/"}, "prefix_alphanumeric",
		},
		{
			"prefix alnum ok",
			AutoLinkInput{enum.AutoLinkTypePrefixWithAlphanumericValue, "REF-", "https://e.com/REF-<alphanum>"}, "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.in.Sanitize()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestSvbValidateURL(t *testing.T) {
	require.NoError(t, validateURL("https://example.com/x"))
	require.Error(t, validateURL("ftp://example.com"))
	require.Error(t, validateURL("https:///x"))
	require.Error(t, validateURL("http://[::1"))
	require.Error(t, validateURL("https://e.com/"+string(make([]byte, 2001))))
}

func TestSvbConvertPatternToRegex(t *testing.T) {
	require.Equal(t, `JIRA-([0-9]+)`, ConvertPatternToRegex(enum.AutoLinkTypePrefixWithNumValue, "JIRA-"))
	require.Equal(t, `A\.B([0-9]+)`, ConvertPatternToRegex(enum.AutoLinkTypePrefixWithNumValue, "A.B"))
	require.Equal(t, `REF-([a-zA-Z0-9]+)`, ConvertPatternToRegex(enum.AutoLinkTypePrefixWithAlphanumericValue, "REF-"))
	require.Equal(t, `a.*`, ConvertPatternToRegex(enum.AutoLinkTypeRegex, "a.*"))
}

func TestSvbCheckScope(t *testing.T) {
	al := &types.AutoLink{SpaceID: svbI64(1), RepoID: svbI64(2)}
	require.NoError(t, checkScope(al, Scope{RepoID: svbI64(2)}))
	require.ErrorIs(t, checkScope(al, Scope{RepoID: svbI64(3)}), usererror.ErrNotFound)
	require.NoError(t, checkScope(al, Scope{SpaceID: svbI64(1)}))
	require.ErrorIs(t, checkScope(al, Scope{SpaceID: svbI64(9)}), usererror.ErrNotFound)
	require.ErrorIs(t, checkScope(al, Scope{}), usererror.ErrNotFound)
	spaceOnly := &types.AutoLink{SpaceID: svbI64(1)}
	require.ErrorIs(t, checkScope(spaceOnly, Scope{RepoID: svbI64(1)}), usererror.ErrNotFound)
	repoOnly := &types.AutoLink{RepoID: svbI64(1)}
	require.ErrorIs(t, checkScope(repoOnly, Scope{SpaceID: svbI64(1)}), usererror.ErrNotFound)
}

func TestSvbAutoLinkCreate(t *testing.T) {
	ctx := context.Background()
	als := &svbAutoLinkStore{}
	svc := svbNewService(als, nil)

	got, err := svc.Create(ctx, 7, Scope{RepoID: svbI64(5)}, &AutoLinkInput{
		Type: enum.AutoLinkTypePrefixWithNumValue, Pattern: "JIRA-", TargetURL: "https://e.com/JIRA-<num>",
	})
	require.NoError(t, err)
	require.Same(t, als.created, got)
	require.Equal(t, "JIRA-([0-9]+)", got.Pattern)
	require.Equal(t, int64(7), got.CreatedBy)
	require.Equal(t, int64(7), got.UpdatedBy)
	require.Equal(t, int64(5), *got.RepoID)
	require.Nil(t, got.SpaceID)
	require.NotZero(t, got.Created)

	_, err = svc.Create(ctx, 1, Scope{}, &AutoLinkInput{})
	require.Error(t, err)

	als.err = errors.New("boom")
	_, err = svc.Create(ctx, 1, Scope{SpaceID: svbI64(1)}, &AutoLinkInput{
		Type: enum.AutoLinkTypeRegex, Pattern: "a", TargetURL: "https://e.com",
	})
	require.ErrorContains(t, err, "failed to create autolink")
}

func TestSvbAutoLinkFindDelete(t *testing.T) {
	ctx := context.Background()
	als := &svbAutoLinkStore{items: map[int64]*types.AutoLink{1: {ID: 1, SpaceID: svbI64(10)}}}
	svc := svbNewService(als, nil)

	got, err := svc.Find(ctx, Scope{SpaceID: svbI64(10)}, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.ID)

	_, err = svc.Find(ctx, Scope{SpaceID: svbI64(11)}, 1)
	require.ErrorIs(t, err, usererror.ErrNotFound)

	_, err = svc.Find(ctx, Scope{SpaceID: svbI64(10)}, 2)
	require.ErrorContains(t, err, "failed to get autolink")

	require.ErrorIs(t, svc.Delete(ctx, Scope{SpaceID: svbI64(11)}, 1), usererror.ErrNotFound)
	require.NoError(t, svc.Delete(ctx, Scope{SpaceID: svbI64(10)}, 1))
	require.Equal(t, int64(1), als.deletedID)

	// A store failure on delete (after a successful find) is wrapped.
	failing := &svbDeleteFailStore{svbAutoLinkStore: *als}
	require.ErrorContains(t, svbNewServiceWith(failing).Delete(ctx, Scope{SpaceID: svbI64(10)}, 1),
		"failed to delete autolink")
}

type svbDeleteFailStore struct {
	svbAutoLinkStore
}

func (f *svbDeleteFailStore) Delete(context.Context, int64) error { return errors.New("boom") }

func svbNewServiceWith(als store.AutoLinkStore) *Service { return &Service{autoLinkStore: als} }

func TestSvbAutoLinkList(t *testing.T) {
	ctx := context.Background()
	mk := func() []*types.AutoLink {
		return []*types.AutoLink{
			{Type: enum.AutoLinkTypePrefixWithNumValue, Pattern: "JIRA-"},
			{Type: enum.AutoLinkTypePrefixWithAlphanumericValue, Pattern: "REF-"},
			{Type: enum.AutoLinkTypeRegex, Pattern: "x"},
		}
	}

	t.Run("direct", func(t *testing.T) {
		als := &svbAutoLinkStore{listResult: mk()}
		got, count, err := svbNewService(als, nil).List(ctx, svbI64(1), nil, &types.AutoLinkFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(3), count)
		require.Equal(t, enum.AutoLinkTypeRegex, got[0].Type)
		require.Equal(t, "^JIRA-(\\d+)$", got[0].Pattern)
		require.Equal(t, "^REF-([\\w-]+)$", got[1].Pattern)
		require.Equal(t, "x", got[2].Pattern)
	})

	t.Run("direct store error", func(t *testing.T) {
		als := &svbAutoLinkStore{err: errors.New("boom")}
		_, _, err := svbNewService(als, nil).List(ctx, svbI64(1), nil, &types.AutoLinkFilter{})
		require.Error(t, err)
	})

	t.Run("inherited with repo", func(t *testing.T) {
		als := &svbAutoLinkStore{listResult: mk()}
		ss := &svbSpaceStore{ids: []int64{1, 2}}
		_, count, err := svbNewService(als, ss).List(
			ctx, svbI64(1), svbI64(9), &types.AutoLinkFilter{Inherited: true})
		require.NoError(t, err)
		require.Equal(t, int64(3), count)
		require.True(t, als.inScope)
		require.Equal(t, int64(9), als.gotRepoID)
		require.Equal(t, []int64{1, 2}, als.gotSpaceIDs)
	})

	t.Run("inherited space only", func(t *testing.T) {
		als := &svbAutoLinkStore{listResult: mk()}
		ss := &svbSpaceStore{ids: []int64{1}}
		_, _, err := svbNewService(als, ss).List(ctx, svbI64(1), nil, &types.AutoLinkFilter{Inherited: true})
		require.NoError(t, err)
		require.Zero(t, als.gotRepoID)
	})

	t.Run("inherited ancestor error", func(t *testing.T) {
		ss := &svbSpaceStore{err: errors.New("boom")}
		_, _, err := svbNewService(&svbAutoLinkStore{}, ss).List(
			ctx, svbI64(1), svbI64(2), &types.AutoLinkFilter{Inherited: true})
		require.Error(t, err)
		_, _, err = svbNewService(&svbAutoLinkStore{}, ss).List(
			ctx, svbI64(1), nil, &types.AutoLinkFilter{Inherited: true})
		require.Error(t, err)
	})

	t.Run("inherited store error", func(t *testing.T) {
		als := &svbAutoLinkStore{err: errors.New("boom")}
		ss := &svbSpaceStore{ids: []int64{1}}
		_, _, err := svbNewService(als, ss).List(ctx, svbI64(1), nil, &types.AutoLinkFilter{Inherited: true})
		require.Error(t, err)
	})
}

func TestSvbAutoLinkUpdate(t *testing.T) {
	ctx := context.Background()
	base := func() *svbAutoLinkStore {
		return &svbAutoLinkStore{items: map[int64]*types.AutoLink{
			1: {
				ID: 1, SpaceID: svbI64(10), Type: enum.AutoLinkTypeRegex,
				Pattern: "abc", TargetURL: "https://e.com/old",
			},
			2: {
				ID: 2, SpaceID: svbI64(10), Type: enum.AutoLinkTypePrefixWithNumValue,
				Pattern: "JIRA-([0-9]+)", TargetURL: "https://e.com/JIRA-<num>",
			},
		}}
	}
	scope := Scope{SpaceID: svbI64(10)}

	t.Run("not found", func(t *testing.T) {
		_, err := svbNewService(base(), nil).Update(ctx, scope, 99, 1, &AutoLinkUpdateInput{})
		require.ErrorContains(t, err, "failed to get autolink")
	})

	t.Run("wrong scope", func(t *testing.T) {
		_, err := svbNewService(base(), nil).Update(ctx, Scope{SpaceID: svbI64(1)}, 1, 1, &AutoLinkUpdateInput{})
		require.ErrorIs(t, err, usererror.ErrNotFound)
	})

	t.Run("update url and pattern", func(t *testing.T) {
		als := base()
		got, err := svbNewService(als, nil).Update(ctx, scope, 1, 5, &AutoLinkUpdateInput{
			Pattern: svbStr("new"), TargetURL: svbStr("https://e.com/new"),
		})
		require.NoError(t, err)
		require.Same(t, als.updated, got)
		require.Equal(t, "new", got.Pattern)
		require.Equal(t, "https://e.com/new", got.TargetURL)
		require.Equal(t, int64(5), got.UpdatedBy)
	})

	t.Run("invalid url", func(t *testing.T) {
		_, err := svbNewService(base(), nil).Update(ctx, scope, 1, 5, &AutoLinkUpdateInput{
			TargetURL: svbStr("ftp://e.com"),
		})
		require.Error(t, err)
	})

	t.Run("change type without pattern", func(t *testing.T) {
		tp := enum.AutoLinkTypePrefixWithNumValue
		_, err := svbNewService(base(), nil).Update(ctx, scope, 1, 5, &AutoLinkUpdateInput{Type: &tp})
		require.ErrorContains(t, err, "pattern must be provided")
	})

	t.Run("change type with pattern validates prefix url", func(t *testing.T) {
		tp := enum.AutoLinkTypePrefixWithNumValue
		_, err := svbNewService(base(), nil).Update(ctx, scope, 1, 5, &AutoLinkUpdateInput{
			Type: &tp, Pattern: svbStr("BUG-"),
		})
		require.ErrorContains(t, err, "prefix_num")

		als := base()
		got, err := svbNewService(als, nil).Update(ctx, scope, 1, 5, &AutoLinkUpdateInput{
			Type: &tp, Pattern: svbStr("BUG-"), TargetURL: svbStr("https://e.com/BUG-<num>"),
		})
		require.NoError(t, err)
		require.Equal(t, enum.AutoLinkTypePrefixWithNumValue, got.Type)
		require.Equal(t, "BUG-([0-9]+)", got.Pattern)
	})

	t.Run("prefix type keeps pattern when only url changes", func(t *testing.T) {
		als := base()
		got, err := svbNewService(als, nil).Update(ctx, scope, 2, 5, &AutoLinkUpdateInput{
			TargetURL: svbStr("https://other.com/x"),
		})
		require.NoError(t, err)
		require.Equal(t, "JIRA-([0-9]+)", got.Pattern)
		require.Equal(t, "https://other.com/x", got.TargetURL)
	})

	t.Run("store error", func(t *testing.T) {
		als := base()
		failing := &svbUpdateFailStore{svbAutoLinkStore: *als}
		_, err := svbNewServiceWith(failing).Update(ctx, scope, 1, 5, &AutoLinkUpdateInput{})
		require.ErrorContains(t, err, "failed to update autolink")
	})
}

type svbUpdateFailStore struct {
	svbAutoLinkStore
}

func (f *svbUpdateFailStore) Update(context.Context, *types.AutoLink) error {
	return errors.New("boom")
}

func TestSvbIsPrefixType(t *testing.T) {
	require.True(t, isPrefixType(enum.AutoLinkTypePrefixWithNumValue))
	require.True(t, isPrefixType(enum.AutoLinkTypePrefixWithAlphanumericValue))
	require.False(t, isPrefixType(enum.AutoLinkTypeRegex))
}

func TestSvbNewService(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	require.NotNil(t, svc)
}
