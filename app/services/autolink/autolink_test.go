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

package autolink

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type fakeAutoLinkStore struct {
	store.AutoLinkStore

	links  map[int64]*types.AutoLink
	nextID int64

	listCalls         []string
	gotRepoID         int64
	gotSpaceIDs       []int64
	err               error
	listInScopeResult []*types.AutoLink
}

func newFakeAutoLinkStore() *fakeAutoLinkStore {
	return &fakeAutoLinkStore{links: map[int64]*types.AutoLink{}, nextID: 1}
}

func (f *fakeAutoLinkStore) Create(_ context.Context, a *types.AutoLink) error {
	if f.err != nil {
		return f.err
	}
	a.ID = f.nextID
	f.nextID++
	f.links[a.ID] = a
	return nil
}

func (f *fakeAutoLinkStore) Update(_ context.Context, a *types.AutoLink) error {
	if f.err != nil {
		return f.err
	}
	f.links[a.ID] = a
	return nil
}

func (f *fakeAutoLinkStore) Find(_ context.Context, id int64) (*types.AutoLink, error) {
	if f.err != nil {
		return nil, f.err
	}
	a, ok := f.links[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return a, nil
}

func (f *fakeAutoLinkStore) Delete(_ context.Context, id int64) error {
	if f.err != nil {
		return f.err
	}
	delete(f.links, id)
	return nil
}

func (f *fakeAutoLinkStore) Count(context.Context, *int64, *int64, *types.AutoLinkFilter) (int64, error) {
	f.listCalls = append(f.listCalls, "count")
	return int64(len(f.links)), f.err
}

func (f *fakeAutoLinkStore) List(context.Context, *int64, *int64, *types.AutoLinkFilter) ([]*types.AutoLink, error) {
	f.listCalls = append(f.listCalls, "list")
	var out []*types.AutoLink
	for _, l := range f.links {
		out = append(out, l)
	}
	return out, f.err
}

func (f *fakeAutoLinkStore) CountInScopes(
	_ context.Context, repoID int64, spaceIDs []int64, _ *types.AutoLinkFilter,
) (int64, error) {
	f.listCalls = append(f.listCalls, "countInScopes")
	f.gotRepoID, f.gotSpaceIDs = repoID, spaceIDs
	return int64(len(f.listInScopeResult)), f.err
}

func (f *fakeAutoLinkStore) ListInScopes(
	context.Context, int64, []int64, *types.AutoLinkFilter,
) ([]*types.AutoLink, error) {
	f.listCalls = append(f.listCalls, "listInScopes")
	return f.listInScopeResult, f.err
}

type fakeSpaceStore struct {
	store.SpaceStore

	ancestors []int64
	err       error
}

func (f fakeSpaceStore) GetAncestorIDs(context.Context, int64) ([]int64, error) {
	return f.ancestors, f.err
}

func ptr[T any](v T) *T { return &v }

func newService() (*Service, *fakeAutoLinkStore, *fakeSpaceStore) {
	links := newFakeAutoLinkStore()
	spaces := &fakeSpaceStore{ancestors: []int64{1, 2}}
	return NewService(nil, spaces, nil, links), links, spaces
}

func TestAutoLinkInput_Sanitize(t *testing.T) {
	valid := func() AutoLinkInput {
		return AutoLinkInput{
			Type: enum.AutoLinkTypePrefixWithNumValue, Pattern: "JIRA-", TargetURL: "https://jira.example.com/JIRA-<num>",
		}
	}
	require.NoError(t, (func() error { in := valid(); return in.Sanitize() })())

	tests := []struct {
		name    string
		mutate  func(*AutoLinkInput)
		wantErr string
	}{
		{"empty pattern", func(in *AutoLinkInput) { in.Pattern = "" }, "pattern is required"},
		{"pattern with spaces", func(in *AutoLinkInput) { in.Pattern = "JIRA -" }, "cannot contain spaces"},
		{"alphanumeric pattern with spaces", func(in *AutoLinkInput) {
			in.Type = enum.AutoLinkTypePrefixWithAlphanumericValue
			in.Pattern = "A B"
		}, "cannot contain spaces"},
		{"invalid regex", func(in *AutoLinkInput) {
			in.Type = enum.AutoLinkTypeRegex
			in.Pattern = "(unclosed"
		}, "invalid regex"},
		{"empty target", func(in *AutoLinkInput) { in.TargetURL = "" }, "target URL is required"},
		{"target too long", func(in *AutoLinkInput) {
			in.TargetURL = "https://example.com/" + strings.Repeat("a", 2000)
		}, "cannot exceed"},
		{"target not parseable", func(in *AutoLinkInput) { in.TargetURL = "https://exa mple.com/%zz" }, "invalid target URL"},
		{"target without host", func(in *AutoLinkInput) { in.TargetURL = "https:///JIRA-<num>" }, "non-empty host"},
		{"target with bad scheme", func(in *AutoLinkInput) {
			in.TargetURL = "ftp://example.com/JIRA-<num>"
		}, "http or https"},
		{"num placeholder missing", func(in *AutoLinkInput) { in.TargetURL = "https://example.com/JIRA-" }, "prefix_num"},
		{"alphanum placeholder missing", func(in *AutoLinkInput) {
			in.Type = enum.AutoLinkTypePrefixWithAlphanumericValue
			in.TargetURL = "https://example.com/JIRA-<num>"
		}, "prefix_alphanumeric"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid()
			tt.mutate(&in)
			err := in.Sanitize()
			require.Error(t, err)
			var userErr *usererror.Error
			require.ErrorAs(t, err, &userErr)
			require.Contains(t, userErr.Message, tt.wantErr)
		})
	}

	t.Run("regex type needs no placeholder", func(t *testing.T) {
		in := AutoLinkInput{Type: enum.AutoLinkTypeRegex, Pattern: `[A-Z]+-\d+`, TargetURL: "http://example.com/x"}
		require.NoError(t, in.Sanitize())
	})
}

func TestConvertPatternToRegex(t *testing.T) {
	require.Equal(t, `JIRA-([0-9]+)`, ConvertPatternToRegex(enum.AutoLinkTypePrefixWithNumValue, "JIRA-"))
	require.Equal(t, `REF-([a-zA-Z0-9]+)`, ConvertPatternToRegex(enum.AutoLinkTypePrefixWithAlphanumericValue, "REF-"))
	require.Equal(t, `a\.b\+([0-9]+)`, ConvertPatternToRegex(enum.AutoLinkTypePrefixWithNumValue, "a.b+"))
	require.Equal(t, `[A-Z]+-\d+`, ConvertPatternToRegex(enum.AutoLinkTypeRegex, `[A-Z]+-\d+`))
}

func TestValidateURL(t *testing.T) {
	require.NoError(t, validateURL("https://example.com/path"))
	require.Error(t, validateURL("ftp://example.com"))
	require.Error(t, validateURL("https:///nohost"))
	require.Error(t, validateURL("https://exa mple.com/%zz"))
	require.Error(t, validateURL("https://example.com/"+strings.Repeat("a", 2000)))
}

func TestCheckScope(t *testing.T) {
	repoScoped := &types.AutoLink{RepoID: ptr(int64(5))}
	spaceScoped := &types.AutoLink{SpaceID: ptr(int64(9))}

	require.NoError(t, checkScope(repoScoped, Scope{RepoID: ptr(int64(5))}))
	require.ErrorIs(t, checkScope(repoScoped, Scope{RepoID: ptr(int64(6))}), usererror.ErrNotFound)
	require.ErrorIs(t, checkScope(spaceScoped, Scope{RepoID: ptr(int64(5))}), usererror.ErrNotFound)

	require.NoError(t, checkScope(spaceScoped, Scope{SpaceID: ptr(int64(9))}))
	require.ErrorIs(t, checkScope(spaceScoped, Scope{SpaceID: ptr(int64(1))}), usererror.ErrNotFound)
	require.ErrorIs(t, checkScope(repoScoped, Scope{SpaceID: ptr(int64(9))}), usererror.ErrNotFound)

	require.ErrorIs(t, checkScope(repoScoped, Scope{}), usererror.ErrNotFound)
}

func TestService_Create(t *testing.T) {
	svc, links, _ := newService()

	created, err := svc.Create(t.Context(), 42, Scope{RepoID: ptr(int64(5))}, &AutoLinkInput{
		Type: enum.AutoLinkTypePrefixWithNumValue, Pattern: "JIRA-", TargetURL: "https://jira.example.com/JIRA-<num>",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), created.ID)
	require.Equal(t, `JIRA-([0-9]+)`, created.Pattern, "prefix patterns are stored as regular expressions")
	require.Equal(t, int64(42), created.CreatedBy)
	require.Equal(t, int64(42), created.UpdatedBy)
	require.Equal(t, int64(5), *created.RepoID)
	require.Nil(t, created.SpaceID)
	require.NotZero(t, created.Created)
	require.Equal(t, created.Created, created.Updated)
	require.Contains(t, links.links, int64(1))

	t.Run("invalid input is rejected before storing", func(t *testing.T) {
		before := len(links.links)
		_, err := svc.Create(t.Context(), 42, Scope{RepoID: ptr(int64(5))}, &AutoLinkInput{})
		require.Error(t, err)
		require.Len(t, links.links, before)
	})

	t.Run("store error", func(t *testing.T) {
		links.err = errors.New("db down")
		defer func() { links.err = nil }()
		_, err := svc.Create(t.Context(), 42, Scope{SpaceID: ptr(int64(1))}, &AutoLinkInput{
			Type: enum.AutoLinkTypeRegex, Pattern: `x+`, TargetURL: "https://example.com",
		})
		require.ErrorIs(t, err, links.err)
	})
}

func TestService_FindAndDelete(t *testing.T) {
	svc, links, _ := newService()
	links.links[1] = &types.AutoLink{ID: 1, RepoID: ptr(int64(5))}
	repoScope := Scope{RepoID: ptr(int64(5))}

	got, err := svc.Find(t.Context(), repoScope, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.ID)

	_, err = svc.Find(t.Context(), Scope{RepoID: ptr(int64(6))}, 1)
	require.ErrorIs(t, err, usererror.ErrNotFound)

	_, err = svc.Find(t.Context(), repoScope, 99)
	require.Error(t, err)

	require.ErrorIs(t, svc.Delete(t.Context(), Scope{RepoID: ptr(int64(6))}, 1), usererror.ErrNotFound)
	require.Contains(t, links.links, int64(1))

	require.Error(t, svc.Delete(t.Context(), repoScope, 99))

	require.NoError(t, svc.Delete(t.Context(), repoScope, 1))
	require.NotContains(t, links.links, int64(1))

	links.links[2] = &types.AutoLink{ID: 2, RepoID: ptr(int64(5))}
	links.err = errors.New("db down")
	// the lookup fails first
	require.Error(t, svc.Delete(t.Context(), repoScope, 2))
}

func TestService_Update(t *testing.T) {
	newLink := func(typ enum.AutoLinkType, pattern, target string) (*Service, *fakeAutoLinkStore, *types.AutoLink) {
		svc, links, _ := newService()
		link := &types.AutoLink{
			ID: 1, RepoID: ptr(int64(5)), Type: typ, Pattern: pattern, TargetURL: target,
		}
		links.links[1] = link
		return svc, links, link
	}
	scope := Scope{RepoID: ptr(int64(5))}

	t.Run("only the target url", func(t *testing.T) {
		svc, _, _ := newLink(enum.AutoLinkTypeRegex, `x+`, "https://old.example.com")
		got, err := svc.Update(t.Context(), scope, 1, 7, &AutoLinkUpdateInput{TargetURL: ptr("https://new.example.com")})
		require.NoError(t, err)
		require.Equal(t, "https://new.example.com", got.TargetURL)
		require.Equal(t, `x+`, got.Pattern)
		require.Equal(t, int64(7), got.UpdatedBy)
	})

	t.Run("new prefix pattern is converted", func(t *testing.T) {
		svc, _, _ := newLink(enum.AutoLinkTypePrefixWithNumValue, `JIRA-([0-9]+)`, "https://x.example.com/JIRA-<num>")
		got, err := svc.Update(t.Context(), scope, 1, 7, &AutoLinkUpdateInput{
			Pattern: ptr("BUG-"), TargetURL: ptr("https://x.example.com/BUG-<num>"),
		})
		require.NoError(t, err)
		require.Equal(t, `BUG-([0-9]+)`, got.Pattern)
	})

	t.Run("changing a prefix type requires a pattern", func(t *testing.T) {
		svc, _, _ := newLink(enum.AutoLinkTypePrefixWithNumValue, `JIRA-([0-9]+)`, "https://x.example.com/JIRA-<num>")
		_, err := svc.Update(t.Context(), scope, 1, 7, &AutoLinkUpdateInput{
			Type: ptr(enum.AutoLinkTypePrefixWithAlphanumericValue),
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "pattern must be provided")
	})

	t.Run("prefix url must keep the placeholder", func(t *testing.T) {
		svc, _, _ := newLink(enum.AutoLinkTypePrefixWithNumValue, `JIRA-([0-9]+)`, "https://x.example.com/JIRA-<num>")
		_, err := svc.Update(t.Context(), scope, 1, 7, &AutoLinkUpdateInput{
			Pattern: ptr("JIRA-"), TargetURL: ptr("https://x.example.com/no-placeholder"),
		})
		require.Error(t, err)
	})

	t.Run("invalid url", func(t *testing.T) {
		svc, _, _ := newLink(enum.AutoLinkTypeRegex, `x+`, "https://old.example.com")
		_, err := svc.Update(t.Context(), scope, 1, 7, &AutoLinkUpdateInput{TargetURL: ptr("ftp://nope.example.com")})
		require.Error(t, err)
	})

	t.Run("empty values are ignored", func(t *testing.T) {
		svc, _, link := newLink(enum.AutoLinkTypeRegex, `x+`, "https://old.example.com")
		got, err := svc.Update(t.Context(), scope, 1, 7, &AutoLinkUpdateInput{Pattern: ptr(""), TargetURL: ptr("")})
		require.NoError(t, err)
		require.Equal(t, link.TargetURL, got.TargetURL)
		require.Equal(t, `x+`, got.Pattern)
	})

	t.Run("type only for regex links", func(t *testing.T) {
		svc, _, _ := newLink(enum.AutoLinkTypePrefixWithNumValue, `JIRA-([0-9]+)`, "https://x.example.com/JIRA-<num>")
		got, err := svc.Update(t.Context(), scope, 1, 7, &AutoLinkUpdateInput{Type: ptr(enum.AutoLinkTypePrefixWithNumValue)})
		require.NoError(t, err)
		require.Equal(t, enum.AutoLinkTypePrefixWithNumValue, got.Type)
	})

	t.Run("wrong scope", func(t *testing.T) {
		svc, _, _ := newLink(enum.AutoLinkTypeRegex, `x+`, "https://old.example.com")
		_, err := svc.Update(t.Context(), Scope{RepoID: ptr(int64(6))}, 1, 7, &AutoLinkUpdateInput{})
		require.ErrorIs(t, err, usererror.ErrNotFound)
	})

	t.Run("unknown link and store errors", func(t *testing.T) {
		svc, links, _ := newLink(enum.AutoLinkTypeRegex, `x+`, "https://old.example.com")
		_, err := svc.Update(t.Context(), scope, 99, 7, &AutoLinkUpdateInput{})
		require.Error(t, err)

		boom := errors.New("write failed")
		links.err = nil
		failing := &failOnUpdate{fakeAutoLinkStore: links, err: boom}
		svc.autoLinkStore = failing
		_, err = svc.Update(t.Context(), scope, 1, 7, &AutoLinkUpdateInput{TargetURL: ptr("https://new.example.com")})
		require.ErrorIs(t, err, boom)
	})
}

type failOnUpdate struct {
	*fakeAutoLinkStore
	err error
}

func (f *failOnUpdate) Update(context.Context, *types.AutoLink) error { return f.err }

func TestIsPrefixType(t *testing.T) {
	require.True(t, isPrefixType(enum.AutoLinkTypePrefixWithNumValue))
	require.True(t, isPrefixType(enum.AutoLinkTypePrefixWithAlphanumericValue))
	require.False(t, isPrefixType(enum.AutoLinkTypeRegex))
}

func TestService_List(t *testing.T) {
	t.Run("plain list rewrites prefix links to regular expressions", func(t *testing.T) {
		svc, links, _ := newService()
		links.links[1] = &types.AutoLink{ID: 1, Type: enum.AutoLinkTypePrefixWithNumValue, Pattern: `JIRA-`}
		links.links[2] = &types.AutoLink{ID: 2, Type: enum.AutoLinkTypePrefixWithAlphanumericValue, Pattern: `REF-`}
		links.links[3] = &types.AutoLink{ID: 3, Type: enum.AutoLinkTypeRegex, Pattern: `x+`}

		got, count, err := svc.List(t.Context(), ptr(int64(1)), nil, &types.AutoLinkFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(3), count)
		require.Equal(t, []string{"count", "list"}, links.listCalls)

		byID := map[int64]*types.AutoLink{}
		for _, l := range got {
			byID[l.ID] = l
		}
		require.Equal(t, "^JIRA-(\\d+)$", byID[1].Pattern)
		require.Equal(t, enum.AutoLinkTypeRegex, byID[1].Type)
		require.Equal(t, "^REF-([\\w-]+)$", byID[2].Pattern)
		require.Equal(t, enum.AutoLinkTypeRegex, byID[2].Type)
		require.Equal(t, `x+`, byID[3].Pattern)
	})

	t.Run("inherited list uses the ancestor spaces", func(t *testing.T) {
		svc, links, _ := newService()
		links.listInScopeResult = []*types.AutoLink{{ID: 1, Type: enum.AutoLinkTypeRegex, Pattern: `x`}}

		filter := &types.AutoLinkFilter{Inherited: true}
		_, count, err := svc.List(t.Context(), ptr(int64(10)), ptr(int64(5)), filter)
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
		require.Equal(t, []string{"countInScopes", "listInScopes"}, links.listCalls)
		require.Equal(t, int64(5), links.gotRepoID)
		require.Equal(t, []int64{1, 2}, links.gotSpaceIDs)

		links.listCalls = nil
		_, _, err = svc.List(t.Context(), ptr(int64(10)), nil, filter)
		require.NoError(t, err)
		require.Equal(t, int64(0), links.gotRepoID, "space level lists have no repo id")
	})

	t.Run("errors", func(t *testing.T) {
		boom := errors.New("boom")

		svc, links, spaces := newService()
		links.err = boom
		_, _, err := svc.List(t.Context(), ptr(int64(1)), nil, &types.AutoLinkFilter{})
		require.ErrorIs(t, err, boom)

		links.err = nil
		spaces.err = boom
		_, _, err = svc.List(t.Context(), ptr(int64(1)), ptr(int64(2)), &types.AutoLinkFilter{Inherited: true})
		require.ErrorIs(t, err, boom)
		_, _, err = svc.List(t.Context(), ptr(int64(1)), nil, &types.AutoLinkFilter{Inherited: true})
		require.ErrorIs(t, err, boom)

		spaces.err = nil
		links.err = boom
		_, _, err = svc.List(t.Context(), ptr(int64(1)), nil, &types.AutoLinkFilter{Inherited: true})
		require.ErrorIs(t, err, boom)
	})
}
